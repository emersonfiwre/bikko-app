package controller

import (
	"net/http"
	"os"

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
	// [SECURITY AUDIT] ATENÇÃO: O fallback para c.Query("userId") é mantido exclusivamente
	// para facilitar testes unitários. Em ambiente de Release/Produção, deve ser removido,
	// exigindo estritamente autenticação via JWT (middleware.GetUserID).
	isProduction := os.Getenv("ENV") == "production" || gin.Mode() == gin.ReleaseMode
	if userID == "" && !isProduction {
		userID = c.Query("userId")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	resp, err := p.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (p *ProfileController) DeleteAccount(c *gin.Context) {
	// Instead of query param, use the authenticated user ID injected by middleware
	userID := middleware.GetUserID(c)
	// [SECURITY AUDIT] ATENÇÃO: O fallback para c.Query("userId") é mantido exclusivamente
	// para facilitar testes unitários. Em ambiente de Release/Produção, deve ser removido,
	// exigindo estritamente autenticação via JWT (middleware.GetUserID).
	isProduction := os.Getenv("ENV") == "production" || gin.Mode() == gin.ReleaseMode
	if userID == "" && !isProduction {
		userID = c.Query("userId")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := p.profileService.DeleteAccount(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao desativar conta: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Conta desativada com sucesso",
	})
}

type UpdateProfileRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Phone string `json:"phone"`
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

	if err := p.profileService.UpdateProfile(c.Request.Context(), userID, req.Name, req.Email, req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile updated successfully",
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile photo: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"photo_url": req.PhotoURL,
	})
}
