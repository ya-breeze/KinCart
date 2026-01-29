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
		family := models.Family{Name: "Test Family"}
		database.DB.Create(&family)
		list := models.ShoppingList{Title: "Test List", FamilyID: family.ID}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.POST("/lists/:id/items", AddItemToList)

		newItem := models.Item{Name: "Bread"}
		jsonValue, _ := json.Marshal(newItem)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/items", list.ID), bytes.NewBuffer(jsonValue))
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
		family := models.Family{Name: "Test Family"}
		database.DB.Create(&family)
		list := models.ShoppingList{Title: "Test List", FamilyID: family.ID}
		database.DB.Create(&list)
		item := models.Item{Name: "Milk", ListID: list.ID}
		database.DB.Create(&item)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.PATCH("/items/:id", UpdateItem)

		updateData := map[string]interface{}{"name": "Organic Milk", "is_bought": true}
		jsonValue, _ := json.Marshal(updateData)
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%d", item.ID), bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updated models.Item
		database.DB.First(&updated, item.ID)
		assert.Equal(t, "Organic Milk", updated.Name)
		assert.True(t, updated.IsBought)
	})
}
