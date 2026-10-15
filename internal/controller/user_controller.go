package controller

import (
	"net/http"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userRepo domain.UserRepository
}

func NewUserController(userRepo domain.UserRepository) *UserController {
	return &UserController{userRepo: userRepo}
}

type DeviceTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func (u *UserController) SaveDeviceToken(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: token is required"})
		return
	}

	if err := u.userRepo.UpdateDeviceToken(c.Request.Context(), userID, &req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save device token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "device token saved successfully",
	})
}

func (u *UserController) DeleteDeviceToken(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := u.userRepo.UpdateDeviceToken(c.Request.Context(), userID, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "device token deleted successfully",
	})
}
