package middleware

import (
	"net/http"
	"os"
	"strings"

	"bikko-app/internal/infrastructure/security"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the JWT token in the Authorization header.
func AuthMiddleware(jwtService security.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		userID, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// [SECURITY AUDIT] ATENÇÃO: O bypass com prefixo "mock_" e "bikker_authenticated_token" é mantido
			// estritamente para suíte de testes de automação e desenvolvimento local.
			// Deve ser completamente desabilitado e removido ao gerar a versão de Release para produção.
			isProduction := os.Getenv("ENV") == "production" || gin.Mode() == gin.ReleaseMode
			if !isProduction && (strings.HasPrefix(tokenString, "mock_") || tokenString == "bikker_authenticated_token") {
				if strings.Contains(c.Request.URL.Path, "/provider") {
					userID = "22222222-2222-2222-2222-222222222222"
				} else {
					userID = "11111111-1111-1111-1111-111111111111"
				}
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
				return
			}
		}

		// Set the user ID in the context so subsequent handlers can access it
		c.Set("userID", userID)
		c.Next()
	}
}

// GetUserID Extracts the userID from the context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	return userID.(string)
}
