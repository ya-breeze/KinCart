package database

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"

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

	// Handle SQLite NOT NULL column migration for existing tables
	if DB.Dialector.Name() == "sqlite" {
		tables := []string{"users", "items", "categories", "shops", "shopping_lists", "receipts", "item_frequencies"}
		for _, table := range tables {
			if DB.Migrator().HasTable(table) && !DB.Migrator().HasColumn(table, "family_id") {
				slog.Info("Adding family_id column with default value to existing table", "table", table)
				// Add column with default 1 to satisfy NOT NULL on existing rows
				err = DB.Exec(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN family_id INTEGER NOT NULL DEFAULT 1", table)).Error
				if err != nil {
					slog.Warn("Failed to manually add family_id column", "table", table, "error", err)
				}
			}
		}
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
		&models.FlyerPage{},
		&models.FlyerItem{},
		&models.JobStatus{},
		&models.Receipt{},
		&models.ReceiptItem{},
	)
	if err != nil {
		slog.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("Database initialized and migrated")

	seedFromEnv()
	seedFlyersFromEnv()
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
			family = models.Family{Family: coremodels.Family{Name: familyName}}
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
				User: coremodels.User{
					Username:     username,
					PasswordHash: string(hash),
					FamilyID:     family.ID,
				},
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
			user.User.PasswordHash = string(hash)
			user.User.FamilyID = family.ID
			if err := DB.Save(&user).Error; err != nil {
				slog.Error("Failed to update seed user from env", "username", username, "error", err)
				continue
			}
			slog.Info("Updated seed user from env", "username", username, "family", familyName)
		}
	}
}

func seedFlyersFromEnv() {
	seedFlyers := os.Getenv("KINCART_SEED_FLYERS")
	if seedFlyers == "" {
		return
	}

	// Format: ShopName:Item1|Price1,Item2|Price2;ShopName2:Item3|Price3
	flyerBlocks := strings.Split(seedFlyers, ";")
	for _, block := range flyerBlocks {
		parts := strings.Split(strings.TrimSpace(block), ":")
		if len(parts) < 2 {
			continue
		}

		shopName := strings.TrimSpace(parts[0])
		itemsStr := parts[1]

		// Create Flyer
		now := time.Now()
		flyer := models.Flyer{
			ShopName:  shopName,
			StartDate: now.AddDate(0, 0, -7),
			EndDate:   now.AddDate(0, 0, 7),
			ParsedAt:  now,
		}

		if err := DB.Create(&flyer).Error; err != nil {
			slog.Error("Failed to seed flyer", "shop", shopName, "error", err)
			continue
		}

		// Create a mock page
		page := models.FlyerPage{
			FlyerID:   flyer.ID,
			IsParsed:  true,
			LocalPath: "placeholder.jpg",
		}
		DB.Create(&page)

		// Parse Items
		itemParts := strings.Split(itemsStr, ",")
		for _, itemPart := range itemParts {
			details := strings.Split(strings.TrimSpace(itemPart), "|")
			itemName := details[0]
			price := 0.0
			if len(details) > 1 {
				price, _ = strconv.ParseFloat(details[1], 64)
			}

			flyerItem := models.FlyerItem{
				FlyerID:     flyer.ID,
				FlyerPageID: page.ID,
				Name:        itemName,
				Price:       price,
				StartDate:   flyer.StartDate,
				EndDate:     flyer.EndDate,
				ShopName:    shopName,
			}

			if err := DB.Create(&flyerItem).Error; err != nil {
				slog.Error("Failed to seed flyer item", "item", itemName, "error", err)
			}
		}
		slog.Info("Seeded flyer and items", "shop", shopName, "count", len(itemParts))
	}
}
