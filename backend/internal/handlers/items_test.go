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
