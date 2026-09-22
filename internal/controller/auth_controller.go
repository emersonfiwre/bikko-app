package controller

import (
	"errors"
	"net/http"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
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

		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "INVALID_CREDENTIALS",
			"message": "Falha ao realizar login. Verifique suas credenciais.",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
