package handlers

import (
	"fmt"

	"github.com/google/uuid"
	"kincart/internal/database"
	"kincart/internal/models"
)

func validateItemsFamily(items []models.Item, familyID uuid.UUID) error {
	for _, item := range items {
		if item.CategoryID != uuid.Nil {
			var cat models.Category
			if err := database.DB.Where("id = ? AND family_id = ?", item.CategoryID, familyID).First(&cat).Error; err != nil {
				return fmt.Errorf("invalid category ID: %s", item.CategoryID)
			}
		}
	}
	return nil
}
