package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type HealthController struct{ ping func(context.Context) error }

func NewHealthController(pings ...func(context.Context) error) *HealthController {
	h := &HealthController{}
	if len(pings) > 0 {
		h.ping = pings[0]
	}
	return h
}
func (h *HealthController) CheckHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "bikko-backend"})
}
func (h *HealthController) CheckReady(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if h.ping == nil || h.ping(ctx) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "database": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "connected"})
}
