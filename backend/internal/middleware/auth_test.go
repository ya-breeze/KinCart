package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupHeader    func(req *http.Request)
		expectedStatus int
	}{
		{
			name: "missing header",
			setupHeader: func(req *http.Request) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid format",
			setupHeader: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidToken")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token",
			setupHeader: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "valid token",
			setupHeader: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":   float64(1),
					"family_id": float64(1),
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString(JWTSecret)
				req.Header.Set("Authorization", "Bearer "+tokenString)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired token",
			setupHeader: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":   float64(1),
					"family_id": float64(1),
					"exp":       time.Now().Add(-time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString(JWTSecret)
				req.Header.Set("Authorization", "Bearer "+tokenString)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "blacklisted token",
			setupHeader: func(req *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":   float64(1),
					"family_id": float64(1),
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString(JWTSecret)
				BlacklistToken(tokenString, time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+tokenString)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware())
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			tt.setupHeader(req)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
