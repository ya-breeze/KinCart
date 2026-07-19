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

func setupListTestDBIsolated() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	database.DB.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Category{}, &models.Receipt{}, &models.ReceiptItem{}, &models.Shop{})
}

func TestListsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CreateList", func(t *testing.T) {
		setupListTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
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
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "List 1",
		}
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
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "To Delete",
		}
		database.DB.Create(&list)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.DELETE("/lists/:id", DeleteList)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/lists/%s", list.ID.String()), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		var count int64
		database.DB.Model(&models.ShoppingList{}).Where("id = ?", list.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestListShopAssociation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// newShopTestEnv returns a router scoped to a fresh family, that family, and
	// a shop belonging to it.
	newShopTestEnv := func() (*gin.Engine, models.Family, models.Shop) {
		setupListTestDBIsolated()
		family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
		database.DB.Create(&family)
		shop := models.Shop{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Name:        "Corner Store",
		}
		database.DB.Create(&shop)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("family_id", family.ID)
			c.Next()
		})
		r.POST("/lists", CreateList)
		r.PATCH("/lists/:id", UpdateList)
		return r, family, shop
	}

	do := func(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("CreateWithValidShopPersistsIt", func(t *testing.T) {
		r, _, shop := newShopTestEnv()

		w := do(r, http.MethodPost, "/lists",
			fmt.Sprintf(`{"title":"Groceries","shop_id":"%s"}`, shop.ID))

		assert.Equal(t, http.StatusCreated, w.Code)
		var created models.ShoppingList
		json.Unmarshal(w.Body.Bytes(), &created)
		if assert.NotNil(t, created.ShopID) {
			assert.Equal(t, shop.ID, *created.ShopID)
		}

		var stored models.ShoppingList
		database.DB.Where("id = ?", created.ID).First(&stored)
		if assert.NotNil(t, stored.ShopID) {
			assert.Equal(t, shop.ID, *stored.ShopID)
		}
	})

	t.Run("CreateWithForeignShopIsRejected", func(t *testing.T) {
		r, _, _ := newShopTestEnv()

		otherFamily := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Other Family"}}
		database.DB.Create(&otherFamily)
		foreignShop := models.Shop{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: otherFamily.ID},
			Name:        "Their Store",
		}
		database.DB.Create(&foreignShop)

		w := do(r, http.MethodPost, "/lists",
			fmt.Sprintf(`{"title":"Groceries","shop_id":"%s"}`, foreignShop.ID))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var count int64
		database.DB.Model(&models.ShoppingList{}).Count(&count)
		assert.Equal(t, int64(0), count, "list must not be created with a foreign shop")
	})

	t.Run("UpdateWithForeignShopIsRejectedAndShopUnchanged", func(t *testing.T) {
		r, family, shop := newShopTestEnv()

		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Groceries",
			Status:      "preparing",
			ShopID:      &shop.ID,
		}
		database.DB.Create(&list)

		otherFamily := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Other Family"}}
		database.DB.Create(&otherFamily)
		foreignShop := models.Shop{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: otherFamily.ID},
			Name:        "Their Store",
		}
		database.DB.Create(&foreignShop)

		w := do(r, http.MethodPatch, "/lists/"+list.ID.String(),
			fmt.Sprintf(`{"shop_id":"%s"}`, foreignShop.ID))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var stored models.ShoppingList
		database.DB.Where("id = ?", list.ID).First(&stored)
		if assert.NotNil(t, stored.ShopID) {
			assert.Equal(t, shop.ID, *stored.ShopID, "original shop must survive a rejected update")
		}
	})

	t.Run("UpdateWithNullShopClearsItAndKeepsOtherFields", func(t *testing.T) {
		r, family, shop := newShopTestEnv()

		list := models.ShoppingList{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
			Title:       "Groceries",
			Status:      "shopping",
			ShopID:      &shop.ID,
		}
		database.DB.Create(&list)

		w := do(r, http.MethodPatch, "/lists/"+list.ID.String(), `{"shop_id":null}`)

		assert.Equal(t, http.StatusOK, w.Code)
		var stored models.ShoppingList
		database.DB.Where("id = ?", list.ID).First(&stored)
		assert.Nil(t, stored.ShopID, "shop association must be cleared")
		// Guards the full-Save path: fields absent from the request body must not
		// be clobbered by zero values.
		assert.Equal(t, "Groceries", stored.Title)
		assert.Equal(t, "shopping", stored.Status)
	})
}
