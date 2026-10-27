package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
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

const maxTrackedIPs = 20000

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips:       make(map[string]ipEntry),
		r:         r,
		b:         b,
		lastSweep: time.Now(),
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()

	// Sweep entries inactive for more than 5 minutes every minute
	if now.Sub(i.lastSweep) > time.Minute {
		for k, e := range i.ips {
			if now.Sub(e.seen) > 5*time.Minute {
				delete(i.ips, k)
			}
		}
		i.lastSweep = now
	}

	e, ok := i.ips[ip]
	if !ok {
		// Defensive memory cap: if threshold reached, evict the oldest entry instead of rejecting with 0 rate
		if len(i.ips) >= maxTrackedIPs {
			var oldestKey string
			var oldestTime time.Time
			for k, entry := range i.ips {
				if oldestKey == "" || entry.seen.Before(oldestTime) {
					oldestKey = k
					oldestTime = entry.seen
				}
			}
			if oldestKey != "" {
				delete(i.ips, oldestKey)
			}
		}
		e = ipEntry{limiter: rate.NewLimiter(i.r, i.b)}
	}
	e.seen = now
	i.ips[ip] = e
	return e.limiter
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter { return i.GetLimiter(ip) }
func (i *IPRateLimiter) Cleanup()                      {} // Entries expire lazily and deterministically

func RateLimitMiddleware(i *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !i.GetLimiter(c.ClientIP()).Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMITED",
				"error":   "RATE_LIMITED",
				"message": "Muitas requisições. Aguarde um instante e tente novamente.",
			})
			return
		}
		c.Next()
	}
}
