package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides secure CORS handling with origin validation
func CORSMiddleware() gin.HandlerFunc {
	// Get allowed origins from environment variable
	// Format: ALLOWED_ORIGINS=https://example.com,https://app.example.com
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")

	// Default allowed origins for development
	// In production, ALLOWED_ORIGINS must be set
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:80",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:80",
	}

	// Parse additional allowed origins from environment
	if allowedOriginsEnv != "" {
		envOrigins := strings.Split(allowedOriginsEnv, ",")
		for _, origin := range envOrigins {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	// Create a map for O(1) lookup
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is in the allowed list
		if origin != "" && allowedOriginsMap[origin] {
			// Only set CORS headers for allowed origins
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if origin != "" && allowedOriginsMap[origin] {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				// Reject preflight from unauthorized origins
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}

		c.Next()
	}
}
