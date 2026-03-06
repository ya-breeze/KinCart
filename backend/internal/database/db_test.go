package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kincart/internal/models"
)

func TestInitDB_SearchTextBackfill(t *testing.T) {
	// 1. Setup in-memory db manually to insert some data BEFORE InitDB runs
	dsn := "file:backfill_test?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Explicitly migrate the required model
	err = db.AutoMigrate(&models.FlyerItem{})
	assert.NoError(t, err)

	// Clear just in case
	db.Exec("DELETE FROM flyer_items")

	// Insert test data using raw SQL to force SearchText to specific values (including NULL)
	// NULL search_text
	db.Exec("INSERT INTO flyer_items (name, categories, keywords, search_text) VALUES (?, ?, ?, NULL)", "Milk", "dairy", "milk")
	// Empty string search_text
	db.Exec("INSERT INTO flyer_items (name, categories, keywords, search_text) VALUES (?, ?, ?, ?)", "Bread", "bakery", "bread", "")
	// Already populated search_text
	db.Exec("INSERT INTO flyer_items (name, categories, keywords, search_text) VALUES (?, ?, ?, ?)", "Eggs", "dairy", "eggs", "already_set")

	// 2. Run InitDB pointing to the same memory DB
	_ = os.Setenv("DB_PATH", dsn)
	defer func() { _ = os.Unsetenv("DB_PATH") }()

	// Since InitDB uses the global DB variable, we can just call it
	InitDB()

	// 3. Verify
	var items []models.FlyerItem
	DB.Find(&items)

	assert.Equal(t, 3, len(items))

	for _, item := range items {
		switch item.Name {
		case "Milk":
			assert.Equal(t, "milk dairy milk", item.SearchText, "NULL search text should be backfilled")
		case "Bread":
			assert.Equal(t, "bread bakery bread", item.SearchText, "Empty string search text should be backfilled")
		case "Eggs":
			assert.Equal(t, "already_set", item.SearchText, "Already populated search text should not be changed")
		}
	}
}
