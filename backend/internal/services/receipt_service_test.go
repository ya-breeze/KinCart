package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kincart/internal/ai"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockParser implements ReceiptParser
type MockParser struct {
	ParseFunc     func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
	ParseTextFunc func(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error)
}

func (m *MockParser) ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, imagePath, knownItems)
	}
	return nil, nil
}

func (m *MockParser) ParseReceiptText(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error) {
	if m.ParseTextFunc != nil {
		return m.ParseTextFunc(ctx, receiptText, knownItems)
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
	family := models.Family{Family: coremodels.Family{Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		Title:       "Weekly",
	}
	db.Create(&list)

	item1 := models.Item{Name: "Milk", ListID: list.ID, IsBought: false}
	db.Create(&item1)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   "test.jpg",
		Status:      "new",
	}
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
	family := models.Family{Family: coremodels.Family{Name: "TestFam"}}
	db.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		Title:       "List",
	}
	db.Create(&list)

	receipt1 := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   "r1.jpg",
		Status:      "new",
	}
	receipt2 := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   "r2.jpg",
		Status:      "parsed",
	}
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

// TestProcessReceipt_TextFile verifies that .txt receipts are routed to ParseReceiptText.
func TestProcessReceipt_TextFile(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		Title:       "Weekly",
	}
	db.Create(&list)

	// Write a real .txt file to the temp directory
	textContent := "Store: Lidl\nMilk 1,99\nBread 2,49\nTotal 4,48"
	relPath := filepath.Join("families", "1", "receipts", "2024", "01", "test_receipt.txt")
	fullPath := filepath.Join(tmpDir, relPath)
	assert.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	assert.NoError(t, os.WriteFile(fullPath, []byte(textContent), 0644))

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   relPath,
		Status:      "new",
	}
	db.Create(&receipt)

	var parseTextCalled bool
	var receivedText string
	mock := &MockParser{
		ParseTextFunc: func(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error) {
			parseTextCalled = true
			receivedText = receiptText
			return &ai.ParsedReceipt{
				StoreName: "Lidl",
				Date:      "2024-01-30",
				Total:     4.48,
				Items: []ai.ParsedReceiptItem{
					{Name: "Milk", Price: 1.99, Quantity: 1, TotalPrice: 1.99},
				},
			}, nil
		},
	}

	svc := NewReceiptService(db, mock, nil, tmpDir)
	err := svc.ProcessReceipt(context.Background(), receipt.ID, list.ID)

	assert.NoError(t, err)
	assert.True(t, parseTextCalled, "ParseReceiptText should be called for .txt files")
	assert.Equal(t, textContent, receivedText)

	var updated models.Receipt
	db.First(&updated, receipt.ID)
	assert.Equal(t, "parsed", updated.Status)
}

// TestCreateReceiptFromText verifies text is saved to filesystem and receipt record created.
func TestCreateReceiptFromText(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{Name: "TestFam"}}
	db.Create(&family)

	storage := NewFileStorageService(tmpDir)
	svc := NewReceiptService(db, nil, storage, tmpDir)

	text := "Store: TestMart\nTotal: 10.50\nMilk 2.50\nBread 8.00"
	receipt, err := svc.CreateReceiptFromText(family.ID, text)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.NotZero(t, receipt.ID)
	assert.True(t, strings.HasSuffix(receipt.ImagePath, ".txt"), "ImagePath should end with .txt")

	// Verify file was written with correct content
	fullPath := filepath.Join(tmpDir, receipt.ImagePath)
	content, readErr := os.ReadFile(fullPath)
	assert.NoError(t, readErr)
	assert.Equal(t, text, string(content))

	// Verify DB record exists
	var dbReceipt models.Receipt
	db.First(&dbReceipt, receipt.ID)
	assert.Equal(t, receipt.ImagePath, dbReceipt.ImagePath)
	assert.Equal(t, family.ID, dbReceipt.FamilyID)
}

// TestProcessReceipt_TextFile_ParseError verifies error path sets status to "error".
func TestProcessReceipt_TextFile_ParseError(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		Title:       "List",
	}
	db.Create(&list)

	relPath := filepath.Join("families", "1", "receipts", "2024", "01", "bad.txt")
	fullPath := filepath.Join(tmpDir, relPath)
	assert.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	assert.NoError(t, os.WriteFile(fullPath, []byte("garbage text"), 0644))

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   relPath,
		Status:      "new",
	}
	db.Create(&receipt)

	mock := &MockParser{
		ParseTextFunc: func(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error) {
			return nil, assert.AnError
		},
	}

	svc := NewReceiptService(db, mock, nil, tmpDir)
	err := svc.ProcessReceipt(context.Background(), receipt.ID, list.ID)

	assert.Error(t, err)

	var updated models.Receipt
	db.First(&updated, receipt.ID)
	assert.Equal(t, "error", updated.Status)
}
