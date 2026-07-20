package handlers

import (
	"fmt"

	"github.com/google/uuid"

	"kincart/internal/database"
	"kincart/internal/models"
)

// validateShopFamily checks that a non-null shop reference belongs to the
// family. A nil shopID means "no shop" and is always valid.
func validateShopFamily(shopID *uuid.UUID, familyID uuid.UUID) error {
	if shopID == nil {
		return nil
	}
	var shop models.Shop
	if err := database.DB.Where("id = ? AND family_id = ?", *shopID, familyID).First(&shop).Error; err != nil {
		return fmt.Errorf("invalid shop ID: %s", *shopID)
	}
	return nil
}

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
