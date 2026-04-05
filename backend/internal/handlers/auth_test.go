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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/ya-breeze/kin-core/auth"
	"github.com/ya-breeze/kin-core/authdb"
	coremodels "github.com/ya-breeze/kin-core/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var err error
	database.DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = database.DB.AutoMigrate(&models.Family{}, &models.User{}, &authdb.RefreshToken{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create a test user
	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
	database.DB.Create(&family)

	password := "password123"
	hash, _ := auth.HashPassword(password)
	user := models.User{
		User: coremodels.User{
			ID:           uuid.New(),
			Username:     "testuser",
			PasswordHash: hash,
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
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				// Cookies should be set
				cookies := w.Result().Cookies()
				var hasAccess, hasRefresh bool
				for _, c := range cookies {
					if c.Name == "kin_access" {
						hasAccess = true
					}
					if c.Name == "kin_refresh" {
						hasRefresh = true
					}
				}
				assert.True(t, hasAccess, "kin_access cookie should be set")
				assert.True(t, hasRefresh, "kin_refresh cookie should be set")
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var err error
	database.DB, err = gorm.Open(sqlite.Open("file:testrefresh?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.DB.AutoMigrate(&models.Family{}, &models.User{}, &authdb.RefreshToken{})

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFamily"}}
	database.DB.Create(&family)

	user := models.User{User: coremodels.User{
		ID:       uuid.New(),
		Username: "testuser",
		FamilyID: family.ID,
	}}
	database.DB.Create(&user)

	// Create a valid refresh token
	validToken := authdb.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IsRevoked: false,
	}
	database.DB.Create(&validToken)

	// Create an expired refresh token
	expiredToken := authdb.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-time.Hour),
		IsRevoked: false,
	}
	database.DB.Create(&expiredToken)

	// Create a revoked refresh token
	revokedToken := authdb.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     "revoked-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IsRevoked: true,
	}
	database.DB.Create(&revokedToken)

	tests := []struct {
		name           string
		cookieValue    string
		expectedStatus int
	}{
		{
			name:           "successful refresh",
			cookieValue:    "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "expired token",
			cookieValue:    "expired-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "revoked token",
			cookieValue:    "revoked-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "non-existent token",
			cookieValue:    "non-existent",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/refresh", Refresh)

			req, _ := http.NewRequest(http.MethodPost, "/refresh", nil)
			req.AddCookie(&http.Cookie{Name: "kin_refresh", Value: tt.cookieValue})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				cookies := w.Result().Cookies()
				var hasAccess bool
				for _, c := range cookies {
					if c.Name == "kin_access" {
						hasAccess = true
					}
				}
				assert.True(t, hasAccess, "kin_access cookie should be set on refresh")
			}
		})
	}
}
