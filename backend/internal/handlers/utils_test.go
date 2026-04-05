package handlers

import (
	"testing"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/google/uuid"
	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateItemsFamily(t *testing.T) {
	// Setup in-memory DB
	var err error
	database.DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.DB.AutoMigrate(&models.Category{}, &models.Family{})

	// Seed data
	family1 := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family 1"}}
	database.DB.Create(&family1)
	family2 := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family 2"}}
	database.DB.Create(&family2)

	cat1 := models.Category{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family1.ID},
		Name:        "Cat 1",
	}
	database.DB.Create(&cat1)

	tests := []struct {
		name     string
		items    []models.Item
		familyID uuid.UUID
		wantErr  bool
	}{
		{
			name: "valid category for family",
			items: []models.Item{
				{CategoryID: cat1.ID},
			},
			familyID: family1.ID,
			wantErr:  false,
		},
		{
			name: "invalid category for family",
			items: []models.Item{
				{CategoryID: cat1.ID},
			},
			familyID: family2.ID,
			wantErr:  true,
		},
		{
			name: "no category (zero UUID) is valid",
			items: []models.Item{
				{CategoryID: uuid.Nil},
			},
			familyID: family1.ID,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateItemsFamily(tt.items, tt.familyID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
