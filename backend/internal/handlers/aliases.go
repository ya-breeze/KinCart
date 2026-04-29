package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kincart/internal/database"
	"kincart/internal/models"
)

func GetAliases(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	q := strings.TrimSpace(c.Query("q"))

	tx := database.DB.Preload("Shop").Where("family_id = ?", familyID).Order("planned_name ASC, purchase_count DESC")
	if len(q) >= 2 {
		tx = tx.Where("LOWER(planned_name) LIKE ?", strings.ToLower(q)+"%")
	}

	var aliases []models.ItemAlias
	if err := tx.Find(&aliases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch aliases"})
		return
	}

	c.JSON(http.StatusOK, aliases)
}

type createAliasRequest struct {
	PlannedName string  `json:"planned_name" binding:"required"`
	ReceiptName string  `json:"receipt_name" binding:"required"`
	ShopID      *string `json:"shop_id"`
	LastPrice   float64 `json:"last_price"`
}

func CreateAlias(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var req createAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var shopID *uuid.UUID
	if req.ShopID != nil && *req.ShopID != "" {
		parsed, err := uuid.Parse(*req.ShopID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shop_id"})
			return
		}
		// Verify shop belongs to this family
		var shop models.Shop
		if err := database.DB.Where("id = ? AND family_id = ?", parsed, familyID).First(&shop).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Shop not found"})
			return
		}
		shopID = &parsed
	}

	// Upsert: find existing or create
	var alias models.ItemAlias
	q := database.DB.Where("family_id = ? AND LOWER(planned_name) = ? AND LOWER(receipt_name) = ?",
		familyID, strings.ToLower(req.PlannedName), strings.ToLower(req.ReceiptName))
	if shopID != nil {
		q = q.Where("shop_id = ?", *shopID)
	} else {
		q = q.Where("shop_id IS NULL")
	}

	if err := q.First(&alias).Error; err != nil {
		alias = models.ItemAlias{
			FamilyID:      familyID,
			PlannedName:   req.PlannedName,
			ReceiptName:   req.ReceiptName,
			ShopID:        shopID,
			LastPrice:     req.LastPrice,
			PurchaseCount: 1,
			LastUsedAt:    time.Now(),
		}
		if err := database.DB.Create(&alias).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alias"})
			return
		}
	} else {
		alias.PurchaseCount++
		if req.LastPrice > 0 {
			alias.LastPrice = req.LastPrice
		}
		alias.LastUsedAt = time.Now()
		if err := database.DB.Save(&alias).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alias"})
			return
		}
	}

	database.DB.Preload("Shop").First(&alias, alias.ID)
	c.JSON(http.StatusOK, alias)
}

func DeleteAlias(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alias id"})
		return
	}

	var alias models.ItemAlias
	if err := database.DB.Where("id = ? AND family_id = ?", id, familyID).First(&alias).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alias not found"})
		return
	}

	if err := database.DB.Delete(&alias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alias"})
		return
	}

	c.Status(http.StatusNoContent)
}
