package handlers

import (
	"bytes"
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

func setupParseTextTestDB(t *testing.T) {
	t.Helper()
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.DB.AutoMigrate(
		&models.Family{}, &models.ShoppingList{}, &models.Item{},
		&models.Category{}, &models.Shop{}, &models.ItemAlias{},
	))
}

// seedAliasForParse inserts an alias the way UpsertItemAlias does, with the
// lowercased columns populated.
func seedAliasForParse(t *testing.T, familyID uuid.UUID, plannedName, receiptName string, price float64) {
	t.Helper()
	alias := models.ItemAlias{
		FamilyID:         familyID,
		PlannedName:      plannedName,
		PlannedNameLower: strings.ToLower(plannedName),
		ReceiptName:      receiptName,
		ReceiptNameLower: strings.ToLower(receiptName),
		LastPrice:        price,
		PurchaseCount:    3,
		LastUsedAt:       time.Now(),
	}
	require.NoError(t, database.DB.Create(&alias).Error)
}

func postParseText(t *testing.T, familyID, listID uuid.UUID, text string) []map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)

	body, err := json.Marshal(map[string]string{"text": text})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/lists/"+listID.String()+"/parse-text", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: listID.String()}}
	c.Set("family_id", familyID)

	ParseListText(c)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var results []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	return results
}

// Regression: the alias lookup behind the paste preview used SQL
// LOWER(planned_name), and SQLite's LOWER() folds ASCII only. Every Cyrillic and
// accented-Czech item name therefore matched nothing, so the price hint (and, with
// this change, the remembered unit and category) silently never appeared for
// exactly the families whose categories this feature is built around.
//
// No GEMINI_API_KEY is needed: ParseListText falls back to the local parser, and
// the enrichment under test happens after parsing either way.
func TestParseListText_MatchesCyrillicAliasName(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Семья"}}
	require.NoError(t, database.DB.Create(&family).Error)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Покупки",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	// Stored capitalised; the pasted text uses lowercase.
	seedAliasForParse(t, family.ID, "Йогурт", "йогурт греческий 2%", 24.90)

	results := postParseText(t, family.ID, list.ID, "йогурт")
	require.Len(t, results, 1)

	assert.Equal(t, float64(24.90), results[0]["suggested_price"],
		"a Cyrillic name must resolve its alias history; SQL LOWER() would find nothing here")
	assert.Equal(t, float64(1), results[0]["alias_count"])
}

// The accented-Czech half of the same bug.
func TestParseListText_MatchesAccentedCzechAliasName(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Rodina"}}
	require.NoError(t, database.DB.Create(&family).Error)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Nákup",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	seedAliasForParse(t, family.ID, "Čočka", "čočka červená", 32.50)

	results := postParseText(t, family.ID, list.ID, "čočka")
	require.Len(t, results, 1)

	assert.Equal(t, float64(32.50), results[0]["suggested_price"],
		"accented Czech must resolve too — LOWER() leaves Č unfolded")
}

// ASCII names worked before the fix and must keep working after it.
func TestParseListText_StillMatchesASCIIAliasName(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Smiths"}}
	require.NoError(t, database.DB.Create(&family).Error)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	seedAliasForParse(t, family.ID, "Milk", "semi-skimmed milk", 1.25)

	results := postParseText(t, family.ID, list.ID, "milk")
	require.Len(t, results, 1)
	assert.Equal(t, float64(1.25), results[0]["suggested_price"])
}

// History supplies the unit when the pasted text named none.
func TestParseListText_RememberedUnitFillsUnspecified(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Smiths"}}
	require.NoError(t, database.DB.Create(&family).Error)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	cat := models.Category{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Name:        "Dairy",
	}
	require.NoError(t, database.DB.Create(&cat).Error)

	alias := models.ItemAlias{
		FamilyID: family.ID, PlannedName: "yogurt", PlannedNameLower: "yogurt",
		ReceiptName: "greek yogurt", ReceiptNameLower: "greek yogurt",
		Unit: "pack", CategoryID: &cat.ID, LastPrice: 2.5, PurchaseCount: 4,
		LastUsedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&alias).Error)

	results := postParseText(t, family.ID, list.ID, "yogurt")
	require.Len(t, results, 1)

	assert.Equal(t, "pack", results[0]["unit"], "no unit in the text → history fills it")
	assert.Equal(t, cat.ID.String(), results[0]["suggested_category_id"])
	assert.Equal(t, "Dairy", results[0]["suggested_category_name"])
}

// A unit written into the text is an explicit choice and must survive.
func TestParseListText_ExplicitUnitBeatsHistory(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Smiths"}}
	require.NoError(t, database.DB.Create(&family).Error)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	alias := models.ItemAlias{
		FamilyID: family.ID, PlannedName: "mouka", PlannedNameLower: "mouka",
		ReceiptName: "mouka hladká", ReceiptNameLower: "mouka hladká",
		Unit: "pack", PurchaseCount: 9, LastUsedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&alias).Error)

	results := postParseText(t, family.ID, list.ID, "2 kg mouka")
	require.Len(t, results, 1)
	assert.Equal(t, "kg", results[0]["unit"],
		"an explicitly pasted unit is a user choice and is never overridden by history")
}

// An item with no history and no AI pick keeps the plain defaults.
func TestParseListText_UnknownItemKeepsPlainDefaults(t *testing.T) {
	setupParseTextTestDB(t)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Smiths"}}
	require.NoError(t, database.DB.Create(&family).Error)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Weekly",
	}
	require.NoError(t, database.DB.Create(&list).Error)

	results := postParseText(t, family.ID, list.ID, "dragonfruit")
	require.Len(t, results, 1)
	assert.Equal(t, "pcs", results[0]["unit"])
	assert.Nil(t, results[0]["suggested_category_id"], "nothing known invents nothing")
}
