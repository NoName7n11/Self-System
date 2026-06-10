package http

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitState struct {
	windowStart time.Time
	count       int
}

func MutationRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 120
	}
	if window <= 0 {
		window = time.Minute
	}

	var mu sync.Mutex
	entries := map[string]rateLimitState{}

	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-Forwarded-For"))
		if key == "" {
			key = strings.TrimSpace(c.ClientIP())
		}
		if key == "" {
			key = "unknown"
		}

		if subject, ok := c.Get("auth.subject"); ok {
			trimmed := strings.TrimSpace(subject.(string))
			if trimmed != "" {
				key = "sub:" + trimmed
			}
		}

		now := time.Now().UTC()

		mu.Lock()
		entry := entries[key]
		if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= window {
			entry = rateLimitState{windowStart: now, count: 0}
		}
		entry.count++
		entries[key] = entry
		allowed := entry.count <= limit
		mu.Unlock()

		if !allowed {
			respondErrorCode(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}
