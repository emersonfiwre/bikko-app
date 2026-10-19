package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type correlationIDKey struct{}

const (
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderRequestID     = "X-Request-ID"
	ContextCorrelationKey = "correlation_id"
)

// WithCorrelationID returns a new context with the given correlation ID attached.
func WithCorrelationID(ctx context.Context, corrID string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, corrID)
}

// CorrelationIDFromContext extracts the correlation ID from a standard context.Context.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return val
	}
	return ""
}

// CorrelationIDMiddleware extracts or generates a Correlation ID for distributed tracing.
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := strings.TrimSpace(c.GetHeader(HeaderCorrelationID))
		if corrID == "" {
			corrID = strings.TrimSpace(c.GetHeader(HeaderRequestID))
		}
		if corrID == "" {
			corrID = uuid.New().String()
		}

		c.Set(ContextCorrelationKey, corrID)
		c.Writer.Header().Set(HeaderCorrelationID, corrID)
		c.Request = c.Request.WithContext(WithCorrelationID(c.Request.Context(), corrID))

		c.Next()
	}
}

// GetCorrelationID returns the correlation ID from the Gin context.
func GetCorrelationID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if val, exists := c.Get(ContextCorrelationKey); exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	if c.Request != nil {
		return CorrelationIDFromContext(c.Request.Context())
	}
	return ""
}
