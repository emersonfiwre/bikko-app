package middleware

import (
	"log/slog"
	"time"

	"bikko-app/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// InitStructuredLogger configures the default slog logger to emit structured JSON to stdout.
func InitStructuredLogger() {
	logger.Init()
}

// SlogMiddleware logs every HTTP request as a structured JSON log entry with correlation_id, handler, latency, and status.
func SlogMiddleware() gin.HandlerFunc {
	InitStructuredLogger()

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
		status := c.Writer.Status()
		handlerName := c.FullPath()
		if handlerName == "" {
			handlerName = c.Request.URL.Path
		}

		attrs := []any{
			slog.String("correlation_id", GetCorrelationID(c)),
			slog.String("handler", handlerName),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Float64("elapsed_ms", elapsedMs),
			slog.String("user_id", GetUserID(c)),
			slog.String("client_ip", c.ClientIP()),
		}

		if errStr := c.Errors.String(); errStr != "" {
			attrs = append(attrs, slog.String("error", errStr))
		}

		if status >= 500 {
			slog.Error("http_request", attrs...)
		} else {
			slog.Info("http_request", attrs...)
		}
	}
}
