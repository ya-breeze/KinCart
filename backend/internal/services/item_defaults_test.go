package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	coremodels "github.com/ya-breeze/kin-core/models"

	"kincart/internal/models"
)

// mkAlias inserts an alias, filling in the lowercased-name column the resolver
// looks names up by.
func mkAlias(t *testing.T, db *gorm.DB, familyID uuid.UUID, name string, shopID *uuid.UUID,
	unit string, categoryID *uuid.UUID, purchaseCount int, lastUsed time.Time) {
	t.Helper()
	alias := models.ItemAlias{
		FamilyID:         familyID,
		PlannedName:      name,
		PlannedNameLower: strings.ToLower(name),
		ReceiptName:      name,
		ReceiptNameLower: strings.ToLower(name),
		ShopID:           shopID,
		Unit:             unit,
		CategoryID:       categoryID,
		PurchaseCount:    purchaseCount,
		LastUsedAt:       lastUsed,
	}
	assert.NoError(t, db.Create(&alias).Error)
}

func mkCategory(t *testing.T, db *gorm.DB, familyID uuid.UUID, name string) models.Category {
	t.Helper()
	cat := models.Category{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        name,
	}
	assert.NoError(t, db.Create(&cat).Error)
	return cat
}

func mkShop(t *testing.T, db *gorm.DB, familyID uuid.UUID, name string) models.Shop {
	t.Helper()
	shop := models.Shop{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Name:        name,
	}
	assert.NoError(t, db.Create(&shop).Error)
	return shop
}

func TestResolveItemDefaults_UnitPrefersListShop(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	makro := mkShop(t, db, fam, "Makro")
	albert := mkShop(t, db, fam, "Albert")

	// Bought loose at Albert far more often, but as a pack at Makro.
	mkAlias(t, db, fam, "jogurt", &albert.ID, "pcs", nil, 10, time.Now())
	mkAlias(t, db, fam, "jogurt", &makro.ID, "pack", nil, 1, time.Now())

	got, err := ResolveItemDefaults(context.Background(), db, fam, "jogurt", &makro.ID)
	assert.NoError(t, err)
	assert.Equal(t, "pack", got.Unit, "the list's shop wins over a more frequent unit elsewhere")
}

func TestResolveItemDefaults_UnitFallsBackAcrossShops(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	albert := mkShop(t, db, fam, "Albert")
	lidl := mkShop(t, db, fam, "Lidl")
	unknown := mkShop(t, db, fam, "Never shopped here")

	mkAlias(t, db, fam, "mleko", &albert.ID, "l", nil, 5, time.Now())
	mkAlias(t, db, fam, "mleko", &lidl.ID, "l", nil, 3, time.Now())
	mkAlias(t, db, fam, "mleko", nil, "ml", nil, 1, time.Now())

	// No history at this shop → most common across shops.
	got, err := ResolveItemDefaults(context.Background(), db, fam, "mleko", &unknown.ID)
	assert.NoError(t, err)
	assert.Equal(t, "l", got.Unit)

	// No shop on the list at all → same fallback.
	got, err = ResolveItemDefaults(context.Background(), db, fam, "mleko", nil)
	assert.NoError(t, err)
	assert.Equal(t, "l", got.Unit)
}

func TestResolveItemDefaults_CategoryUsesMostRecent(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Dairy")
	breakfast := mkCategory(t, db, fam, "Breakfast")

	old := time.Now().Add(-72 * time.Hour)
	// Filed under Dairy many times, then deliberately moved to Breakfast once.
	mkAlias(t, db, fam, "jogurt", nil, "pcs", &dairy.ID, 20, old)
	mkAlias(t, db, fam, "jogurt", nil, "pcs", &breakfast.ID, 1, time.Now())

	got, err := ResolveItemDefaults(context.Background(), db, fam, "jogurt", nil)
	assert.NoError(t, err)
	assert.NotNil(t, got.CategoryID)
	assert.Equal(t, breakfast.ID, *got.CategoryID,
		"a recent recategorisation must not be outvoted by how often it was filed the old way")
}

func TestResolveItemDefaults_CategoryTieBrokenByPurchaseCount(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Dairy")
	snacks := mkCategory(t, db, fam, "Snacks")

	same := time.Now()
	mkAlias(t, db, fam, "syr", nil, "kg", &snacks.ID, 2, same)
	mkAlias(t, db, fam, "syr", nil, "kg", &dairy.ID, 9, same)

	got, err := ResolveItemDefaults(context.Background(), db, fam, "syr", nil)
	assert.NoError(t, err)
	assert.NotNil(t, got.CategoryID)
	assert.Equal(t, dairy.ID, *got.CategoryID, "identical timestamps break toward more purchases")
}

func TestResolveItemDefaults_EmptyWhenUnknown(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()

	got, err := ResolveItemDefaults(context.Background(), db, fam, "never seen", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", got.Unit)
	assert.Nil(t, got.CategoryID)
	assert.False(t, got.Known(), "nothing resolved means the caller keeps pcs/uncategorized")
}

func TestResolveItemDefaults_IgnoresOtherFamilies(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	other := uuid.New()
	cat := mkCategory(t, db, other, "Dairy")
	mkAlias(t, db, other, "jogurt", nil, "pack", &cat.ID, 5, time.Now())

	got, err := ResolveItemDefaults(context.Background(), db, fam, "jogurt", nil)
	assert.NoError(t, err)
	assert.False(t, got.Known(), "another family's history must never leak in")
}

// The name lookup must fold case Go-side. SQLite's LOWER() is ASCII-only, so a
// SQL-side comparison would miss this entirely.
func TestResolveItemDefaults_CyrillicNameMatchesRegardlessOfCase(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	cat := mkCategory(t, db, fam, "Молочное")
	mkAlias(t, db, fam, "Йогурт", nil, "pack", &cat.ID, 3, time.Now())

	got, err := ResolveItemDefaults(context.Background(), db, fam, "йогурт", nil)
	assert.NoError(t, err)
	assert.Equal(t, "pack", got.Unit)
	assert.NotNil(t, got.CategoryID)
	assert.Equal(t, cat.ID, *got.CategoryID)
}

func TestResolveItemDefaultsBatch_ResolvesManyNames(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Dairy")
	mkAlias(t, db, fam, "mleko", nil, "l", &dairy.ID, 4, time.Now())
	mkAlias(t, db, fam, "Chleba", nil, "pcs", nil, 2, time.Now())

	got, err := ResolveItemDefaultsBatch(context.Background(), db, fam,
		[]string{"mleko", "CHLEBA", "unknown item"}, nil)
	assert.NoError(t, err)

	assert.Equal(t, "l", got["mleko"].Unit)
	assert.Equal(t, "pcs", got["chleba"].Unit, "results are keyed by the lowercased name")
	assert.False(t, got["unknown item"].Known())
}

func TestMatchCategoryName_NonASCIICaseInsensitive(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Молочное")
	mkCategory(t, db, fam, "Ovoce a zelenina")

	categories, err := LoadFamilyCategories(context.Background(), db, fam)
	assert.NoError(t, err)

	got := MatchCategoryName(categories, "молочное")
	assert.NotNil(t, got, "Cyrillic must fold Go-side; SQL LOWER() would fail this")
	assert.Equal(t, dairy.ID, *got)

	// Whitespace the model may pad the value with must not defeat the match.
	got = MatchCategoryName(categories, "  Ovoce a zelenina ")
	assert.NotNil(t, got)
}

func TestMatchCategoryName_UnmatchedLeavesUncategorized(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	mkCategory(t, db, fam, "Молочное")

	categories, err := LoadFamilyCategories(context.Background(), db, fam)
	assert.NoError(t, err)

	assert.Nil(t, MatchCategoryName(categories, "Dairy"),
		"an English guess that matches no family category invents nothing")
	assert.Nil(t, MatchCategoryName(categories, ""))
	assert.Nil(t, MatchCategoryName(nil, "Молочное"))
}

func TestCategoryNames_SkipsEmpty(t *testing.T) {
	names := CategoryNames([]models.Category{{Name: "Dairy"}, {Name: ""}, {Name: "Молочное"}})
	assert.Equal(t, []string{"Dairy", "Молочное"}, names)
}

// Guards the rule that a caller which simply does not know must not erase history
// gathered from earlier purchases.
func TestUpsertItemAlias_EmptyValuesDoNotEraseHistory(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Dairy")

	_, err := UpsertItemAlias(db, fam, "jogurt", "selský jogurt", 25.9, nil, "pack", &dairy.ID)
	assert.NoError(t, err)

	// A later purchase that knows neither unit nor category.
	_, err = UpsertItemAlias(db, fam, "jogurt", "selský jogurt", 27.5, nil, "", nil)
	assert.NoError(t, err)

	var alias models.ItemAlias
	assert.NoError(t, db.Where("family_id = ?", fam).First(&alias).Error)
	assert.Equal(t, "pack", alias.Unit, "an empty unit must not wipe a remembered one")
	assert.NotNil(t, alias.CategoryID, "a nil category must not wipe a remembered one")
	assert.Equal(t, dairy.ID, *alias.CategoryID)
	assert.Equal(t, 27.5, alias.LastPrice, "price still updates")
	assert.Equal(t, 2, alias.PurchaseCount)
}

func TestUpsertItemAlias_NewValuesOverwrite(t *testing.T) {
	db := setupTestDB()
	fam := uuid.New()
	dairy := mkCategory(t, db, fam, "Dairy")
	breakfast := mkCategory(t, db, fam, "Breakfast")

	_, err := UpsertItemAlias(db, fam, "jogurt", "selský jogurt", 25.9, nil, "pcs", &dairy.ID)
	assert.NoError(t, err)
	_, err = UpsertItemAlias(db, fam, "jogurt", "selský jogurt", 25.9, nil, "pack", &breakfast.ID)
	assert.NoError(t, err)

	var alias models.ItemAlias
	assert.NoError(t, db.Where("family_id = ?", fam).First(&alias).Error)
	assert.Equal(t, "pack", alias.Unit)
	assert.Equal(t, breakfast.ID, *alias.CategoryID, "latest wins when the caller does know")
}

func TestCategoryIDPtr_ZeroUUIDBecomesNil(t *testing.T) {
	assert.Nil(t, CategoryIDPtr(uuid.Nil),
		"the zero UUID means uncategorized, not a category that exists")
	id := uuid.New()
	assert.Equal(t, &id, CategoryIDPtr(id))
}
