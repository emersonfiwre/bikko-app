package controller

import (
	"errors"
	"net/http"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/service"

	"github.com/gin-gonic/gin"
)

type ProfileController struct {
	profileService service.ProfileService
}

func NewProfileController(profileService service.ProfileService) *ProfileController {
	return &ProfileController{profileService: profileService}
}

func (p *ProfileController) GetProfile(c *gin.Context) {
	// Instead of query param, use the authenticated user ID injected by middleware
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	resp, err := p.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "profile service unavailable"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (p *ProfileController) DeleteAccount(c *gin.Context) {
	// Instead of query param, use the authenticated user ID injected by middleware
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := p.profileService.DeleteAccount(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao desativar conta"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Conta desativada com sucesso",
	})
}

type UpdateProfileRequest struct {
	Name          string `json:"name" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Phone         string `json:"phone"`
	FirebaseToken string `json:"firebase_token"`
}

func (p *ProfileController) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	newToken, err := p.profileService.UpdateProfileWithToken(c.Request.Context(), userID, req.Name, req.Email, req.Phone, req.FirebaseToken)
	if err != nil {
		if errors.Is(err, service.ErrPhoneChangeRequiresFirebaseToken) || errors.Is(err, service.ErrInvalidFirebaseToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "FIREBASE_TOKEN_INVALID",
				"message": "Autenticação do novo número via Firebase é obrigatória",
			})
			return
		}
		if errors.Is(err, service.ErrPhoneAlreadyInUse) {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "PHONE_ALREADY_IN_USE",
				"message": "Este número de telefone já está vinculado a outra conta",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	resp := gin.H{
		"status":  "success",
		"message": "Profile updated successfully",
	}
	if newToken != "" {
		resp["token"] = newToken
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateProfilePhotoRequest receives the public Cloudflare R2 URL of an image
// previously uploaded by the client through a pre-signed PUT URL.
type UpdateProfilePhotoRequest struct {
	PhotoURL string `json:"photo_url" binding:"required,url"`
}

func (p *ProfileController) UpdateProfilePhoto(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfilePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	if err := p.profileService.UpdateProfilePhoto(c.Request.Context(), userID, req.PhotoURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"photo_url": req.PhotoURL,
	})
}
