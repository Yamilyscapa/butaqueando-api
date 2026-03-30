package middleware

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type authRateBucket struct {
	count       int
	windowStart time.Time
}

var (
	authRateLimitMu sync.Mutex
	authRateBuckets = map[string]authRateBucket{}
)

func AuthRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 || window <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		now := time.Now().UTC()
		key := c.ClientIP() + ":" + c.FullPath()

		authRateLimitMu.Lock()
		bucket := authRateBuckets[key]
		if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= window {
			bucket = authRateBucket{count: 1, windowStart: now}
		} else {
			bucket.count++
		}

		authRateBuckets[key] = bucket
		remaining := window - now.Sub(bucket.windowStart)
		overLimit := bucket.count > limit
		authRateLimitMu.Unlock()

		if overLimit {
			retryAfterSeconds := int(math.Ceil(remaining.Seconds()))
			if retryAfterSeconds < 0 {
				retryAfterSeconds = 0
			}

			httpx.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", gin.H{
				"limit":             limit,
				"windowSeconds":     int(window.Seconds()),
				"retryAfterSeconds": retryAfterSeconds,
			})
			return
		}

		c.Next()
	}
}
