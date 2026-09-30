package controller

import (
	"log"
	"net/http"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/cache"

	"github.com/gin-gonic/gin"
)

type BootstrapController struct {
	redisService *cache.RedisService
}

func NewBootstrapController(redisService *cache.RedisService) *BootstrapController {
	return &BootstrapController{redisService: redisService}
}

func (b *BootstrapController) GetBootstrap(c *gin.Context) {
	appVersion := c.GetHeader("X-App-Version")
	platform := c.GetHeader("X-Platform")
	log.Printf("[BootstrapController] Header received: X-App-Version='%s', X-Platform='%s'", appVersion, platform)

	var features map[string]bool
	if b.redisService != nil {
		var err error
		features, err = b.redisService.GetFeatures(c.Request.Context())
		if err != nil {
			log.Printf("[BootstrapController] Error fetching features: %v", err)
		}
	}

	if features == nil {
		features = map[string]bool{
			"enable_profile_reviews":     false,
			"enable_quote_counter_offer": true,
			"enable_instant_chat":        false,
			"enable_pix_direct_payment":  true,
			"enable_dark_mode_beta":      false,
		}
	}

	c.JSON(http.StatusOK, domain.BootstrapResponse{
		Features:           features,
		MinRequiredVersion: "1.0.0",
	})
}
