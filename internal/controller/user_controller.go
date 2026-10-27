package controller

import (
	"errors"
	"io"
	"net/http"
	"strings"

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

	var err error
	if devices, ok := u.userRepo.(domain.UserDeviceRepository); ok {
		appType := strings.ToLower(c.GetHeader("X-App-Type"))
		if appType == "" {
			appType = "client"
		}
		if appType != "client" && appType != "provider" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application type"})
			return
		}
		err = devices.RegisterDevice(c.Request.Context(), userID, appType, req.Token)
	} else {
		err = u.userRepo.UpdateDeviceToken(c.Request.Context(), userID, &req.Token)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save device token"})
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

	var err error
	if devices, ok := u.userRepo.(domain.UserDeviceRepository); ok {
		appType := strings.ToLower(c.GetHeader("X-App-Type"))
		if appType == "" {
			appType = "client"
		}
		if appType != "client" && appType != "provider" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application type"})
			return
		}
		var req struct {
			Token string `json:"token"`
		}
		if c.Request.ContentLength != 0 {
			if bindErr := c.ShouldBindJSON(&req); bindErr != nil && !errors.Is(bindErr, io.EOF) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token payload"})
				return
			}
		}
		err = devices.DeleteDevice(c.Request.Context(), userID, appType, req.Token)
	} else {
		err = u.userRepo.UpdateDeviceToken(c.Request.Context(), userID, nil)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "device token deleted successfully",
	})
}
