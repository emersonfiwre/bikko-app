package controller

import (
	"bikko-app/internal/domain"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxIdempotencyKeyLength = 128

// requiredIdempotencyKey keeps mutation retries deterministic. The key is scoped
// to the authenticated actor in persistence, so it must not be accepted from a
// request body where it could be accidentally replayed or overwritten.
func requiredIdempotencyKey(c *gin.Context) (string, bool) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" || len(key) > maxIdempotencyKeyLength {
		contractError(c, domain.ErrInvalidInput)
		return "", false
	}
	return key, true
}
