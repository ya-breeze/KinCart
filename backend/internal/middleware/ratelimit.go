package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// LoginRateLimiter returns a middleware that limits the number of login attempts to 5 per minute per IP.
func LoginRateLimiter() gin.HandlerFunc {
	// Define the rate (5 requests per minute)
	rate := limiter.Rate{
		Limit:  5,
		Period: 60, // in seconds
	}

	// Use an in-memory store for rate limiting
	store := memory.NewStore()

	// Create a new limiter instance
	instance := limiter.New(store, rate)

	return func(c *gin.Context) {
		// Use client IP as the key for rate limiting
		key := c.ClientIP()

		context, err := instance.Get(c, key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal rate limiter error"})
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		// Check if the rate limit has been reached
		if context.Reached {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many login attempts. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
