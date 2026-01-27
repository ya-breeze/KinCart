package database

import (
	"log/slog"
	"os"
	"strings"

	"kincart/internal/models"

	"golang.org/x/crypto/bcrypt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/kincart.db"
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Auto Migration
	err = DB.AutoMigrate(
		&models.Family{},
		&models.User{},
		&models.ShoppingList{},
		&models.Item{},
		&models.Category{},
		&models.Shop{},
		&models.ShopCategoryOrder{},
		&models.ItemFrequency{},
		&models.Flyer{},
		&models.FlyerItem{},
	)
	if err != nil {
		slog.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("Database initialized and migrated")

	seedFromEnv()
}

func seedFromEnv() {
	seedUsers := os.Getenv("KINCART_SEED_USERS")
	if seedUsers == "" {
		return
	}

	// Parse comma-separated list of family:username:password triplets
	triplets := strings.Split(seedUsers, ",")
	for _, triplet := range triplets {
		parts := strings.Split(strings.TrimSpace(triplet), ":")
		if len(parts) != 3 {
			slog.Warn("Invalid seed user format, expected family:username:password", "triplet", triplet)
			continue
		}

		familyName := strings.TrimSpace(parts[0])
		username := strings.TrimSpace(parts[1])
		password := strings.TrimSpace(parts[2])

		if familyName == "" || username == "" || password == "" {
			slog.Warn("Empty family, username, or password in seed triplet", "triplet", triplet)
			continue
		}

		// Find or create family
		var family models.Family
		if err := DB.Where("name = ?", familyName).First(&family).Error; err != nil {
			family = models.Family{Name: familyName}
			if err := DB.Create(&family).Error; err != nil {
				slog.Error("Failed to seed family from env", "family", familyName, "error", err)
				continue
			}
			slog.Info("Seeded family from env", "family_name", familyName)
		}

		// Find or create user
		var user models.User
		if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
			// User doesn't exist, create new
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				slog.Error("Failed to hash seed user password", "username", username, "error", err)
				continue
			}
			user = models.User{
				Username:     username,
				PasswordHash: string(hash),
				FamilyID:     family.ID,
			}
			if err := DB.Create(&user).Error; err != nil {
				slog.Error("Failed to seed user from env", "username", username, "error", err)
				continue
			}
			slog.Info("Seeded user from env", "username", username, "family", familyName)
		} else {
			// User exists, update password and family
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				slog.Error("Failed to hash seed user password", "username", username, "error", err)
				continue
			}
			user.PasswordHash = string(hash)
			user.FamilyID = family.ID
			if err := DB.Save(&user).Error; err != nil {
				slog.Error("Failed to update seed user from env", "username", username, "error", err)
				continue
			}
			slog.Info("Updated seed user from env", "username", username, "family", familyName)
		}
	}
}
