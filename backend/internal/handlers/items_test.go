package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/google/uuid"
	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupItemTestDBIsolated() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	database.DB.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Category{}, &models.ItemFrequency{})
}

func TestItemsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("AddItemToList", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.POST("/lists/:id/items", AddItemToList)

		newItem := models.Item{Name: "Bread"}
		jsonValue, _ := json.Marshal(newItem)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/items", list.ID.String()), bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var created models.Item
		json.Unmarshal(w.Body.Bytes(), &created)
		assert.Equal(t, "Bread", created.Name)
		assert.Equal(t, list.ID, created.ListID)

		// Check frequency
		var freq models.ItemFrequency
		database.DB.Where("item_name = ?", "Bread").First(&freq)
		assert.Equal(t, 1, freq.Frequency)
	})

	t.Run("UpdateItem", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)
		item := models.Item{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Name:        "Milk",
			ListID:      list.ID,
		}
		database.DB.Create(&item)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.PATCH("/items/:id", UpdateItem)

		updateData := map[string]interface{}{"name": "Organic Milk", "is_bought": true}
		jsonValue, _ := json.Marshal(updateData)
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%s", item.ID.String()), bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updated models.Item
		database.DB.First(&updated, "id = ?", item.ID)
		assert.Equal(t, "Organic Milk", updated.Name)
		assert.True(t, updated.IsBought)
	})

	t.Run("CreatedItemCannotBeBothBoughtAndAbsent", func(t *testing.T) {
		// AddItemToList binds a full models.Item, so a client can name both flags
		// in one payload -- UpdateItem's guard never sees these requests.
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.POST("/lists/:id/items", AddItemToList)

		body := map[string]interface{}{"name": "Milk", "is_bought": true, "is_absent": true}
		jsonValue, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/items", list.ID.String()), bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var created models.Item
		database.DB.Where("list_id = ? AND name = ?", list.ID, "Milk").First(&created)
		assert.True(t, created.IsBought)
		assert.False(t, created.IsAbsent, "bought wins on create, as it does on update")
	})

	t.Run("UpdateItemPersistsIsAbsent", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)
		item := models.Item{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Name:        "Saffron",
			ListID:      list.ID,
		}
		database.DB.Create(&item)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.PATCH("/items/:id", UpdateItem)

		// Setting is_absent true must round-trip through the map-based Updates.
		updateData := map[string]interface{}{"is_absent": true}
		jsonValue, _ := json.Marshal(updateData)
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%s", item.ID.String()), bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updated models.Item
		database.DB.First(&updated, "id = ?", item.ID)
		assert.True(t, updated.IsAbsent)
		assert.False(t, updated.IsBought)

		// And clearing it again must stick. GORM's map-based Updates does not skip
		// false here (unlike struct updates, where false is the zero value and would
		// be dropped) -- this assertion is what guards that distinction.
		clearData := map[string]interface{}{"is_absent": false}
		jsonValue, _ = json.Marshal(clearData)
		req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%s", item.ID.String()), bytes.NewBuffer(jsonValue))
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		database.DB.First(&updated, "id = ?", item.ID)
		assert.False(t, updated.IsAbsent)
	})

	t.Run("GetListSerializesIsAbsent", func(t *testing.T) {
		// Not setupItemTestDBIsolated: GetList does Preload("Receipts"), which
		// errors into a 404 unless the receipts table exists.
		setupLinkAliasDB()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)
		database.DB.Create(&models.Item{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Name:        "Cardamom",
			ListID:      list.ID,
			IsAbsent:    true,
		})

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.GET("/lists/:id", GetList)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%s", list.ID.String()), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Assert on the wire format, not the decoded struct: decoding into
		// models.Item would pass even if the field were dropped from the JSON.
		assert.Contains(t, w.Body.String(), `"is_absent":true`)
	})

	t.Run("BoughtAbsentExclusivity", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.PATCH("/items/:id", UpdateItem)

		newItem := func(name string, bought, absent bool) models.Item {
			it := models.Item{
				TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
				Name:        name,
				ListID:      list.ID,
				IsBought:    bought,
				IsAbsent:    absent,
			}
			database.DB.Create(&it)
			return it
		}
		patch := func(id uuid.UUID, body map[string]interface{}) *httptest.ResponseRecorder {
			jsonValue, _ := json.Marshal(body)
			req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%s", id.String()), bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			return w
		}
		reload := func(id uuid.UUID) models.Item {
			var it models.Item
			database.DB.First(&it, "id = ?", id)
			return it
		}

		t.Run("marking an absent item bought clears absent", func(t *testing.T) {
			it := newItem("Thyme", false, true)
			w := patch(it.ID, map[string]interface{}{"is_bought": true})

			assert.Equal(t, http.StatusOK, w.Code)
			got := reload(it.ID)
			assert.True(t, got.IsBought)
			assert.False(t, got.IsAbsent, "bought must win over absent")
		})

		t.Run("a bought item cannot be marked absent", func(t *testing.T) {
			it := newItem("Basil", true, false)
			w := patch(it.ID, map[string]interface{}{"is_absent": true})

			assert.Equal(t, http.StatusBadRequest, w.Code)
			got := reload(it.ID)
			assert.True(t, got.IsBought, "rejected patch must leave the row untouched")
			assert.False(t, got.IsAbsent)
		})

		t.Run("setting both at once resolves to bought", func(t *testing.T) {
			it := newItem("Oregano", false, false)
			w := patch(it.ID, map[string]interface{}{"is_bought": true, "is_absent": true})

			assert.Equal(t, http.StatusOK, w.Code)
			got := reload(it.ID)
			assert.True(t, got.IsBought)
			assert.False(t, got.IsAbsent)
		})

		t.Run("clearing bought while setting absent is allowed", func(t *testing.T) {
			// The end state (absent, not bought) is legal, so this must not 400
			// even though it names is_absent on a currently-bought item.
			it := newItem("Parsley", true, false)
			w := patch(it.ID, map[string]interface{}{"is_bought": false, "is_absent": true})

			assert.Equal(t, http.StatusOK, w.Code)
			got := reload(it.ID)
			assert.False(t, got.IsBought)
			assert.True(t, got.IsAbsent)
		})

		t.Run("non-boolean flag values are rejected", func(t *testing.T) {
			// Regression: a plain type assertion treated these as "not set", so the
			// exclusivity check was skipped while GORM still wrote the coerced value.
			// SQLite stores is_bought as INTEGER, so 1 became true and the row ended
			// up both bought and absent.
			for _, body := range []map[string]interface{}{
				{"is_bought": 1},
				{"is_absent": 1},
				{"is_bought": "true"},
			} {
				it := newItem("Sage", false, true)
				w := patch(it.ID, body)

				assert.Equal(t, http.StatusBadRequest, w.Code, "body %v must be rejected", body)
				got := reload(it.ID)
				assert.False(t, got.IsBought && got.IsAbsent, "row must never hold both flags")
				assert.True(t, got.IsAbsent, "rejected patch must leave the row untouched")
			}
		})

		t.Run("unrelated patches on a bought item still work", func(t *testing.T) {
			it := newItem("Chives", true, false)
			w := patch(it.ID, map[string]interface{}{"name": "Fresh Chives"})

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "Fresh Chives", reload(it.ID).Name)
		})
	})
}

func setupLinkAliasDB() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	database.DB.AutoMigrate(
		&models.ShoppingList{}, &models.Item{}, &models.Family{},
		&models.Category{}, &models.ItemFrequency{}, &models.ItemAlias{},
		&models.Receipt{}, &models.ReceiptItem{},
	)
}

func linkAliasRouter(familyID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("family_id", familyID)
		c.Next()
	})
	r.POST("/items/link-alias", LinkItemAsAlias)
	return r
}

func TestLinkItemAsAlias_success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	database.DB.Create(&family)
	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Title: "L"}
	database.DB.Create(&list)

	planned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "Курица филе", ListID: list.ID}
	database.DB.Create(&planned)
	receiptItemID := uint(1)
	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "KUŘ. PRSNÍ ŘÍZEK", Price: 166.9, ListID: list.ID, ReceiptItemID: &receiptItemID}
	database.DB.Create(&scanned)

	body := map[string]interface{}{
		"receipt_item_id": scanned.ID.String(),
		"planned_item_id": planned.ID.String(),
	}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	linkAliasRouter(family.ID).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Alias created
	var alias models.ItemAlias
	err := database.DB.Where("family_id = ? AND planned_name = ?", family.ID, "Курица филе").First(&alias).Error
	assert.NoError(t, err)
	assert.Equal(t, "KUŘ. PRSNÍ ŘÍZEK", alias.ReceiptName)
	assert.InDelta(t, 166.9, alias.LastPrice, 0.01)

	// Planned item deleted
	var found models.Item
	err = database.DB.Where("id = ?", planned.ID).First(&found).Error
	assert.Error(t, err, "planned item should be deleted")
}

func TestLinkItemAsAlias_wrongFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family1 := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	family2 := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F2"}}
	database.DB.Create(&family1)
	database.DB.Create(&family2)
	list1 := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family1.ID}, Title: "L1"}
	list2 := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family2.ID}, Title: "L2"}
	database.DB.Create(&list1)
	database.DB.Create(&list2)

	planned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family2.ID}, Name: "Foreign item", ListID: list2.ID}
	database.DB.Create(&planned)
	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family1.ID}, Name: "Receipt item", ListID: list1.ID}
	database.DB.Create(&scanned)

	body := map[string]interface{}{
		"receipt_item_id": scanned.ID.String(),
		"planned_item_id": planned.ID.String(), // belongs to family2, not family1
	}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	linkAliasRouter(family1.ID).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLinkItemAsAlias_freeTextName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	database.DB.Create(&family)
	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Title: "L"}
	database.DB.Create(&list)

	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "MAGNESIA 1,5L", Price: 15.9, ListID: list.ID}
	database.DB.Create(&scanned)

	body := map[string]interface{}{
		"receipt_item_id": scanned.ID.String(),
		"planned_name":    "Минералка",
	}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	linkAliasRouter(family.ID).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var alias models.ItemAlias
	err := database.DB.Where("family_id = ? AND planned_name = ?", family.ID, "Минералка").First(&alias).Error
	assert.NoError(t, err)
	assert.Equal(t, "MAGNESIA 1,5L", alias.ReceiptName)

	// Scanned item NOT deleted
	var item models.Item
	err = database.DB.Where("id = ?", scanned.ID).First(&item).Error
	assert.NoError(t, err)
}

func TestLinkItemAsAlias_bothNilOrBothSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	database.DB.Create(&family)
	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Title: "L"}
	database.DB.Create(&list)
	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "Item", ListID: list.ID}
	database.DB.Create(&scanned)

	r := linkAliasRouter(family.ID)

	// Both nil (only receipt_item_id)
	bodyJSON, _ := json.Marshal(map[string]interface{}{"receipt_item_id": scanned.ID.String()})
	req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Both set
	bodyJSON, _ = json.Marshal(map[string]interface{}{
		"receipt_item_id": scanned.ID.String(),
		"planned_item_id": uuid.New().String(),
		"planned_name":    "something",
	})
	req, _ = http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLinkItemAsAlias_emptyOrWhitespacePlannedName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	database.DB.Create(&family)
	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Title: "L"}
	database.DB.Create(&list)
	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "Item", ListID: list.ID}
	database.DB.Create(&scanned)

	r := linkAliasRouter(family.ID)
	for _, name := range []string{"", "   "} {
		bodyJSON, _ := json.Marshal(map[string]interface{}{
			"receipt_item_id": scanned.ID.String(),
			"planned_name":    name,
		})
		req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "name=%q", name)
	}
}

func TestLinkItemAsAlias_orphanReceipt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLinkAliasDB()

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F1"}}
	database.DB.Create(&family)
	list := models.ShoppingList{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Title: "L"}
	database.DB.Create(&list)

	// ReceiptItemID points to a non-existent ReceiptItem
	orphanID := uint(9999)
	scanned := models.Item{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID}, Name: "Orphan Item", Price: 10.0, ListID: list.ID, ReceiptItemID: &orphanID}
	database.DB.Create(&scanned)

	body := map[string]interface{}{
		"receipt_item_id": scanned.ID.String(),
		"planned_name":    "My Item",
	}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/items/link-alias", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	linkAliasRouter(family.ID).ServeHTTP(w, req)

	// Should succeed (shop_id falls back to nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var alias models.ItemAlias
	err := database.DB.Where("family_id = ? AND planned_name_lower = ?", family.ID, "my item").First(&alias).Error
	assert.NoError(t, err)
	assert.Nil(t, alias.ShopID)
}

func frequentItemsRouter(familyID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("family_id", familyID)
		c.Next()
	})
	r.GET("/family/frequent-items", GetFrequentItems)
	r.GET("/family/frequent-items/hidden", GetHiddenFrequentItems)
	r.DELETE("/family/frequent-items/:id", DeleteFrequentItem)
	r.PATCH("/family/frequent-items/:id/restore", RestoreFrequentItem)
	return r
}

func TestFrequentItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetFrequentItems_FiltersLowFrequency", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		// freq=1 should be excluded; freq=3 should be included
		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "sleva", Frequency: 1})
		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "mleko", Frequency: 3})

		req, _ := http.NewRequest(http.MethodGet, "/family/frequent-items", nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "mleko", result[0]["item_name"])
	})

	t.Run("GetFrequentItems_Limit", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		for i := 0; i < 15; i++ {
			database.DB.Create(&models.ItemFrequency{
				FamilyID:  family.ID,
				ItemName:  fmt.Sprintf("item%d", i),
				Frequency: 2,
			})
		}

		req, _ := http.NewRequest(http.MethodGet, "/family/frequent-items", nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.LessOrEqual(t, len(result), 10)
	})

	t.Run("DeleteFrequentItem_Success", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		freq := models.ItemFrequency{FamilyID: family.ID, ItemName: "chleb", Frequency: 5}
		database.DB.Create(&freq)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/family/frequent-items/%d", freq.ID), nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		var check models.ItemFrequency
		err := database.DB.First(&check, freq.ID).Error
		assert.NoError(t, err, "row should still exist (soft-delete)")
		assert.True(t, check.IsHidden, "row should be hidden")
	})

	t.Run("DeleteFrequentItem_WrongFamily", func(t *testing.T) {
		setupItemTestDBIsolated()
		familyA := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family A"}}
		familyB := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family B"}}
		database.DB.Create(&familyA)
		database.DB.Create(&familyB)

		freq := models.ItemFrequency{FamilyID: familyA.ID, ItemName: "vejce", Frequency: 4}
		database.DB.Create(&freq)

		// authenticate as familyB, try to delete familyA's item
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/family/frequent-items/%d", freq.ID), nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(familyB.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		// item must still exist
		var check models.ItemFrequency
		err := database.DB.First(&check, freq.ID).Error
		assert.NoError(t, err)
	})

	t.Run("GetFrequentItems_ExcludesHidden", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "visible", Frequency: 3})
		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "hidden", Frequency: 5, IsHidden: true})

		req, _ := http.NewRequest(http.MethodGet, "/family/frequent-items", nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "visible", result[0]["item_name"])
	})

	t.Run("GetHiddenFrequentItems_ReturnsHiddenOnly", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "active", Frequency: 3})
		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "gone", Frequency: 5, IsHidden: true})

		req, _ := http.NewRequest(http.MethodGet, "/family/frequent-items/hidden", nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "gone", result[0]["item_name"])
	})

	t.Run("GetHiddenFrequentItems_EmptyWhenNoneHidden", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		database.DB.Create(&models.ItemFrequency{FamilyID: family.ID, ItemName: "normal", Frequency: 3})

		req, _ := http.NewRequest(http.MethodGet, "/family/frequent-items/hidden", nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("RestoreFrequentItem_Success", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)

		freq := models.ItemFrequency{FamilyID: family.ID, ItemName: "mleko", Frequency: 4, IsHidden: true}
		database.DB.Create(&freq)

		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/family/frequent-items/%d/restore", freq.ID), nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(family.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		var check models.ItemFrequency
		database.DB.First(&check, freq.ID)
		assert.False(t, check.IsHidden, "item should be visible again after restore")
	})

	t.Run("RestoreFrequentItem_WrongFamily", func(t *testing.T) {
		setupItemTestDBIsolated()
		familyA := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family A"}}
		familyB := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Family B"}}
		database.DB.Create(&familyA)
		database.DB.Create(&familyB)

		freq := models.ItemFrequency{FamilyID: familyA.ID, ItemName: "syr", Frequency: 3, IsHidden: true}
		database.DB.Create(&freq)

		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/family/frequent-items/%d/restore", freq.ID), nil)
		w := httptest.NewRecorder()
		frequentItemsRouter(familyB.ID).ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		// item must remain hidden
		var check models.ItemFrequency
		database.DB.First(&check, freq.ID)
		assert.True(t, check.IsHidden)
	})

	t.Run("HiddenItem_NotResurrectedByAddItem", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)

		// pre-seed a hidden frequency row
		freq := models.ItemFrequency{FamilyID: family.ID, ItemName: "Butter", Frequency: 5, IsHidden: true}
		database.DB.Create(&freq)

		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
		r.POST("/lists/:id/items", AddItemToList)

		newItem := models.Item{Name: "Butter"}
		body, _ := json.Marshal(newItem)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/items", list.ID.String()), bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// frequency row must still be hidden and counter must not have changed
		var check models.ItemFrequency
		database.DB.First(&check, freq.ID)
		assert.True(t, check.IsHidden, "hidden item should not be un-hidden by adding item")
		assert.Equal(t, 5, check.Frequency, "frequency should not change for hidden item")
	})

	t.Run("HiddenItem_NotResurrectedByCaseVariant", func(t *testing.T) {
		setupItemTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Test List",
		}
		database.DB.Create(&list)

		// hidden as "Milk" (capitalized)
		freq := models.ItemFrequency{FamilyID: family.ID, ItemName: "Milk", Frequency: 3, IsHidden: true}
		database.DB.Create(&freq)

		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
		r.POST("/lists/:id/items", AddItemToList)

		// add lowercase variant
		newItem := models.Item{Name: "milk"}
		body, _ := json.Marshal(newItem)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/items", list.ID.String()), bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// original row must remain hidden, no new row created
		var check models.ItemFrequency
		database.DB.First(&check, freq.ID)
		assert.True(t, check.IsHidden, "case variant must not un-hide the original")

		var count int64
		database.DB.Model(&models.ItemFrequency{}).Where("family_id = ? AND LOWER(item_name) = 'milk'", family.ID).Count(&count)
		assert.Equal(t, int64(1), count, "must not create a second row for the case variant")
	})
}
