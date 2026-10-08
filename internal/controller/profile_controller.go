package controller

import (
	"net/http"

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
		userID = c.Query("userId") // Fallback if no auth token is provided during transition
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
	if userID == "" {
		userID = c.Query("userId")
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

