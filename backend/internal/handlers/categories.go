package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var categories []models.Category
	if err := database.DB.Where("family_id = ?", familyID).Order("sort_order asc").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func ReorderCategories(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var req []struct {
		ID        uint `json:"id"`
		SortOrder int  `json:"sort_order"`
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
	familyID := c.MustGet("family_id").(uint)

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
	familyID := c.MustGet("family_id").(uint)
	categoryID := c.Param("id")

	// Validate category ID
	catID, err := strconv.ParseUint(categoryID, 10, 32)
	if err != nil {
		slog.Warn("Invalid category ID format", "category_id", categoryID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var category models.Category
	if err := database.DB.Where("id = ? AND family_id = ?", uint(catID), familyID).First(&category).Error; err != nil {
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
	familyID := c.MustGet("family_id").(uint)
	categoryID := c.Param("id")

	// Validate category ID to prevent SQL injection
	catID, err := strconv.ParseUint(categoryID, 10, 32)
	if err != nil {
		slog.Warn("Invalid category ID format in delete", "category_id", categoryID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Set items that use this category to null or uncategorized
	database.DB.Model(&models.Item{}).Where("category_id = ?", uint(catID)).Update("category_id", 0)

	if err := database.DB.Where("id = ? AND family_id = ?", uint(catID), familyID).Delete(&models.Category{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	slog.Info("Category deleted", "category_id", catID, "family_id", familyID)
	c.Status(http.StatusNoContent)
}
