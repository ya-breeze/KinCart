package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kincart/internal/ai"
	"kincart/internal/models"

	"github.com/google/uuid"
	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockParser implements ReceiptParser
type MockParser struct {
	ParseFunc      func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
	ParseTextFunc  func(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error)
	MatchItemsFunc func(ctx context.Context, receiptItems []string, plannedItems []string) (*ai.MatchResult, error)
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

func (m *MockParser) MatchReceiptItems(ctx context.Context, receiptItems []string, plannedItems []string) (*ai.MatchResult, error) {
	if m.MatchItemsFunc != nil {
		return m.MatchItemsFunc(ctx, receiptItems, plannedItems)
	}
	return &ai.MatchResult{}, nil
}

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Receipt{}, &models.ReceiptItem{}, &models.ItemFrequency{}, &models.Category{}, &models.Shop{}, &models.ItemAlias{})
	return db
}

func TestProcessReceipt_Success(t *testing.T) {
	db := setupTestDB()

	// Setup Data
	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	db.Create(&list)

	item1 := models.Item{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Name:        "Milk",
		ListID:      list.ID,
		IsBought:    false,
	}
	db.Create(&item1)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   "test.jpg",
		Status:      "new",
	}
	db.Create(&receipt)

	// Setup Mock — Milk gets auto-matched via AI; Bread is unmatched (not on planned list)
	mock := &MockParser{
		ParseFunc: func(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error) {
			return &ai.ParsedReceipt{
				StoreName: "SuperMart",
				Date:      "2024-01-30",
				Total:     10.5,
				Items: []ai.ParsedReceiptItem{
					{Name: "Milk", Price: 2.5, Quantity: 1, TotalPrice: 2.5},
					{Name: "Bread", Price: 8.0, Quantity: 1, TotalPrice: 8.0}, // Not on planned list
				},
			}, nil
		},
		MatchItemsFunc: func(ctx context.Context, receiptItems []string, plannedItems []string) (*ai.MatchResult, error) {
			// Return high-confidence match for Milk → Milk
			result := &ai.MatchResult{}
			for _, ri := range receiptItems {
				if ri == "Milk" {
					result.Suggestions = append(result.Suggestions, ai.MatchSuggestion{
						ReceiptItemName: "Milk",
						Matches:         []ai.MatchCandidate{{PlannedItemName: "Milk", Confidence: 95}},
					})
				}
			}
			return result, nil
		},
	}

	svc := NewReceiptService(db, mock, nil, "/tmp")

	// Act
	err := svc.ProcessReceipt(context.Background(), receipt.ID, list.ID)

	// Assert
	assert.NoError(t, err)

	// Check Receipt Status — "pending_review" because Bread is an unmatched extra item
	var updatedReceipt models.Receipt
	db.First(&updatedReceipt, "id = ?", receipt.ID)
	assert.Equal(t, "pending_review", updatedReceipt.Status)
	assert.NotNil(t, updatedReceipt.ShopID)
	assert.Equal(t, 10.5, updatedReceipt.Total)

	// Check Milk was matched and marked bought
	var updatedItem1 models.Item
	db.First(&updatedItem1, "id = ?", item1.ID)
	assert.True(t, updatedItem1.IsBought)
	assert.Equal(t, 2.5, updatedItem1.Price)
	assert.NotNil(t, updatedItem1.ReceiptItemID)
	assert.Equal(t, 1.0, updatedItem1.Quantity)

	// Check Bread created a ReceiptItem with "unmatched" status (not a new planned Item)
	var breadReceiptItem models.ReceiptItem
	db.Where("name = ?", "Bread").First(&breadReceiptItem)
	assert.NotZero(t, breadReceiptItem.ID)
	assert.Equal(t, "unmatched", breadReceiptItem.MatchStatus)

	// Check List Title Updated
	var updatedList models.ShoppingList
	db.First(&updatedList, "id = ?", list.ID)
	assert.True(t, strings.Contains(updatedList.Title, "(2024-01-30)"))

	// Check ActualAmount updated (only Milk is bought: 2.5)
	assert.Equal(t, 2.5, updatedList.ActualAmount)
}

func TestProcessPendingReceipts(t *testing.T) {
	db := setupTestDB()

	// Setup
	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFam"}}
	db.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "List",
	}
	db.Create(&list)

	receipt1 := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		ListID:      &list.ID,
		ImagePath:   "r1.jpg",
		Status:      "new",
	}
	receipt2 := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
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
	db.First(&r1, "id = ?", receipt1.ID)
	assert.Equal(t, "parsed", r1.Status)
}

// TestProcessReceipt_TextFile verifies that .txt receipts are routed to ParseReceiptText.
func TestProcessReceipt_TextFile(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	db.Create(&list)

	// Write a real .txt file to the temp directory
	textContent := "Store: Lidl\nMilk 1,99\nBread 2,49\nTotal 4,48"
	relPath := filepath.Join("families", "test", "receipts", "2024", "01", "test_receipt.txt")
	fullPath := filepath.Join(tmpDir, relPath)
	assert.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	assert.NoError(t, os.WriteFile(fullPath, []byte(textContent), 0644))

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
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
	db.First(&updated, "id = ?", receipt.ID)
	// No planned items → Milk is unmatched → pending_review
	assert.Equal(t, "pending_review", updated.Status)
}

// TestCreateReceiptFromText verifies text is saved to filesystem and receipt record created.
func TestCreateReceiptFromText(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFam"}}
	db.Create(&family)

	storage := NewFileStorageService(tmpDir)
	svc := NewReceiptService(db, nil, storage, tmpDir)

	text := "Store: TestMart\nTotal: 10.50\nMilk 2.50\nBread 8.00"
	receipt, err := svc.CreateReceiptFromText(family.ID, text)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.NotEqual(t, uuid.Nil, receipt.ID)
	assert.True(t, strings.HasSuffix(receipt.ImagePath, ".txt"), "ImagePath should end with .txt")

	// Verify file was written with correct content
	fullPath := filepath.Join(tmpDir, receipt.ImagePath)
	content, readErr := os.ReadFile(fullPath)
	assert.NoError(t, readErr)
	assert.Equal(t, text, string(content))

	// Verify DB record exists
	var dbReceipt models.Receipt
	db.First(&dbReceipt, "id = ?", receipt.ID)
	assert.Equal(t, receipt.ImagePath, dbReceipt.ImagePath)
	assert.Equal(t, family.ID, dbReceipt.FamilyID)
}

// TestProcessReceipt_TextFile_ParseError verifies error path sets status to "error".
func TestProcessReceipt_TextFile_ParseError(t *testing.T) {
	db := setupTestDB()
	tmpDir := t.TempDir()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "TestFam"}}
	db.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "List",
	}
	db.Create(&list)

	relPath := filepath.Join("families", "test", "receipts", "2024", "01", "bad.txt")
	fullPath := filepath.Join(tmpDir, relPath)
	assert.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	assert.NoError(t, os.WriteFile(fullPath, []byte("garbage text"), 0644))

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
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
	db.First(&updated, "id = ?", receipt.ID)
	assert.Equal(t, "error", updated.Status)
}
