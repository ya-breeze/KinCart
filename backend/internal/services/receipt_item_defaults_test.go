package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coremodels "github.com/ya-breeze/kin-core/models"

	"kincart/internal/ai"
	"kincart/internal/models"
)

// receiptFixture builds a family, a list, a receipt and one unmatched receipt item,
// with an optional AI mock. The item name is the caller's, so history seeded for it
// resolves. gemini is nil unless a mock is supplied.
func receiptFixture(t *testing.T, mock ReceiptParser, itemName string) (
	db *gorm.DB, svc *ReceiptService, familyID uuid.UUID, receiptItemID uint) {
	t.Helper()
	db = setupTestDB()
	svc = NewReceiptService(db, mock, nil, "/tmp")

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Fam"}}
	require.NoError(t, db.Create(&family).Error)
	familyID = family.ID

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Title:       "List",
	}
	require.NoError(t, db.Create(&list).Error)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ListID:      &list.ID,
		ImagePath:   "r.jpg",
		Status:      "pending_review",
	}
	require.NoError(t, db.Create(&receipt).Error)

	ri := models.ReceiptItem{
		ReceiptID: receipt.ID, Name: itemName, Price: 42, Quantity: 1,
		Unit: "", MatchStatus: "unmatched",
	}
	require.NoError(t, db.Create(&ri).Error)
	return db, svc, familyID, ri.ID
}

func seedCategory(t *testing.T, db *gorm.DB, familyID uuid.UUID, name string, sortOrder int) models.Category {
	t.Helper()
	cat := models.Category{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        name, SortOrder: sortOrder,
	}
	require.NoError(t, db.Create(&cat).Error)
	return cat
}

func seedAliasWithCategory(t *testing.T, db *gorm.DB, familyID uuid.UUID, name string, categoryID *uuid.UUID, unit string) {
	t.Helper()
	alias := models.ItemAlias{
		FamilyID: familyID, PlannedName: name, PlannedNameLower: strings.ToLower(name),
		ReceiptName: name, ReceiptNameLower: strings.ToLower(name),
		Unit: unit, CategoryID: categoryID, PurchaseCount: 3, LastUsedAt: time.Now(),
	}
	require.NoError(t, db.Create(&alias).Error)
}

func createdItem(t *testing.T, db *gorm.DB, receiptItemID uint) models.Item {
	t.Helper()
	var ri models.ReceiptItem
	require.NoError(t, db.First(&ri, receiptItemID).Error)
	require.NotNil(t, ri.MatchedItemID, "a new item should have been created")
	var item models.Item
	require.NoError(t, db.First(&item, "id = ?", *ri.MatchedItemID).Error)
	return item
}

// The remembered category wins over the first-by-sort-order default that this change
// replaces. "Snacks" sorts first, but history filed the item under "Dairy".
func TestConfirmMatch_NewItemUsesRememberedCategory(t *testing.T) {
	db, svc, familyID, receiptItemID := receiptFixture(t, nil, "yogurt")
	seedCategory(t, db, familyID, "Snacks", 0) // first by sort order
	dairy := seedCategory(t, db, familyID, "Dairy", 5)
	seedAliasWithCategory(t, db, familyID, "yogurt", &dairy.ID, "pack")

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID))

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, dairy.ID, item.CategoryID, "remembered category, not the first by sort order")
	assert.Equal(t, "pack", item.Unit, "remembered unit fills the receipt's blank unit")
}

// No history, no AI client → uncategorized (uuid.Nil), proving the old
// first-by-sort-order default is gone rather than merely reordered.
func TestConfirmMatch_NewItemNoHistoryNoAILeavesUncategorized(t *testing.T) {
	db, svc, familyID, receiptItemID := receiptFixture(t, nil, "dragonfruit")
	seedCategory(t, db, familyID, "Snacks", 0)

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID))

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, uuid.Nil, item.CategoryID, "no history + no AI must not fall back to the first category")
}

// No history, AI available → the AI's constrained pick, matched Go-side.
func TestConfirmMatch_NewItemFallsBackToAI(t *testing.T) {
	var gotNames []string
	mock := &MockParser{
		SuggestDefaultFunc: func(_ context.Context, name string, categories []string) (ai.SuggestedItemDefaults, error) {
			gotNames = categories
			return ai.SuggestedItemDefaults{Unit: "kg", Category: "Молочное"}, nil
		},
	}
	db, svc, familyID, receiptItemID := receiptFixture(t, mock, "тофу")
	seedCategory(t, db, familyID, "Snacks", 0)
	dairy := seedCategory(t, db, familyID, "Молочное", 5)

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID))

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, dairy.ID, item.CategoryID, "AI-suggested category, matched Cyrillic-safe")
	assert.Equal(t, "kg", item.Unit)
	assert.Contains(t, gotNames, "Молочное", "the family's own names are offered to the AI")
}

// AI returns a name the family does not have → uncategorized, never invented.
func TestConfirmMatch_NewItemAIUnmatchedLeavesUncategorized(t *testing.T) {
	mock := &MockParser{
		SuggestDefaultFunc: func(_ context.Context, _ string, _ []string) (ai.SuggestedItemDefaults, error) {
			return ai.SuggestedItemDefaults{Unit: "pcs", Category: "Nonexistent"}, nil
		},
	}
	db, svc, familyID, receiptItemID := receiptFixture(t, mock, "widget")
	seedCategory(t, db, familyID, "Dairy", 0)

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID))

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, uuid.Nil, item.CategoryID, "an unmatched AI name invents no category")
}

// Linking a receipt item to an existing planned item creates no new item, so it must
// not pay for a history read + AI call whose result would be discarded.
func TestConfirmMatch_LinkingExistingItemSkipsResolution(t *testing.T) {
	aiCalls := 0
	mock := &MockParser{
		SuggestDefaultFunc: func(_ context.Context, _ string, _ []string) (ai.SuggestedItemDefaults, error) {
			aiCalls++
			return ai.SuggestedItemDefaults{}, nil
		},
	}
	db, svc, familyID, receiptItemID := receiptFixture(t, mock, "тофу")

	// An existing planned item on the receipt's list, with no history for its name,
	// so resolution — if wrongly run — would fall through to the AI.
	var ri models.ReceiptItem
	require.NoError(t, db.First(&ri, receiptItemID).Error)
	var receipt models.Receipt
	require.NoError(t, db.First(&receipt, "id = ?", ri.ReceiptID).Error)
	planned := models.Item{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        "planned thing", ListID: *receipt.ListID, IsBought: false,
	}
	require.NoError(t, db.Create(&planned).Error)

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, &planned.ID, familyID))
	assert.Equal(t, 0, aiCalls, "linking an existing item must not trigger the AI categorize call")
}

// A failing AI call (an error, as a timeout would produce) must not fail the confirm:
// the new item is created uncategorized. This is the degradation the geminiCategorizeTimeout
// relies on.
func TestConfirmMatch_NewItemAIErrorLeavesUncategorized(t *testing.T) {
	mock := &MockParser{
		SuggestDefaultFunc: func(_ context.Context, _ string, _ []string) (ai.SuggestedItemDefaults, error) {
			return ai.SuggestedItemDefaults{}, context.DeadlineExceeded
		},
	}
	db, svc, familyID, receiptItemID := receiptFixture(t, mock, "widget")
	seedCategory(t, db, familyID, "Dairy", 0)

	require.NoError(t, svc.ConfirmMatch(context.Background(), receiptItemID, nil, familyID),
		"an AI error must degrade, not fail the confirm")

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, uuid.Nil, item.CategoryID)
}

// An auto-matched purchase (ProcessReceipt, high confidence) records the planned
// item's unit/category to history, so the remembered defaults stay fresh instead of
// freezing at whatever they were when the alias was first created.
func TestProcessReceipt_AutoMatchRecordsCategoryToHistory(t *testing.T) {
	db := setupTestDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Fam"}}
	require.NoError(t, db.Create(&family).Error)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	require.NoError(t, db.Create(&list).Error)
	dairy := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "Dairy"}
	require.NoError(t, db.Create(&dairy).Error)

	// Planned item already filed under Dairy, as "pack".
	milk := models.Item{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Name:        "Milk", ListID: list.ID, CategoryID: dairy.ID, Unit: "pack",
	}
	require.NoError(t, db.Create(&milk).Error)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		ListID:      &list.ID, ImagePath: "r.jpg", Status: "new",
	}
	require.NoError(t, db.Create(&receipt).Error)

	mock := &MockParser{
		ParseFunc: func(_ context.Context, _ string, _ []string) (*ai.ParsedReceipt, error) {
			return &ai.ParsedReceipt{
				StoreName: "Shop", Date: "2024-01-30", Total: 2.5,
				Items: []ai.ParsedReceiptItem{{Name: "Milk", Price: 2.5, Quantity: 1, TotalPrice: 2.5}},
			}, nil
		},
		MatchItemsFunc: func(_ context.Context, _ []string, _ []string) (*ai.MatchResult, error) {
			return &ai.MatchResult{Suggestions: []ai.MatchSuggestion{{
				ReceiptItemName: "Milk",
				Matches:         []ai.MatchCandidate{{PlannedItemName: "Milk", Confidence: 95}},
			}}}, nil
		},
	}
	svc := NewReceiptService(db, mock, nil, "/tmp")

	require.NoError(t, svc.ProcessReceipt(context.Background(), receipt.ID, list.ID))

	// The auto-match must have created/refreshed an alias carrying the category.
	var alias models.ItemAlias
	require.NoError(t, db.Where("family_id = ? AND planned_name_lower = ?", family.ID, "milk").First(&alias).Error,
		"auto-match should record an alias for the purchased item")
	require.NotNil(t, alias.CategoryID)
	assert.Equal(t, dairy.ID, *alias.CategoryID, "the planned item's category is recorded to history")
	assert.Equal(t, "pack", alias.Unit)
}

// ConfirmAllMatches applies remembered categories to every unmatched item it creates.
func TestConfirmAllMatches_NewItemsUseRememberedCategory(t *testing.T) {
	db, svc, familyID, receiptItemID := receiptFixture(t, nil, "yogurt")
	seedCategory(t, db, familyID, "Snacks", 0)
	dairy := seedCategory(t, db, familyID, "Dairy", 5)
	seedAliasWithCategory(t, db, familyID, "yogurt", &dairy.ID, "pack")

	// Fetch the receipt ID via the seeded item.
	var ri models.ReceiptItem
	require.NoError(t, db.First(&ri, receiptItemID).Error)

	require.NoError(t, svc.ConfirmAllMatches(context.Background(), ri.ReceiptID, familyID))

	item := createdItem(t, db, receiptItemID)
	assert.Equal(t, dairy.ID, item.CategoryID)
	assert.Equal(t, "pack", item.Unit)
}
