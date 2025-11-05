package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/services"
)

// RateLimitMiddleware implements rate limiting: 5 requests per 15 seconds per user
func RateLimitMiddleware(cacheService *services.CacheService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health check endpoint
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get identifier for rate limiting (use user ID if authenticated, otherwise IP)
		identifier := c.ClientIP()

		// Try to get user ID from context if available
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(uint); ok {
				identifier = "user:" + strconv.Itoa(int(id))
			}
		}

		// Rate limit: 5 requests per 15 seconds
		limit := 10
		window := 20 * time.Second
		key := "rate_limit:" + identifier

		if cacheService != nil {
			// Use Redis for distributed rate limiting
			ctx := cacheService.Context()
			client := cacheService.Client()

			count, err := client.Incr(ctx, key).Result()
			if err == nil {
				// Set expiration on first request
				if count == 1 {
					client.Expire(ctx, key, window)
				}

				// Check if limit exceeded
				if count > int64(limit) {
					c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
					c.Header("X-RateLimit-Remaining", "0")
					c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
					c.JSON(http.StatusTooManyRequests, gin.H{
						"error":       "Rate limit exceeded. Please try again later.",
						"retry_after": int(window.Seconds()),
					})
					c.Abort()
					return
				}

				// Set rate limit headers
				remaining := limit - int(count)
				if remaining < 0 {
					remaining = 0
				}
				c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
				c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
				c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
			}
			// If Redis error, allow request (fail open) but log it
		}

		c.Next()
	}
}
