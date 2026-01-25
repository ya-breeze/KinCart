package middleware

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// UploadSecurityMiddleware adds security headers to prevent XSS and other attacks
// when serving uploaded files
func UploadSecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent rendering in iframes
		c.Header("X-Frame-Options", "DENY")

		// Strict CSP to prevent script execution
		c.Header("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox")

		// Get file extension
		ext := strings.ToLower(filepath.Ext(c.Request.URL.Path))

		// Set appropriate Content-Type and force download for potentially dangerous files
		switch ext {
		case ".jpg", ".jpeg":
			c.Header("Content-Type", "image/jpeg")
		case ".png":
			c.Header("Content-Type", "image/png")
		case ".webp":
			c.Header("Content-Type", "image/webp")
		case ".svg":
			// SVG can contain scripts, force download
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", "attachment")
		default:
			// Unknown file type, force download
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", "attachment")
		}

		c.Next()
	}
}
