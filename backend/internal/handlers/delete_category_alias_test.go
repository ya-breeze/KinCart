/** delete-category alias cleanup */
package handlers

import (
	"net/http"
	"net/http/httptest"
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

// Deleting a category must also clear it from purchase history, or ItemAlias.CategoryID
// dangles and the remembered-defaults resolver would prefill a dead id.
func TestDeleteCategory_ClearsAliasCategoryReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.DB.AutoMigrate(
		&models.Family{}, &models.Category{}, &models.ItemAlias{},
		&models.ShoppingList{}, &models.Item{},
	))

	fam := uuid.New()
	require.NoError(t, database.DB.Create(&models.Family{Family: coremodels.Family{ID: fam, Name: "F"}}).Error)
	cat := models.Category{TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: fam}, Name: "Dairy"}
	require.NoError(t, database.DB.Create(&cat).Error)

	alias := models.ItemAlias{
		FamilyID: fam, PlannedName: "milk", PlannedNameLower: "milk",
		ReceiptName: "milk", ReceiptNameLower: "milk",
		CategoryID: &cat.ID, PurchaseCount: 2, LastUsedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&alias).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/categories/"+cat.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: cat.ID.String()}}
	c.Set("family_id", fam)
	DeleteCategory(c)

	// (w.Code is not asserted: c.Status(204) with no body is not flushed through a
	// bare gin test context. The DB effect below is the actual contract.)
	var reloaded models.ItemAlias
	require.NoError(t, database.DB.First(&reloaded, alias.ID).Error)
	assert.Nil(t, reloaded.CategoryID, "the alias category reference must be cleared on delete")
}
