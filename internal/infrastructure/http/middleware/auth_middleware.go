package middleware

import (
	"errors"
	"net/http"
	"strings"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware accepts only signed sessions. Passing the identity repository also
// checks account status on every request, so soft deletion revokes existing sessions.
func AuthMiddleware(jwtService security.JWTService, repositories ...domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "valid Bearer session required"})
			return
		}
		userID, err := jwtService.ValidateToken(parts[1])
		if err != nil || strings.TrimSpace(userID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}
		if len(repositories) > 0 && repositories[0] != nil {
			user, err := repositories[0].GetByID(c.Request.Context(), userID)
			if errors.Is(err, domain.ErrNotFound) || (err == nil && user == nil) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account is inactive"})
				return
			}
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "identity service unavailable"})
				return
			}
			isBikker := user.IsBikker && user.Bikker != nil
			c.Set("isBikker", isBikker)
			// Availability controls discovery/dispatch, not the account's role. A
			// provider must be able to reach profile settings while offline and turn
			// availability back on.
			c.Set("isProviderAvailable", isBikker && user.Bikker.IsActive)
		}
		c.Set("userID", userID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	id, _ := c.Get("userID")
	userID, _ := id.(string)
	return userID
}

func RequireProvider() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool("isBikker") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "active provider account required"})
			return
		}
		c.Next()
	}
}
