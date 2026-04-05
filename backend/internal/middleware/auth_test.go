package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/ya-breeze/kin-core/auth"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	familyID := uuid.New()

	makeValidToken := func() string {
		token, _ := auth.GenerateAccessToken(userID, &familyID, JWTSecret, time.Hour)
		return token
	}

	makeExpiredToken := func() string {
		// Use negative duration — GenerateAccessToken sets exp = now + duration
		token, _ := auth.GenerateAccessToken(userID, &familyID, JWTSecret, -time.Hour)
		return token
	}

	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		expectedStatus int
	}{
		{
			name:           "missing cookie",
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token in cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "kin_access", Value: "invalid.token"})
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "valid token",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "kin_access", Value: makeValidToken()})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired token",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "kin_access", Value: makeExpiredToken()})
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			// Pass nil DB — blacklist check is skipped when DB is nil
			r.Use(AuthMiddleware(nil))
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
