package handlers

import (
	"fmt"
	"kincart/internal/database"
	"kincart/internal/models"
)

func validateItemsFamily(items []models.Item, familyID uint) error {
	for _, item := range items {
		if item.CategoryID != 0 {
			var cat models.Category
			if err := database.DB.Where("id = ? AND family_id = ?", item.CategoryID, familyID).First(&cat).Error; err != nil {
				return fmt.Errorf("invalid category ID: %d", item.CategoryID)
			}
		}
	}
	return nil
}
