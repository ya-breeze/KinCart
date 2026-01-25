package database

import (
	"log/slog"
	"os"

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
	)
	if err != nil {
		slog.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("Database initialized and migrated")

	seedFromEnv()
}

func seedFromEnv() {
	familyName := os.Getenv("KINCART_SEED_FAMILY")
	username := os.Getenv("KINCART_SEED_USER")
	password := os.Getenv("KINCART_SEED_PASS")

	if familyName == "" || username == "" || password == "" {
		return
	}

	var family models.Family
	if err := DB.Where("name = ?", familyName).First(&family).Error; err != nil {
		family = models.Family{Name: familyName}
		if err := DB.Create(&family).Error; err != nil {
			slog.Error("Failed to seed family from env", "error", err)
			return
		}
		slog.Info("Seeded family from env", "family_name", familyName)
	}

	var user models.User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("Failed to hash seed user password", "error", err)
			return
		}
		user = models.User{
			Username:     username,
			PasswordHash: string(hash),
			FamilyID:     family.ID,
		}
		if err := DB.Create(&user).Error; err != nil {
			slog.Error("Failed to seed user from env", "error", err)
			return
		}
		slog.Info("Seeded user from env", "username", username)
	}
}
