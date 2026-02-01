package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	database.DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = database.DB.AutoMigrate(&models.Family{}, &models.User{})
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
		})
	}
}
