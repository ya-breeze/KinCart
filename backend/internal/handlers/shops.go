package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
)

func GetShops(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var shops []models.Shop
	if err := database.DB.Where("family_id = ?", familyID).Find(&shops).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shops"})
		return
	}

	c.JSON(http.StatusOK, shops)
}

func CreateShop(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var shop models.Shop
	if err := c.ShouldBindJSON(&shop); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shop.FamilyID = familyID
	if err := database.DB.Create(&shop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create shop"})
		return
	}

	slog.Info("Shop created", "name", shop.Name, "family_id", familyID)
	c.JSON(http.StatusCreated, shop)
}

func UpdateShop(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	shopID := c.Param("id")

	// Validate shop ID
	sID, err := strconv.ParseUint(shopID, 10, 32)
	if err != nil {
		slog.Warn("Invalid shop ID format", "shop_id", shopID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shop ID"})
		return
	}

	var shop models.Shop
	if err := database.DB.Where("id = ? AND family_id = ?", uint(sID), familyID).First(&shop).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shop not found"})
		return
	}

	if err := c.ShouldBindJSON(&shop); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Save(&shop)
	slog.Info("Shop updated", "shop_id", sID, "family_id", familyID)
	c.JSON(http.StatusOK, shop)
}

func DeleteShop(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	shopID := c.Param("id")

	// Validate shop ID
	sID, err := strconv.ParseUint(shopID, 10, 32)
	if err != nil {
		slog.Warn("Invalid shop ID format in delete", "shop_id", shopID, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shop ID"})
		return
	}

	if err := database.DB.Where("id = ? AND family_id = ?", uint(sID), familyID).Delete(&models.Shop{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete shop"})
		return
	}

	// Also delete category orders for this shop
	database.DB.Where("shop_id = ?", uint(sID)).Delete(&models.ShopCategoryOrder{})

	slog.Info("Shop deleted", "shop_id", sID, "family_id", familyID)
	c.Status(http.StatusNoContent)
}

func GetShopCategoryOrder(c *gin.Context) {
	shopID := c.Param("id")

	var orders []models.ShopCategoryOrder
	database.DB.Where("shop_id = ?", shopID).Order("sort_order ASC").Find(&orders)

	c.JSON(http.StatusOK, orders)
}

func SetShopCategoryOrder(c *gin.Context) {
	shopID := c.Param("id")

	var req []struct {
		CategoryID uint `json:"category_id"`
		SortOrder  int  `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Where("shop_id = ?", shopID).Delete(&models.ShopCategoryOrder{})

	for _, item := range req {
		database.DB.Create(&models.ShopCategoryOrder{
			ShopID:     parseID(shopID),
			CategoryID: item.CategoryID,
			SortOrder:  item.SortOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shop category order updated"})
}

func parseID(id string) uint {
	var val uint
	fmt.Sscanf(id, "%d", &val)
	return val
}
