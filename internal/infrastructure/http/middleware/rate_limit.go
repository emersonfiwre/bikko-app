package middleware

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
	"time"
)

type ipEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}
type IPRateLimiter struct {
	mu        sync.Mutex
	ips       map[string]ipEntry
	r         rate.Limit
	b         int
	lastSweep time.Time
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{ips: map[string]ipEntry{}, r: r, b: b, lastSweep: time.Now()}
}
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()
	if now.Sub(i.lastSweep) > time.Minute {
		for k, e := range i.ips {
			if now.Sub(e.seen) > 10*time.Minute {
				delete(i.ips, k)
			}
		}
		i.lastSweep = now
	}
	e, ok := i.ips[ip]
	if !ok {
		if len(i.ips) >= 10000 {
			return rate.NewLimiter(0, 0)
		}
		e = ipEntry{limiter: rate.NewLimiter(i.r, i.b)}
	}
	e.seen = now
	i.ips[ip] = e
	return e.limiter
}
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter { return i.GetLimiter(ip) }
func (i *IPRateLimiter) Cleanup()                      {} // Entries expire lazily; no unowned background goroutine.
func RateLimitMiddleware(i *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !i.GetLimiter(c.ClientIP()).Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "RATE_LIMITED"})
			return
		}
		c.Next()
	}
}
