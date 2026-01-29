package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUploadSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		path                string
		expectedContentType string
		expectedDisposition string
	}{
		{
			name:                "JPEG image",
			path:                "/uploads/test.jpg",
			expectedContentType: "image/jpeg",
			expectedDisposition: "",
		},
		{
			name:                "PNG image",
			path:                "/uploads/test.png",
			expectedContentType: "image/png",
			expectedDisposition: "",
		},
		{
			name:                "SVG image (security risk)",
			path:                "/uploads/test.svg",
			expectedContentType: "application/octet-stream",
			expectedDisposition: "attachment",
		},
		{
			name:                "Unknown file type",
			path:                "/uploads/test.exe",
			expectedContentType: "application/octet-stream",
			expectedDisposition: "attachment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(UploadSecurityMiddleware())
			r.GET("/uploads/*path", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
			assert.Equal(t, tt.expectedContentType, w.Header().Get("Content-Type"))
			assert.Equal(t, tt.expectedDisposition, w.Header().Get("Content-Disposition"))
		})
	}
}
