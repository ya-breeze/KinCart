package handlers

import (
	"net/http"
	"time"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ya-breeze/kin-core/db"
	coremodels "github.com/ya-breeze/kin-core/models"
	"gorm.io/gorm"
)

func GetLists(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var lists []models.ShoppingList
	if err := database.DB.Preload("Receipts").Scopes(db.Scope(familyID)).Order("created_at desc").Find(&lists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lists"})
		return
	}

	c.JSON(http.StatusOK, lists)
}

func GetList(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	listID := c.Param("id")

	var list models.ShoppingList
	if err := database.DB.Preload("Items").Preload("Receipts").Preload("Receipts.Items").Preload("Receipts.Shop").Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	c.JSON(http.StatusOK, list)
}

func CreateList(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var list models.ShoppingList
	if err := c.ShouldBindJSON(&list); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	list.TenantModel.ID = uuid.New()
	list.TenantModel.FamilyID = familyID
	list.Status = "preparing"

	if err := validateItemsFamily(list.Items, familyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create list"})
		return
	}

	c.JSON(http.StatusCreated, list)
}

func UpdateList(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	listID := c.Param("id")

	var list models.ShoppingList
	if err := database.DB.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	if err := c.ShouldBindJSON(&list); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateItemsFamily(list.Items, familyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle completion status change
	if list.Status == "completed" && (list.CompletedAt == nil) {
		now := time.Now()
		list.CompletedAt = &now
	} else if list.Status != "completed" {
		list.CompletedAt = nil
	}

	if err := database.DB.Where("id = ?", listID).Save(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update list"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func DuplicateList(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	listID := c.Param("id")

	var originalList models.ShoppingList
	if err := database.DB.Preload("Items").Where("id = ? AND family_id = ?", listID, familyID).First(&originalList).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	newList := models.ShoppingList{
		TenantModel:     coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		Title:           originalList.Title + " (Copy)",
		Status:          "preparing",
		EstimatedAmount: originalList.EstimatedAmount,
	}

	if err := database.DB.Create(&newList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new list"})
		return
	}

	for _, item := range originalList.Items {
		newItem := models.Item{
			TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			CategoryID:  item.CategoryID,
			ListID:      newList.ID,
			IsBought:    false,
			IsUrgent:    item.IsUrgent,
			FlyerItemID: item.FlyerItemID,
		}
		database.DB.Create(&newItem)
	}

	c.JSON(http.StatusCreated, newList)
}

func DeleteList(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	listID := c.Param("id")

	var list models.ShoppingList
	if err := database.DB.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Delete list items first
		if err := tx.Where("list_id = ?", list.ID).Delete(&models.Item{}).Error; err != nil {
			return err
		}

		// Unlink receipts (set list_id to NULL)
		if err := tx.Model(&models.Receipt{}).Where("list_id = ?", list.ID).Update("list_id", nil).Error; err != nil {
			return err
		}

		// Delete the list — use explicit WHERE string to avoid GORM skipping zero UUID primary key
		return tx.Where("id = ?", listID).Delete(&models.ShoppingList{}).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete list"})
		return
	}

	c.Status(http.StatusNoContent)
}
