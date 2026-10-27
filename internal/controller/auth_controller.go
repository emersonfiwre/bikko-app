package controller

import (
	"errors"
	"net/http"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"bikko-app/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService service.AuthService
}

func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (a *AuthController) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados de registro inválidos: " + err.Error()})
		return
	}

	resp, err := a.authService.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidFirebaseToken) || errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "message": "Verificação de telefone obrigatória"})
			return
		}
		if errors.Is(err, service.ErrUserAlreadyExists) || errors.Is(err, domain.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{
				"code":    "USER_ALREADY_EXISTS",
				"error":   "user_already_exists",
				"message": "Você já possui uma conta cadastrada. Por favor, realize o login.",
			})
			return
		}

		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Dados de cadastro inválidos"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "REGISTRATION_FAILED", "message": "Cadastro temporariamente indisponível"})
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (a *AuthController) CheckPhone(c *gin.Context) {
	var req model.CheckPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Dados inválidos: " + err.Error(),
		})
		return
	}

	resp, err := a.authService.CheckPhone(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Telefone inválido ou não fornecido.",
			})
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "AUTH_UNAVAILABLE", "message": "Autenticação temporariamente indisponível"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (a *AuthController) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Dados de login inválidos: " + err.Error(),
		})
		return
	}

	resp, err := a.authService.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUnconfirmedEmail) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "UNCONFIRMED_EMAIL",
				"message": "Por favor, verifique seu e-mail para confirmar sua conta antes de entrar.",
			})
			return
		}
		if errors.Is(err, service.ErrClientNeedsBikkerRegistration) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "CLIENT_NEEDS_BIKKER_REGISTRATION",
				"message": "Este número possui apenas cadastro de Cliente. Cadastre suas informações de Bikker para continuar.",
			})
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "Usuário não encontrado. Realize o cadastro.",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidFirebaseToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_OTP",
				"message": "Token ou código de verificação inválido ou expirado.",
			})
			return
		}

		if !errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "AUTH_UNAVAILABLE", "message": "Autenticação temporariamente indisponível"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "INVALID_CREDENTIALS",
			"message": "Falha ao realizar login. Verifique suas credenciais.",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
