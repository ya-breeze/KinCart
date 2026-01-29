package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kincart/internal/models"
)

func TestSeedFromEnv(t *testing.T) {
	// Setup strictly for this test
	// Use in-memory DB for speed and isolation
	var err error
	DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migration
	err = DB.AutoMigrate(
		&models.Family{},
		&models.User{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	tests := []struct {
		name      string
		envValue  string
		wantUsers map[string]string // username -> familyName
	}{
		{
			name:     "Single family, multiple users",
			envValue: "Smith:dad:pass1,Smith:mom:pass2,Smith:kid:pass3",
			wantUsers: map[string]string{
				"dad": "Smith",
				"mom": "Smith",
				"kid": "Smith",
			},
		},
		{
			name:     "Multiple families",
			envValue: "Jones:alice:passA,Brown:bob:passB",
			wantUsers: map[string]string{
				"alice": "Jones",
				"bob":   "Brown",
			},
		},
		{
			name:     "Invalid format ignored",
			envValue: "InvalidEntry,Smith:valid:pass",
			wantUsers: map[string]string{
				"valid": "Smith",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean tables before each run
			DB.Exec("DELETE FROM users")
			DB.Exec("DELETE FROM families")

			_ = os.Setenv("KINCART_SEED_USERS", tt.envValue)
			// Call the private function directly since we are in the same package
			seedFromEnv()

			for username, wantFamily := range tt.wantUsers {
				var user models.User
				result := DB.Preload("Family").Where("username = ?", username).First(&user)
				assert.NoError(t, result.Error, "User %s should exist", username)
				assert.Equal(t, wantFamily, user.Family.Name, "User %s should belong to family %s", username, wantFamily)
			}
		})
	}
}
