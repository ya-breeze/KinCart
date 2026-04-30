package handlers

// Regression tests for two systemic bugs fixed in this codebase:
//
// 1. Zero-UUID TenantModel: handlers that called database.DB.Create() without
//    first setting TenantModel.ID = uuid.New() and TenantModel.FamilyID = familyID.
//    The first insert succeeded with a zero-UUID primary key; every subsequent
//    insert failed with "UNIQUE constraint failed".
//
// 2. GORM zero-value delete/save: handlers that called db.Delete(&loadedModel) or
//    db.Save(&loadedModel) when the model's primary key was a zero UUID. GORM
//    detected the zero value and dropped the id condition from the WHERE clause,
//    causing AllowGlobalUpdate=false to block the query → 500.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kincart/internal/database"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"
)

func setupRegressionTestDB() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	database.DB.AutoMigrate(
		&models.Family{},
		&models.ShoppingList{},
		&models.Item{},
		&models.Category{},
		&models.Shop{},
		&models.Receipt{},
		&models.ReceiptItem{},
		&models.ItemAlias{},
		&models.ItemFrequency{},
	)
}

// ── Zero-UUID TenantModel regression ─────────────────────────────────────────

func TestCreateList_AssignsUniqueNonZeroIDs(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.POST("/lists", CreateList)

	var ids [2]uuid.UUID
	for i, title := range []string{"First List", "Second List"} {
		body, _ := json.Marshal(map[string]string{"title": title})
		req, _ := http.NewRequest(http.MethodPost, "/lists", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "create #%d should succeed, got: %s", i+1, w.Body.String())
		var list models.ShoppingList
		json.Unmarshal(w.Body.Bytes(), &list)
		assert.NotEqual(t, uuid.Nil, list.TenantModel.ID, "list #%d must not have zero UUID", i+1)
		assert.Equal(t, family.ID, list.TenantModel.FamilyID)
		ids[i] = list.TenantModel.ID
	}
	assert.NotEqual(t, ids[0], ids[1], "both lists must have distinct IDs")
}

func TestAddItem_AssignsUniqueNonZeroIDs(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Test List",
	}
	database.DB.Create(&list)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.POST("/lists/:id/items", AddItemToList)

	var ids [2]uuid.UUID
	for i, name := range []string{"Apples", "Milk"} {
		body, _ := json.Marshal(map[string]string{"name": name})
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/items", list.ID), bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "add item #%d should succeed, got: %s", i+1, w.Body.String())
		var item models.Item
		json.Unmarshal(w.Body.Bytes(), &item)
		assert.NotEqual(t, uuid.Nil, item.TenantModel.ID, "item #%d must not have zero UUID", i+1)
		assert.Equal(t, family.ID, item.TenantModel.FamilyID)
		ids[i] = item.TenantModel.ID
	}
	assert.NotEqual(t, ids[0], ids[1], "both items must have distinct IDs")
}

func TestCreateCategory_AssignsUniqueNonZeroIDs(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.POST("/categories", CreateCategory)

	var ids [2]uuid.UUID
	for i, name := range []string{"Dairy", "Produce"} {
		body, _ := json.Marshal(map[string]string{"name": name})
		req, _ := http.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "create category #%d should succeed, got: %s", i+1, w.Body.String())
		var cat models.Category
		json.Unmarshal(w.Body.Bytes(), &cat)
		assert.NotEqual(t, uuid.Nil, cat.TenantModel.ID, "category #%d must not have zero UUID", i+1)
		assert.Equal(t, family.ID, cat.TenantModel.FamilyID)
		ids[i] = cat.TenantModel.ID
	}
	assert.NotEqual(t, ids[0], ids[1], "both categories must have distinct IDs")
}

func TestCreateShop_AssignsUniqueNonZeroIDs(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.POST("/shops", CreateShop)

	var ids [2]uuid.UUID
	for i, name := range []string{"Lidl", "Billa"} {
		body, _ := json.Marshal(map[string]string{"name": name})
		req, _ := http.NewRequest(http.MethodPost, "/shops", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "create shop #%d should succeed, got: %s", i+1, w.Body.String())
		var shop models.Shop
		json.Unmarshal(w.Body.Bytes(), &shop)
		assert.NotEqual(t, uuid.Nil, shop.TenantModel.ID, "shop #%d must not have zero UUID", i+1)
		assert.Equal(t, family.ID, shop.TenantModel.FamilyID)
		ids[i] = shop.TenantModel.ID
	}
	assert.NotEqual(t, ids[0], ids[1], "both shops must have distinct IDs")
}

// ── GORM zero-value delete regression ────────────────────────────────────────

func TestDeleteList_WorksWithZeroUUIDRecord(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)

	// Simulate the pre-fix state: a list whose primary key is the zero UUID.
	// We bypass GORM's Create (which has the same zero-value issue) and use raw SQL.
	database.DB.Exec(
		"INSERT INTO shopping_lists (id, family_id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))",
		uuid.Nil.String(), family.ID.String(), "Poisoned", "preparing",
	)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.DELETE("/lists/:id", DeleteList)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/lists/%s", uuid.Nil), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "delete zero-UUID list should return 204, got: %s", w.Body.String())

	var count int64
	database.DB.Model(&models.ShoppingList{}).Where("id = ?", uuid.Nil.String()).Count(&count)
	assert.Equal(t, int64(0), count, "zero-UUID list should be gone after delete")
}

func TestDeleteItem_WorksWithZeroUUIDRecord(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Test List",
	}
	database.DB.Create(&list)

	// Simulate a poisoned item with zero UUID but a valid list_id.
	database.DB.Exec(
		"INSERT INTO items (id, family_id, name, list_id, quantity, unit, is_bought, is_urgent, price, created_at, updated_at) VALUES (?, ?, ?, ?, 1, 'pcs', 0, 0, 0, datetime('now'), datetime('now'))",
		uuid.Nil.String(), uuid.Nil.String(), "Poisoned Item", list.ID.String(),
	)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.DELETE("/items/:id", DeleteItem)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/items/%s", uuid.Nil), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "delete zero-UUID item should return 204, got: %s", w.Body.String())

	var count int64
	database.DB.Model(&models.Item{}).Where("id = ?", uuid.Nil.String()).Count(&count)
	assert.Equal(t, int64(0), count, "zero-UUID item should be gone after delete")
}

// ── GORM zero-value save regression ──────────────────────────────────────────

func TestUpdateList_PersiststsAndReturnsPersisted(t *testing.T) {
	// Regression: UpdateList previously called database.DB.Save(&list) without checking
	// the error. On DB failure the handler silently returned 200 with stale in-memory data.
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Original",
		Status:      "preparing",
	}
	database.DB.Create(&list)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.PATCH("/lists/:id", UpdateList)

	body, _ := json.Marshal(map[string]string{"title": "Updated", "status": "preparing"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/lists/%s", list.ID), bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "update should succeed: %s", w.Body.String())

	// Verify the change is actually in the DB, not just returned from in-memory struct
	var fresh models.ShoppingList
	database.DB.Where("id = ?", list.ID).First(&fresh)
	assert.Equal(t, "Updated", fresh.Title, "update must be persisted to DB")
}

func TestUpdateList_CannotOverwriteTenantFields(t *testing.T) {
	// ShouldBindJSON(&list) would overwrite TenantModel.ID and FamilyID if the
	// client sends them in the body. The handler must restore them after binding.
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Original",
		Status:      "preparing",
	}
	database.DB.Create(&list)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.PATCH("/lists/:id", UpdateList)

	// Attempt to overwrite id and family_id via the request body
	body, _ := json.Marshal(map[string]interface{}{
		"title":     "Hijacked",
		"id":        uuid.Nil.String(),
		"family_id": uuid.Nil.String(),
		"status":    "preparing",
	})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/lists/%s", list.ID), bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var fresh models.ShoppingList
	database.DB.Where("id = ?", list.ID).First(&fresh)
	assert.Equal(t, list.ID, fresh.TenantModel.ID, "id must not be overwritten by client")
	assert.Equal(t, family.ID, fresh.TenantModel.FamilyID, "family_id must not be overwritten by client")
	assert.Equal(t, "Hijacked", fresh.Title) // title change is fine
}

// ── Happy-path tests for previously uncovered handlers ────────────────────────

func TestUpdateCategory_UpdatesAndReturns200(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	cat := models.Category{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Name:        "Old Name",
	}
	database.DB.Create(&cat)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.PATCH("/categories/:id", UpdateCategory)

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/categories/%s", cat.ID), bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updated models.Category
	json.Unmarshal(w.Body.Bytes(), &updated)
	assert.Equal(t, "New Name", updated.Name)
}

func TestUpdateShop_UpdatesAndReturns200(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	shop := models.Shop{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Name:        "Old Shop",
	}
	database.DB.Create(&shop)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.PATCH("/shops/:id", UpdateShop)

	body, _ := json.Marshal(map[string]string{"name": "New Shop"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/shops/%s", shop.ID), bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updated models.Shop
	json.Unmarshal(w.Body.Bytes(), &updated)
	assert.Equal(t, "New Shop", updated.Name)
}

func TestDeleteAlias_Returns204(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	alias := models.ItemAlias{
		FamilyID:      family.ID,
		PlannedName:   "milk",
		ReceiptName:   "whole milk 1L",
		PurchaseCount: 3,
		LastUsedAt:    time.Now(),
	}
	database.DB.Create(&alias)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.DELETE("/family/aliases/:id", DeleteAlias)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/family/aliases/%d", alias.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	var count int64
	database.DB.Model(&models.ItemAlias{}).Where("id = ?", alias.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestUpdateAlias_UpdatesAndReturns200(t *testing.T) {
	setupRegressionTestDB()
	gin.SetMode(gin.TestMode)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "F"}}
	database.DB.Create(&family)
	alias := models.ItemAlias{
		FamilyID:      family.ID,
		PlannedName:   "yogurt",
		ReceiptName:   "greek yogurt 500g",
		LastPrice:     3.50,
		PurchaseCount: 2,
		LastUsedAt:    time.Now(),
	}
	database.DB.Create(&alias)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("family_id", family.ID); c.Next() })
	r.PATCH("/family/aliases/:id", UpdateAlias)

	newPrice := 4.20
	body, _ := json.Marshal(map[string]interface{}{"last_price": newPrice})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/family/aliases/%d", alias.ID), bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updated models.ItemAlias
	database.DB.First(&updated, alias.ID)
	assert.InDelta(t, newPrice, updated.LastPrice, 0.001)
}
