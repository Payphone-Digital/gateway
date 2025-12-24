// middleware/rate_limit.go
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RateLimiter struct {
	tokens     map[string][]time.Time
	maxRequest int
	duration   time.Duration
	mu         sync.Mutex
}

func NewRateLimiter(maxRequest int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     make(map[string][]time.Time),
		maxRequest: maxRequest,
		duration:   duration,
	}
}

func (rl *RateLimiter) cleanup(now time.Time) {
	for ip, tokens := range rl.tokens {
		var valid []time.Time
		for _, t := range tokens {
			if now.Sub(t) <= rl.duration {
				valid = append(valid, t)
			}
		}
		if len(valid) > 0 {
			rl.tokens[ip] = valid
		} else {
			delete(rl.tokens, ip)
		}
	}
}

func RateLimit(maxRequest int, duration time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(maxRequest, duration)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		// Cleanup old tokens
		limiter.cleanup(now)

		// Get tokens for IP
		tokens := limiter.tokens[ip]

		// Check if rate limit exceeded
		if len(tokens) >= maxRequest {
			logger.GetLogger().Warn("Rate limit exceeded",
				zap.String("client_ip", ip),
				zap.String("user_agent", c.GetHeader("User-Agent")),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("current_requests", len(tokens)),
				zap.Int("max_requests", maxRequest),
				zap.Duration("duration", duration),
				zap.Time("retry_after", now.Add(duration)),
			)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": duration.Seconds(),
			})
			c.Abort()
			return
		}

		// Add new token
		limiter.tokens[ip] = append(tokens, now)

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequest))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", maxRequest-len(tokens)-1))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(duration).Unix()))

		c.Next()
	}
}
