package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kincart/internal/database"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup in-memory DB
	var err error
	database.DB, err = gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = database.DB.AutoMigrate(&models.Family{}, &models.User{}, &models.RefreshToken{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create a test user
	family := models.Family{Family: coremodels.Family{Name: "Test Family"}}
	database.DB.Create(&family)

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.User{
		User: coremodels.User{
			Username:     "testuser",
			PasswordHash: string(hash),
			FamilyID:     family.ID,
		},
	}
	database.DB.Create(&user)

	tests := []struct {
		name           string
		request        LoginRequest
		expectedStatus int
	}{
		{
			name: "successful login",
			request: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid password",
			request: LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			request: LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/login", Login)

			jsonValue, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NotEmpty(t, resp["token"])
				assert.NotEmpty(t, resp["refresh_token"])
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup in-memory DB
	var err error
	database.DB, err = gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.DB.AutoMigrate(&models.User{}, &models.RefreshToken{})

	user := models.User{User: coremodels.User{Username: "testuser"}}
	database.DB.Create(&user)

	// Create a valid refresh token
	validToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IsRevoked: false,
	}
	database.DB.Create(&validToken)

	// Create an expired refresh token
	expiredToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-time.Hour),
		IsRevoked: false,
	}
	database.DB.Create(&expiredToken)

	// Create a revoked refresh token
	revokedToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     "revoked-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IsRevoked: true,
	}
	database.DB.Create(&revokedToken)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "successful refresh",
			token:          "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "expired token",
			token:          "expired-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "revoked token",
			token:          "revoked-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "non-existent token",
			token:          "non-existent",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/refresh", Refresh)

			reqBody := map[string]string{"refresh_token": tt.token}
			jsonValue, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NotEmpty(t, resp["token"])
			}
		})
	}
}
