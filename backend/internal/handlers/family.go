package handlers

import (
	"net/http"

	"kincart/internal/database"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
)

func GetFamilyConfig(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var family models.Family
	if err := database.DB.Where("id = ?", familyID).First(&family).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Family not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":     family.Name,
		"currency": family.Currency,
	})
}

func UpdateFamilyConfig(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var req struct {
		Currency string `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.Family{}).Where("id = ?", familyID).Update("currency", req.Currency).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update currency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated"})
}
