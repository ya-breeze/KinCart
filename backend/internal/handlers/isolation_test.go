package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kincart/internal/database"
	"kincart/internal/models"
)

func setupTestDB() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	database.DB.AutoMigrate(
		&models.Family{},
		&models.User{},
		&models.ShoppingList{},
		&models.Item{},
		&models.Category{},
		&models.Shop{},
		&models.ShopCategoryOrder{},
	)
}

func TestDataIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB()

	// 1. Create two families
	familyA := models.Family{Name: "FamilyA"}
	familyB := models.Family{Name: "FamilyB"}
	database.DB.Create(&familyA)
	database.DB.Create(&familyB)

	// 2. Create users
	userA := models.User{Username: "userA", FamilyID: familyA.ID}
	userB := models.User{Username: "userB", FamilyID: familyB.ID}
	database.DB.Create(&userA)
	database.DB.Create(&userB)

	// 3. Create a category in Family B
	catB := models.Category{Name: "CatB", FamilyID: familyB.ID}
	database.DB.Create(&catB)

	// 4. Create a shop in Family B
	shopB := models.Shop{Name: "ShopB", FamilyID: familyB.ID}
	database.DB.Create(&shopB)

	// 5. Create a list in Family A
	listA := models.ShoppingList{Title: "ListA", FamilyID: familyA.ID}
	database.DB.Create(&listA)

	t.Run("AddItemToList - Cross-family Category Leak", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(int(listA.ID))}}

		// Try to use catB in listA
		body := map[string]interface{}{
			"name":        "Milk",
			"category_id": catB.ID,
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/lists/1/items", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		AddItemToList(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid category ID")
	})

	t.Run("GetShopCategoryOrder - Cross-family Shop Access", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(int(shopB.ID))}}

		GetShopCategoryOrder(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Shop not found")
	})

	t.Run("SetShopCategoryOrder - Cross-family Shop Access", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(int(shopB.ID))}}

		body := []map[string]interface{}{}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/shops/1/categories/order", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		SetShopCategoryOrder(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("SetShopCategoryOrder - Cross-family Category Leak", func(t *testing.T) {
		// Create a shop in Family A
		shopA := models.Shop{Name: "ShopA", FamilyID: familyA.ID}
		database.DB.Create(&shopA)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(int(shopA.ID))}}

		// Try to use catB in shopA order
		body := []map[string]interface{}{
			{"category_id": catB.ID, "sort_order": 1},
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/shops/2/categories/order", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		SetShopCategoryOrder(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid category ID")
	})

	t.Run("CreateList - Cross-family Category Leak via Nested Items", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)

		// Try to create a list with an item pointing to catB (belonging to familyB)
		body := map[string]interface{}{
			"title": "New List",
			"items": []map[string]interface{}{
				{"name": "Stolen Item", "category_id": catB.ID},
			},
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/lists", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		CreateList(c)

		// If it's vulnerable, it will return 201 Created
		// If it's secure, it should return 400 Bad Request
		assert.NotEqual(t, http.StatusCreated, w.Code, "Should not allow creating items with categories from other families")
	})

	t.Run("UpdateItem - Cross-family Category Leak", func(t *testing.T) {
		// Create an item in Family A
		itemA := models.Item{Name: "ItemA", ListID: listA.ID}
		database.DB.Create(&itemA)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(int(itemA.ID))}}

		// Try to update itemA to use catB
		body := map[string]interface{}{
			"category_id": catB.ID,
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("PATCH", "/items/1", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateItem(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid category ID")
	})
}
