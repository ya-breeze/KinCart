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

// setupConfirmMatchFixture creates the minimal DB state needed for ConfirmMatch tests:
// a family, list, receipt, and one receipt item with the given matchStatus.
func setupConfirmMatchFixture(t *testing.T) (db *gorm.DB, svc *ReceiptService, familyID uuid.UUID, listID uuid.UUID, receiptItemID uint) {
	t.Helper()
	db = setupTestDB()
	svc = NewReceiptService(db, nil, nil, "/tmp")

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Fam"}}
	db.Create(&family)
	familyID = family.ID

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Title:       "List",
	}
	db.Create(&list)
	listID = list.ID

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ListID:      &listID,
		ImagePath:   "r.jpg",
		Status:      "pending_review",
	}
	db.Create(&receipt)
	receiptID := receipt.ID

	ri := models.ReceiptItem{
		ReceiptID:   receiptID,
		Name:        "PRAŽMA",
		Price:       369,
		Quantity:    1,
		MatchStatus: "unmatched",
	}
	db.Create(&ri)
	receiptItemID = ri.ID
	return db, svc, familyID, listID, receiptItemID
}

// TestConfirmMatch_Unmatch_DeletesReceiptCreatedItem verifies that unmatching a receipt-created
// item (IsReceiptCreated=true) soft-deletes it instead of leaving it as a phantom.
func TestConfirmMatch_Unmatch_DeletesReceiptCreatedItem(t *testing.T) {
	db, svc, familyID, _, receiptItemID := setupConfirmMatchFixture(t)

	// First call: Add as new (plannedItemID == nil, not previously matched) → creates item
	err := svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
	assert.NoError(t, err)

	var ri models.ReceiptItem
	db.First(&ri, receiptItemID)
	assert.Equal(t, "confirmed", ri.MatchStatus)
	assert.NotNil(t, ri.MatchedItemID)
	createdItemID := *ri.MatchedItemID

	var createdItem models.Item
	db.First(&createdItem, "id = ?", createdItemID)
	assert.True(t, createdItem.IsBought)
	assert.True(t, createdItem.IsReceiptCreated)

	// Second call: Unmatch (plannedItemID == nil on a confirmed item) → should DELETE the created item
	err = svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
	assert.NoError(t, err)

	// The receipt-created item must be soft-deleted, not left as a phantom
	var count int64
	db.Model(&models.Item{}).Where("id = ?", createdItemID).Count(&count)
	assert.Equal(t, int64(0), count, "receipt-created item should be soft-deleted after unmatch")
}

// TestConfirmMatch_Unmatch_KeepsPlannedItem verifies that unmatching a pre-existing planned item
// (ReceiptItemID != this receipt item) only unlinks it — does not delete it.
func TestConfirmMatch_Unmatch_KeepsPlannedItem(t *testing.T) {
	db, svc, familyID, listID, receiptItemID := setupConfirmMatchFixture(t)

	// Pre-existing planned item (user added before receipt upload)
	planned := models.Item{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        "дорадо",
		ListID:      listID,
		IsBought:    false,
	}
	db.Create(&planned)

	// Confirm: link receipt item to planned item
	err := svc.ConfirmMatch(context.Background(), receiptItemID, &planned.ID, familyID)
	assert.NoError(t, err)

	var ri models.ReceiptItem
	db.First(&ri, receiptItemID)
	assert.Equal(t, "confirmed", ri.MatchStatus)

	// Unmatch: plannedItemID == nil on a confirmed item
	err = svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
	assert.NoError(t, err)

	// Planned item must still exist, just unlinked
	var stillThere models.Item
	err = db.First(&stillThere, "id = ?", planned.ID).Error
	assert.NoError(t, err, "pre-existing planned item must not be deleted on unmatch")
	assert.Nil(t, stillThere.ReceiptItemID, "ReceiptItemID should be cleared")
	assert.False(t, stillThere.IsBought, "IsBought should be reverted")
}

// TestConfirmMatch_RepeatedAddNew_NoPhantomAccumulation is the regression test for the bug
// where clicking "Add as new" → "Unmatch" → "Add as new" accumulated phantom items.
func TestConfirmMatch_RepeatedAddNew_NoPhantomAccumulation(t *testing.T) {
	db, svc, familyID, listID, receiptItemID := setupConfirmMatchFixture(t)

	for range 3 {
		// Add as new
		err := svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
		assert.NoError(t, err)
		// Unmatch
		err = svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
		assert.NoError(t, err)
	}
	// Final "Add as new" — leaves one item confirmed
	err := svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID)
	assert.NoError(t, err)

	// Exactly one item must exist in the list — no phantoms
	var items []models.Item
	db.Where("list_id = ?", listID).Find(&items)
	assert.Len(t, items, 1, "only the final confirmed item should exist; no phantom duplicates")
	assert.True(t, items[0].IsBought)
	assert.NotNil(t, items[0].ReceiptItemID)
}

// TestBuildItemMatches_AlreadyBoughtItemMatchable verifies that a planned item that was manually
// marked as bought before the receipt upload is still included in the matching pool.
// Uses ASCII-only names because SQLite's LOWER() only handles ASCII characters.
func TestBuildItemMatches_AlreadyBoughtItemMatchable(t *testing.T) {
	db := setupTestDB()
	svc := NewReceiptService(db, nil, nil, t.TempDir())

	familyID := uuid.New()
	family := models.Family{Family: coremodels.Family{ID: familyID, Name: "F"}}
	db.Create(&family)

	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID}}
	db.Create(&list)

	// Planned item already bought (manually ticked during shopping), not yet linked to any receipt item.
	milk := models.Item{
		TenantModel:   coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:          "Milk",
		ListID:        list.ID,
		IsBought:      true,
		ReceiptItemID: nil,
	}
	db.Create(&milk)

	// Alias: receipt name "WHOLE MILK 1L" → planned name "milk"
	alias := models.ItemAlias{
		FamilyID:         familyID,
		PlannedName:      "Milk",
		PlannedNameLower: "milk",
		ReceiptName:      "WHOLE MILK 1L",
		ReceiptNameLower: "whole milk 1l",
	}
	db.Create(&alias)

	parsedItems := []ai.ParsedReceiptItem{
		{Name: "WHOLE MILK 1L", Price: 18.90, Quantity: 1},
	}

	plans := svc.buildItemMatches(context.Background(), familyID, []models.Item{milk}, parsedItems)

	assert.Len(t, plans, 1)
	assert.Equal(t, matchStatusAuto, plans[0].MatchStatus, "already-bought item should be auto-matched via alias")
	assert.NotNil(t, plans[0].PlannedItemID)
	assert.Equal(t, milk.ID, *plans[0].PlannedItemID)
}

// TestBuildItemMatches_BoughtWithReceiptIDExcluded verifies that an already-bought item that is
// already linked to a receipt item is NOT included as a match candidate (it is already claimed).
func TestBuildItemMatches_BoughtWithReceiptIDExcluded(t *testing.T) {
	db := setupTestDB()
	svc := NewReceiptService(db, nil, nil, t.TempDir())

	familyID := uuid.New()
	family := models.Family{Family: coremodels.Family{ID: familyID, Name: "F"}}
	db.Create(&family)

	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID}}
	db.Create(&list)

	existingReceiptItemID := uint(999)
	milk := models.Item{
		TenantModel:   coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:          "Milk",
		ListID:        list.ID,
		IsBought:      true,
		ReceiptItemID: &existingReceiptItemID, // already claimed by another receipt item
	}
	db.Create(&milk)

	alias := models.ItemAlias{
		FamilyID:         familyID,
		PlannedName:      "Milk",
		PlannedNameLower: "milk",
		ReceiptName:      "WHOLE MILK 1L",
		ReceiptNameLower: "whole milk 1l",
	}
	db.Create(&alias)

	parsedItems := []ai.ParsedReceiptItem{
		{Name: "WHOLE MILK 1L", Price: 18.90, Quantity: 1},
	}

	plans := svc.buildItemMatches(context.Background(), familyID, []models.Item{milk}, parsedItems)

	assert.Len(t, plans, 1)
	assert.Equal(t, matchStatusUnmatched, plans[0].MatchStatus, "item already claimed by another receipt must not be matched again")
	assert.Nil(t, plans[0].PlannedItemID)
}

// TestGetReceiptMatches_AlreadyBoughtIncluded verifies that already-bought, unlinked items appear
// in the already_bought_items field (not unmatched_planned_items) of the matches response.
func TestGetReceiptMatches_AlreadyBoughtIncluded(t *testing.T) {
	db := setupTestDB()
	svc := NewReceiptService(db, nil, nil, t.TempDir())

	familyID := uuid.New()
	family := models.Family{Family: coremodels.Family{ID: familyID, Name: "F"}}
	db.Create(&family)

	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID}}
	db.Create(&list)

	// Already bought, not linked to any receipt item
	boughtItem := models.Item{
		TenantModel:   coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:          "Milk",
		ListID:        list.ID,
		IsBought:      true,
		ReceiptItemID: nil,
	}
	db.Create(&boughtItem)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ListID:      &list.ID,
		ImagePath:   "r.jpg",
		Status:      "pending_review",
	}
	db.Create(&receipt)

	ri := models.ReceiptItem{
		ReceiptID:   receipt.ID,
		Name:        "WHOLE MILK 1L",
		Price:       18.90,
		Quantity:    1,
		MatchStatus: matchStatusUnmatched,
	}
	db.Create(&ri)

	resp, err := svc.GetReceiptMatches(receipt.ID, familyID)
	assert.NoError(t, err)
	assert.Len(t, resp.AlreadyBoughtItems, 1, "bought unlinked item should appear in already_bought_items")
	assert.Equal(t, boughtItem.ID, resp.AlreadyBoughtItems[0].ID)
	assert.Len(t, resp.UnmatchedPlannedItems, 0, "bought item must not appear in unmatched_planned_items")
}

// TestNormalizePackItems verifies that multi-pack lines are collapsed to quantity=1.
func TestNormalizePackItems(t *testing.T) {
	cases := []struct {
		name       string
		itemName   string
		quantity   float64
		price      float64
		totalPrice float64
		wantQty    float64
		wantPrice  float64
		wantUnit   string
	}{
		{
			name:     "6×150g pack collapses",
			itemName: "Jogurt 6×150g", quantity: 6, price: 14.83, totalPrice: 89.0,
			wantQty: 1, wantPrice: 89.0, wantUnit: "pack",
		},
		{
			name:     "6x150g (ASCII x) collapses",
			itemName: "Jogurt 6x150g", quantity: 6, price: 14.83, totalPrice: 89.0,
			wantQty: 1, wantPrice: 89.0, wantUnit: "pack",
		},
		{
			name:     "3-pack collapses",
			itemName: "Máslo 3-pack", quantity: 3, price: 30.0, totalPrice: 90.0,
			wantQty: 1, wantPrice: 90.0, wantUnit: "pack",
		},
		{
			name:     "3 pack (with space) collapses",
			itemName: "Máslo 3 pack", quantity: 3, price: 30.0, totalPrice: 90.0,
			wantQty: 1, wantPrice: 90.0, wantUnit: "pack",
		},
		{
			name:     "no pack indicator — unchanged",
			itemName: "Mléko", quantity: 3, price: 25.0, totalPrice: 75.0,
			wantQty: 3, wantPrice: 25.0, wantUnit: "pcs",
		},
		{
			name:     "quantity=1 — unchanged even with pack name",
			itemName: "Jogurt 6×150g", quantity: 1, price: 89.0, totalPrice: 89.0,
			wantQty: 1, wantPrice: 89.0, wantUnit: "pcs",
		},
		{
			// "4+2" is handled by the AI prompt; the regex intentionally excludes \d+\+\d+
			// to avoid false-positives on supplement names like "Omega 3+6".
			name:     "4+2 promo — not collapsed by regex (AI handles it)",
			itemName: "Pivo 4+2", quantity: 6, price: 20.0, totalPrice: 120.0,
			wantQty: 6, wantPrice: 20.0, wantUnit: "pcs",
		},
		{
			name:     "Omega 3+6 supplement — not collapsed (false-positive guard)",
			itemName: "Omega 3+6", quantity: 2, price: 150.0, totalPrice: 300.0,
			wantQty: 2, wantPrice: 150.0, wantUnit: "pcs",
		},
		{
			// "10ks" is handled by the AI prompt; the regex intentionally excludes \d+ks
			// to avoid false-positives on pharmacy products like "Magne B6 60ks".
			name:     "10ks eggs — not collapsed by regex (AI handles it)",
			itemName: "Vejce 10ks", quantity: 10, price: 3.5, totalPrice: 35.0,
			wantQty: 10, wantPrice: 3.5, wantUnit: "pcs",
		},
		{
			name:     "Magne B6 60ks — not collapsed (false-positive guard)",
			itemName: "Magne B6 60ks", quantity: 2, price: 149.5, totalPrice: 299.0,
			wantQty: 2, wantPrice: 149.5, wantUnit: "pcs",
		},
		{
			name:     "total_price=0 — not collapsed even with pack name",
			itemName: "Jogurt 6×150g", quantity: 6, price: 14.83, totalPrice: 0,
			wantQty: 6, wantPrice: 14.83, wantUnit: "pcs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := &ai.ParsedReceipt{
				Items: []ai.ParsedReceiptItem{
					{Name: tc.itemName, Quantity: tc.quantity, Price: tc.price, TotalPrice: tc.totalPrice, Unit: "pcs"},
				},
			}
			normalizePackItems(parsed)
			assert.Equal(t, tc.wantQty, parsed.Items[0].Quantity, "quantity")
			assert.Equal(t, tc.wantPrice, parsed.Items[0].Price, "price")
			assert.Equal(t, tc.wantUnit, parsed.Items[0].Unit, "unit")
		})
	}
}

// TestConfirmMatch_AlreadyBoughtItem_UpdatesPriceAndLinks verifies that ConfirmMatch correctly
// links a receipt item to a planned item that was already manually bought (IsBought stays true).
func TestConfirmMatch_AlreadyBoughtItem_UpdatesPriceAndLinks(t *testing.T) {
	db, svc, familyID, listID, receiptItemID := setupConfirmMatchFixture(t)

	// Pre-existing item that was manually ticked before the receipt upload
	planned := models.Item{
		TenantModel:   coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:          "Milk",
		ListID:        listID,
		IsBought:      true,
		ReceiptItemID: nil,
	}
	db.Create(&planned)

	// Confirm: link the receipt item to the already-bought planned item
	err := svc.ConfirmMatch(context.Background(), receiptItemID, &planned.ID, familyID)
	assert.NoError(t, err)

	var updated models.Item
	db.First(&updated, "id = ?", planned.ID)
	assert.True(t, updated.IsBought, "IsBought must remain true")
	assert.NotNil(t, updated.ReceiptItemID, "ReceiptItemID must be set")
	assert.Equal(t, receiptItemID, *updated.ReceiptItemID)

	var ri models.ReceiptItem
	db.First(&ri, receiptItemID)
	assert.Equal(t, matchStatusConfirmed, ri.MatchStatus)
	assert.Equal(t, &planned.ID, ri.MatchedItemID)

	// No extra items should have been created
	var itemCount int64
	db.Model(&models.Item{}).Where("list_id = ?", listID).Count(&itemCount)
	assert.Equal(t, int64(1), itemCount, "no duplicate item should be created")
}

// TestConfirmMatch_AbsentItem_ClearsAbsent verifies that a receipt match resolves
// an item the shopper had marked "not found": the purchase is proof they got it,
// so IsAbsent must be cleared rather than left contradicting IsBought.
func TestConfirmMatch_AbsentItem_ClearsAbsent(t *testing.T) {
	db, svc, familyID, listID, receiptItemID := setupConfirmMatchFixture(t)

	// Shopper marked it absent in-store; the receipt later proves otherwise.
	// ASCII name on purpose -- the alias lookup uses LOWER(receipt_name),
	// which only behaves for ASCII in SQLite (see CLAUDE.md note 7).
	planned := models.Item{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        "Butter",
		ListID:      listID,
		IsBought:    false,
		IsAbsent:    true,
	}
	db.Create(&planned)

	err := svc.ConfirmMatch(context.Background(), receiptItemID, &planned.ID, familyID)
	assert.NoError(t, err)

	var updated models.Item
	db.First(&updated, "id = ?", planned.ID)
	assert.True(t, updated.IsBought, "a matched receipt item must be bought")
	assert.False(t, updated.IsAbsent, "bought and absent are mutually exclusive")
	assert.NotNil(t, updated.ReceiptItemID)
}
