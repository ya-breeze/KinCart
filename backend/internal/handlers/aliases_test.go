package handlers

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

func setupAliasTestDB() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	database.DB.AutoMigrate(
		&models.Family{},
		&models.User{},
		&models.Shop{},
		&models.ItemAlias{},
	)
}

func TestAliasIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAliasTestDB()

	familyA := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "FamilyA"}}
	familyB := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "FamilyB"}}
	database.DB.Create(&familyA)
	database.DB.Create(&familyB)

	shopB := models.Shop{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyB.ID},
		Name:        "ShopB",
	}
	database.DB.Create(&shopB)

	// Seed: FamilyB has an alias
	aliasB := models.ItemAlias{
		FamilyID:      familyB.ID,
		PlannedName:   "jogurt",
		ReceiptName:   "selský jogurt 1%",
		LastPrice:     29.90,
		PurchaseCount: 5,
		LastUsedAt:    time.Now(),
	}
	database.DB.Create(&aliasB)

	t.Run("GetAliases - FamilyA cannot see FamilyB aliases", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Request = httptest.NewRequest("GET", "/api/family/aliases", nil)

		GetAliases(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []models.ItemAlias
		json.NewDecoder(w.Body).Decode(&result)
		assert.Empty(t, result, "FamilyA should not see FamilyB's aliases")
	})

	t.Run("GetItemSuggestions - FamilyA cannot see FamilyB aliases", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Request = httptest.NewRequest("GET", "/api/family/item-suggestions?q=jog", nil)

		GetItemSuggestions(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []itemSuggestion
		json.NewDecoder(w.Body).Decode(&result)
		assert.Empty(t, result, "FamilyA should not see FamilyB's aliases as suggestions")
	})

	t.Run("DeleteAlias - FamilyA cannot delete FamilyB alias", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasB.ID)}}
		c.Request = httptest.NewRequest("DELETE", "/api/family/aliases/"+fmt.Sprintf("%d", aliasB.ID), nil)

		DeleteAlias(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		// Confirm alias still exists
		var count int64
		database.DB.Model(&models.ItemAlias{}).Where("id = ?", aliasB.ID).Count(&count)
		assert.Equal(t, int64(1), count, "FamilyB's alias must not be deleted by FamilyA")
	})

	t.Run("CreateAlias - FamilyA cannot attach FamilyB's shop", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)

		shopBIDStr := shopB.ID.String()
		body := createAliasRequest{
			PlannedName: "mléko",
			ReceiptName: "Tatra polotučné 1l",
			ShopID:      &shopBIDStr,
			LastPrice:   25.90,
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/api/family/aliases", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		CreateAlias(c)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject shop belonging to another family")

		// Confirm no alias was created for FamilyA
		var count int64
		database.DB.Model(&models.ItemAlias{}).Where("family_id = ? AND planned_name = ?", familyA.ID, "mléko").Count(&count)
		assert.Equal(t, int64(0), count, "No alias must be created with cross-family shop")
	})

	t.Run("CreateAlias - FamilyA alias is stored under FamilyA only", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)

		body := createAliasRequest{
			PlannedName: "chléb",
			ReceiptName: "Chléb konzumní 500g",
			LastPrice:   32.00,
		}
		jsonBody, _ := json.Marshal(body)
		c.Request = httptest.NewRequest("POST", "/api/family/aliases", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		CreateAlias(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// Confirm alias is only in FamilyA
		var aliasA models.ItemAlias
		database.DB.Where("family_id = ? AND planned_name = ?", familyA.ID, "chléb").First(&aliasA)
		assert.Equal(t, familyA.ID, aliasA.FamilyID)

		// Confirm FamilyB still has only its original alias
		var countB int64
		database.DB.Model(&models.ItemAlias{}).Where("family_id = ?", familyB.ID).Count(&countB)
		assert.Equal(t, int64(1), countB, "FamilyB should still have exactly its original alias")
	})
}

func TestUpdateAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	database.DB.AutoMigrate(&models.Family{}, &models.Shop{}, &models.ItemAlias{})

	familyA := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "FamilyA"}}
	familyB := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "FamilyB"}}
	database.DB.Create(&familyA)
	database.DB.Create(&familyB)

	shopA := models.Shop{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyA.ID},
		Name:        "ShopA",
	}
	shopB := models.Shop{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyB.ID},
		Name:        "ShopB",
	}
	database.DB.Create(&shopA)
	database.DB.Create(&shopB)

	aliasA := models.ItemAlias{
		FamilyID:      familyA.ID,
		PlannedName:   "jogurt",
		ReceiptName:   "selský jogurt 1%",
		LastPrice:     29.90,
		PurchaseCount: 3,
		LastUsedAt:    time.Now(),
	}
	aliasB := models.ItemAlias{
		FamilyID:      familyB.ID,
		PlannedName:   "mléko",
		ReceiptName:   "Tatra polotučné 1l",
		LastPrice:     25.00,
		PurchaseCount: 1,
		LastUsedAt:    time.Now(),
	}
	database.DB.Create(&aliasA)
	database.DB.Create(&aliasB)

	t.Run("UpdateAlias - happy path: update receipt_name and price", func(t *testing.T) {
		newName := "selský jogurt 2%"
		newPrice := 31.90
		body := updateAliasRequest{ReceiptName: &newName, LastPrice: &newPrice}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasA.ID)}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/"+fmt.Sprintf("%d", aliasA.ID), bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var result models.ItemAlias
		json.NewDecoder(w.Body).Decode(&result)
		assert.Equal(t, "selský jogurt 2%", result.ReceiptName)
		assert.InDelta(t, 31.90, result.LastPrice, 0.01)

		// Verify persisted in DB
		var persisted models.ItemAlias
		database.DB.First(&persisted, aliasA.ID)
		assert.Equal(t, "selský jogurt 2%", persisted.ReceiptName)
	})

	t.Run("UpdateAlias - attach own-family shop", func(t *testing.T) {
		shopIDStr := shopA.ID.String()
		body := updateAliasRequest{ShopID: &shopIDStr}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasA.ID)}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/"+fmt.Sprintf("%d", aliasA.ID), bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var persisted models.ItemAlias
		database.DB.First(&persisted, aliasA.ID)
		assert.NotNil(t, persisted.ShopID)
		assert.Equal(t, shopA.ID, *persisted.ShopID)
	})

	t.Run("UpdateAlias - FamilyA cannot update FamilyB alias", func(t *testing.T) {
		newName := "hacked"
		body := updateAliasRequest{ReceiptName: &newName}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasB.ID)}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/"+fmt.Sprintf("%d", aliasB.ID), bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		// FamilyB alias unchanged
		var persisted models.ItemAlias
		database.DB.First(&persisted, aliasB.ID)
		assert.Equal(t, "Tatra polotučné 1l", persisted.ReceiptName)
	})

	t.Run("UpdateAlias - FamilyA cannot attach FamilyB shop", func(t *testing.T) {
		shopBIDStr := shopB.ID.String()
		body := updateAliasRequest{ShopID: &shopBIDStr}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasA.ID)}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/"+fmt.Sprintf("%d", aliasA.ID), bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Shop on aliasA is still ShopA (from previous test), not ShopB
		var persisted models.ItemAlias
		database.DB.First(&persisted, aliasA.ID)
		assert.NotEqual(t, shopB.ID, persisted.ShopID)
	})

	t.Run("UpdateAlias - clear shop by passing empty string", func(t *testing.T) {
		emptyShop := ""
		body := updateAliasRequest{ShopID: &emptyShop}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", aliasA.ID)}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/"+fmt.Sprintf("%d", aliasA.ID), bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var persisted models.ItemAlias
		database.DB.First(&persisted, aliasA.ID)
		assert.Nil(t, persisted.ShopID, "ShopID should be cleared")
	})

	t.Run("UpdateAlias - invalid id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("family_id", familyA.ID)
		c.Params = []gin.Param{{Key: "id", Value: "not-a-number"}}
		c.Request = httptest.NewRequest("PATCH", "/api/family/aliases/not-a-number", bytes.NewBuffer([]byte(`{}`)))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateAlias(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
