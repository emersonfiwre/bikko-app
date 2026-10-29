package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware sets standard HTTP security headers on all responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME-sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		// Enable XSS filter in blocking mode
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		// Restrict referrer information
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Restrict features and framing
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")

		c.Next()
	}
}
