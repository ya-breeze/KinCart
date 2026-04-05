package handlers

import (
	"log/slog"
	"net/http"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetCategories(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var categories []models.Category
	if err := database.DB.Where("family_id = ?", familyID).Order("sort_order asc").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func ReorderCategories(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var req []struct {
		ID        uuid.UUID `json:"id"`
		SortOrder int       `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()
	for _, cat := range req {
		if err := tx.Model(&models.Category{}).Where("id = ? AND family_id = ?", cat.ID, familyID).Update("sort_order", cat.SortOrder).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category order"})
			return
		}
	}
	tx.Commit()

	slog.Info("Categories reordered", "family_id", familyID)
	c.JSON(http.StatusOK, gin.H{"message": "Order updated"})
}

func CreateCategory(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category.FamilyID = familyID
	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	slog.Info("Category created", "name", category.Name, "family_id", familyID)
	c.JSON(http.StatusCreated, category)
}

func UpdateCategory(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	categoryIDStr := c.Param("id")

	catID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		slog.Warn("Invalid category ID format", "category_id", categoryIDStr, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var category models.Category
	if err := database.DB.Where("id = ? AND family_id = ?", catID, familyID).First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Save(&category)
	slog.Info("Category updated", "category_id", catID, "family_id", familyID)
	c.JSON(http.StatusOK, category)
}

func DeleteCategory(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	categoryIDStr := c.Param("id")

	catID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		slog.Warn("Invalid category ID format in delete", "category_id", categoryIDStr, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Clear category from items scoped by family
	database.DB.Model(&models.Item{}).
		Where("category_id = ? AND list_id IN (SELECT id FROM shopping_lists WHERE family_id = ?)", catID, familyID).
		Update("category_id", uuid.Nil)

	if err := database.DB.Where("id = ? AND family_id = ?", catID, familyID).Delete(&models.Category{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	slog.Info("Category deleted", "category_id", catID, "family_id", familyID)
	c.Status(http.StatusNoContent)
}
