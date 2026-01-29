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

func setupListTestDBIsolated() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	database.DB.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Category{}, &models.Receipt{}, &models.ReceiptItem{})
}

func TestListsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CreateList", func(t *testing.T) {
		setupListTestDBIsolated()
		family := models.Family{Name: "Test Family"}
		database.DB.Create(&family)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.POST("/lists", CreateList)

		newList := models.ShoppingList{Title: "New List"}
		jsonValue, _ := json.Marshal(newList)
		req, _ := http.NewRequest(http.MethodPost, "/lists", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var created models.ShoppingList
		json.Unmarshal(w.Body.Bytes(), &created)
		assert.Equal(t, "New List", created.Title)
		assert.Equal(t, family.ID, created.FamilyID)
	})

	t.Run("GetLists", func(t *testing.T) {
		setupListTestDBIsolated()
		family := models.Family{Name: "Test Family"}
		database.DB.Create(&family)
		list := models.ShoppingList{Title: "List 1", FamilyID: family.ID}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.GET("/lists", GetLists)

		req, _ := http.NewRequest(http.MethodGet, "/lists", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var lists []models.ShoppingList
		json.Unmarshal(w.Body.Bytes(), &lists)
		assert.NotEmpty(t, lists)
	})

	t.Run("DeleteList", func(t *testing.T) {
		setupListTestDBIsolated()
		family := models.Family{Name: "Test Family"}
		database.DB.Create(&family)
		list := models.ShoppingList{Title: "To Delete", FamilyID: family.ID}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.DELETE("/lists/:id", DeleteList)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/lists/%d", list.ID), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		var count int64
		database.DB.Model(&models.ShoppingList{}).Where("id = ?", list.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
