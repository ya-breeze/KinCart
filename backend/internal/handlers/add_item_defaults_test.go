package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	coremodels "github.com/ya-breeze/kin-core/models"

	"kincart/internal/database"
	"kincart/internal/models"
)

func setupAddItemTestDB(t *testing.T) {
	t.Helper()
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.DB.AutoMigrate(
		&models.Family{}, &models.ShoppingList{}, &models.Item{},
		&models.Category{}, &models.Shop{}, &models.ItemAlias{}, &models.ItemFrequency{},
	))
}

func newAddTestList(t *testing.T, familyID uuid.UUID, shopID *uuid.UUID) models.ShoppingList {
	t.Helper()
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Title:       "Weekly",
		ShopID:      shopID,
	}
	require.NoError(t, database.DB.Create(&list).Error)
	return list
}

func doAddItemToList(t *testing.T, familyID, listID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/lists/"+listID.String()+"/items", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: listID.String()}}
	c.Set("family_id", familyID)
	AddItemToList(c)
	return w
}

func doBulkAdd(t *testing.T, familyID, listID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/lists/"+listID.String()+"/items/bulk", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: listID.String()}}
	c.Set("family_id", familyID)
	BulkAddItems(c)
	return w
}

func seedAddAlias(t *testing.T, familyID uuid.UUID, shopID *uuid.UUID, unit string, categoryID *uuid.UUID) {
	name := "yogurt"
	t.Helper()
	alias := models.ItemAlias{
		FamilyID: familyID, PlannedName: name, PlannedNameLower: strings.ToLower(name),
		ReceiptName: name, ReceiptNameLower: strings.ToLower(name),
		ShopID: shopID, Unit: unit, CategoryID: categoryID,
		PurchaseCount: 3, LastUsedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&alias).Error)
}

// A manual single add of a remembered item is prefilled from history.
func TestAddItemToList_FillsFromHistory(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	dairy := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Dairy"}
	require.NoError(t, database.DB.Create(&dairy).Error)
	list := newAddTestList(t, fam, nil)
	seedAddAlias(t, fam, nil, "pack", &dairy.ID)

	w := doAddItemToList(t, fam, list.ID, `{"name":"yogurt","quantity":1,"unit":"pcs"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.Item
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "pack", got.Unit)
	assert.Equal(t, dairy.ID, got.CategoryID)
}

// An explicit category on the request survives; history does not overwrite it.
func TestAddItemToList_ExplicitCategoryKept(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	dairy := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Dairy"}
	breakfast := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Breakfast"}
	require.NoError(t, database.DB.Create(&dairy).Error)
	require.NoError(t, database.DB.Create(&breakfast).Error)
	list := newAddTestList(t, fam, nil)
	seedAddAlias(t, fam, nil, "pack", &dairy.ID)

	body := `{"name":"yogurt","quantity":1,"unit":"kg","category_id":"` + breakfast.ID.String() + `"}`
	w := doAddItemToList(t, fam, list.ID, body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.Item
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "kg", got.Unit, "an explicit unit is never overridden")
	assert.Equal(t, breakfast.ID, got.CategoryID, "an explicit category is never overridden")
}

// An unseen item makes no AI call and keeps the plain defaults. (No GEMINI_API_KEY
// is set in tests, so any attempt to reach AI on this path would be a build/runtime
// signal; the assertion is that the defaults are simply left untouched.)
func TestAddItemToList_UnknownKeepsDefaults(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	list := newAddTestList(t, fam, nil)

	w := doAddItemToList(t, fam, list.ID, `{"name":"dragonfruit","quantity":1,"unit":"pcs"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.Item
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "pcs", got.Unit)
	assert.Equal(t, uuid.Nil, got.CategoryID)
}

// Unit memory is per-shop: the list's shop wins over a more frequent unit elsewhere.
func TestAddItemToList_UsesListShopForUnit(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	makro := models.Shop{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Makro"}
	albert := models.Shop{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Albert"}
	require.NoError(t, database.DB.Create(&makro).Error)
	require.NoError(t, database.DB.Create(&albert).Error)
	list := newAddTestList(t, fam, &makro.ID)
	seedAddAlias(t, fam, &albert.ID, "pcs", nil)
	seedAddAlias(t, fam, &makro.ID, "pack", nil)

	w := doAddItemToList(t, fam, list.ID, `{"name":"yogurt","quantity":1,"unit":"pcs"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.Item
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "pack", got.Unit)
}

// Bulk add resolves every item from history.
func TestBulkAddItems_FillsFromHistory(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	dairy := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Dairy"}
	require.NoError(t, database.DB.Create(&dairy).Error)
	list := newAddTestList(t, fam, nil)
	seedAddAlias(t, fam, nil, "pack", &dairy.ID)

	body := `[{"name":"yogurt","quantity":1,"unit":"pcs"},{"name":"unknown","quantity":1,"unit":"pcs"}]`
	w := doBulkAdd(t, fam, list.ID, body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var items []models.Item
	require.NoError(t, database.DB.Where("list_id = ?", list.ID).Order("name").Find(&items).Error)
	require.Len(t, items, 2)
	// "unknown" then "yogurt" by name order.
	assert.Equal(t, "pcs", items[0].Unit, "unseen item keeps the default")
	assert.Equal(t, uuid.Nil, items[0].CategoryID)
	assert.Equal(t, "pack", items[1].Unit)
	assert.Equal(t, dairy.ID, items[1].CategoryID)
}

// Regression: BulkAddItems did not validate category ownership, unlike every other
// item-write path, so a client could file items under another family's category.
func TestBulkAddItems_RejectsForeignCategory(t *testing.T) {
	setupAddItemTestDB(t)
	fam := uuid.New()
	other := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: other, Name: "O"}}).Error)
	foreignCat := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: other}, Name: "Theirs"}
	require.NoError(t, database.DB.Create(&foreignCat).Error)
	list := newAddTestList(t, fam, nil)

	body := `[{"name":"x","quantity":1,"unit":"pcs","category_id":"` + foreignCat.ID.String() + `"}]`
	w := doBulkAdd(t, fam, list.ID, body)
	assert.Equal(t, http.StatusBadRequest, w.Code, "a foreign category must be rejected, not silently stored")

	var count int64
	database.DB.Model(&models.Item{}).Where("list_id = ?", list.ID).Count(&count)
	assert.Equal(t, int64(0), count, "nothing is created when validation fails")
}
