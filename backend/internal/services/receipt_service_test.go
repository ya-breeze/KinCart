package services

import (
	"context"
	"strings"
	"testing"

	"kincart/internal/ai"
	"kincart/internal/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockParser implements ReceiptParser
type MockParser struct {
	ParseFunc func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
}

func (m *MockParser) ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, imagePath, knownItems)
	}
	return nil, nil
}

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Receipt{}, &models.ReceiptItem{}, &models.ItemFrequency{}, &models.Category{}, &models.Shop{})
	return db
}

func TestProcessReceipt_Success(t *testing.T) {
	db := setupTestDB()

	// Setup Data
	family := models.Family{Name: "TestFam"}
	db.Create(&family)

	list := models.ShoppingList{Title: "Weekly", FamilyID: family.ID}
	db.Create(&list)

	item1 := models.Item{Name: "Milk", ListID: list.ID, IsBought: false}
	db.Create(&item1)

	receipt := models.Receipt{FamilyID: family.ID, ListID: &list.ID, ImagePath: "test.jpg", Status: "new"}
	db.Create(&receipt)

	// Setup Mock
	mock := &MockParser{
		ParseFunc: func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error) {
			return &ai.ParsedReceipt{
				StoreName: "SuperMart",
				Date:      "2024-01-30",
				Total:     10.5,
				Items: []ai.ParsedReceiptItem{
					{Name: "Milk", Price: 2.5, Quantity: 1, TotalPrice: 2.5},
					{Name: "Bread", Price: 8.0, Quantity: 1, TotalPrice: 8.0}, // New item
				},
			}, nil
		},
	}

	svc := NewReceiptService(db, mock, nil, "/tmp")

	// Act
	err := svc.ProcessReceipt(context.Background(), receipt.ID, list.ID)

	// Assert
	assert.NoError(t, err)

	// Check Receipt Status
	var updatedReceipt models.Receipt
	db.First(&updatedReceipt, receipt.ID)
	assert.Equal(t, "parsed", updatedReceipt.Status)
	assert.NotZero(t, updatedReceipt.ShopID)
	assert.Equal(t, 10.5, updatedReceipt.Total)

	// Check Items Linked
	var updatedItem1 models.Item
	db.First(&updatedItem1, item1.ID)
	assert.True(t, updatedItem1.IsBought)
	assert.Equal(t, 2.5, updatedItem1.Price)
	assert.NotNil(t, updatedItem1.ReceiptItemID)
	assert.Equal(t, 1.0, updatedItem1.Quantity) // Quantity should be updated

	// Check New Item Created
	var newItem models.Item
	db.Where("name = ?", "Bread").First(&newItem)
	assert.NotZero(t, newItem.ID)
	assert.Equal(t, list.ID, newItem.ListID)
	assert.True(t, newItem.IsBought)
	assert.NotNil(t, newItem.ReceiptItemID)

	// Check List Title Updated
	var updatedList models.ShoppingList
	db.First(&updatedList, list.ID)
	assert.True(t, strings.Contains(updatedList.Title, "(2024-01-30)"))

	// Check ActualAmount updated (2.5 + 8.0 = 10.5)
	assert.Equal(t, 10.5, updatedList.ActualAmount)
}

func TestProcessPendingReceipts(t *testing.T) {
	db := setupTestDB()

	// Setup
	family := models.Family{Name: "TestFam"}
	db.Create(&family)
	list := models.ShoppingList{Title: "List", FamilyID: family.ID}
	db.Create(&list)

	receipt1 := models.Receipt{FamilyID: family.ID, ListID: &list.ID, ImagePath: "r1.jpg", Status: "new"}
	receipt2 := models.Receipt{FamilyID: family.ID, ListID: &list.ID, ImagePath: "r2.jpg", Status: "parsed"}
	db.Create(&receipt1)
	db.Create(&receipt2)

	mock := &MockParser{
		ParseFunc: func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error) {
			return &ai.ParsedReceipt{Date: "2024-01-30"}, nil
		},
	}

	svc := NewReceiptService(db, mock, nil, "/tmp")

	// Act
	err := svc.ProcessPendingReceipts(context.Background())
	assert.NoError(t, err)

	// Assert
	var r1 models.Receipt
	db.First(&r1, receipt1.ID)
	assert.Equal(t, "parsed", r1.Status)
}
