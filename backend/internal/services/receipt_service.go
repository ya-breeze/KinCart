package services

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"kincart/internal/ai"
	"kincart/internal/models"
)

type ReceiptParser interface {
	ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
}

type ReceiptService struct {
	db           *gorm.DB
	gemini       ReceiptParser
	fileStorage  *FileStorageService
	receiptsPath string // Base path for reading files for Gemini (absolute or relative to cwd)
}

func NewReceiptService(db *gorm.DB, gemini ReceiptParser, fileStorage *FileStorageService, receiptsPath string) *ReceiptService {
	return &ReceiptService{
		db:           db,
		gemini:       gemini,
		fileStorage:  fileStorage,
		receiptsPath: receiptsPath,
	}
}

func (s *ReceiptService) CreateReceipt(familyID uint, file *multipart.FileHeader) (*models.Receipt, error) {
	// Save file
	path, err := s.fileStorage.SaveReceipt(familyID, file)
	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	receipt := models.Receipt{
		FamilyID:  familyID,
		ImagePath: path,
		Date:      time.Now(), // Default to now, will be updated by parser
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		return nil, err
	}

	return &receipt, nil
}

func (s *ReceiptService) ProcessReceipt(ctx context.Context, receiptID uint, listID uint) error {
	var receipt models.Receipt
	if err := s.db.Preload("Items").First(&receipt, receiptID).Error; err != nil {
		return fmt.Errorf("receipt not found: %w", err)
	}

	// Check if Gemini is available
	if s.gemini == nil {
		return fmt.Errorf("gemini client not available")
	}

	// 1. Get List Items for context
	var listItems []models.Item
	if err := s.db.Where("list_id = ?", listID).Find(&listItems).Error; err != nil {
		return fmt.Errorf("failed to fetch list items: %w", err)
	}

	knownItemNames := make([]string, len(listItems))
	for i, item := range listItems {
		knownItemNames[i] = item.Name
	}

	// 2. Parse with Gemini
	// Construct full path for Gemini
	fullPath := filepath.Join(s.receiptsPath, receipt.ImagePath)
	parsed, err := s.gemini.ParseReceipt(ctx, fullPath, knownItemNames)
	if err != nil {
		// Update status to error
		s.db.Model(&receipt).Update("status", "error")
		return fmt.Errorf("gemini parsing failed: %w", err)
	}

	// 3. Transaction to update DB
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Update Receipt Info
		date, _ := time.Parse("2006-01-02", parsed.Date) // Ignore error, default to zero time
		receipt.Date = date
		receipt.Total = parsed.Total
		receipt.Status = "parsed"

		// Find/Create Shop
		if parsed.StoreName != "" {
			shopID, err := s.findOrCreateShop(tx, receipt.FamilyID, parsed.StoreName)
			if err != nil {
				return err
			}
			receipt.ShopID = shopID
		}

		if err := tx.Save(&receipt).Error; err != nil {
			return err
		}

		// Update Shopping List Title with Date
		s.updateListTitle(tx, listID, receipt.Date)

		// Process Items
		if err := s.processReceiptItems(tx, receipt.ID, listID, receipt.FamilyID, parsed.Items, listItems); err != nil {
			return err
		}

		// Recalculate List Total
		return s.recalculateListTotal(tx, listID, receipt.FamilyID)
	})
}

func (s *ReceiptService) findOrCreateShop(tx *gorm.DB, familyID uint, storeName string) (*uint, error) {
	var shop models.Shop
	if err := tx.Where("LOWER(name) = ? AND family_id = ?", strings.ToLower(storeName), familyID).First(&shop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			shop = models.Shop{
				Name:     storeName,
				FamilyID: familyID,
			}
			if createErr := tx.Create(&shop).Error; createErr != nil {
				return nil, createErr
			}
		} else {
			return nil, err
		}
	}
	return &shop.ID, nil
}

func (s *ReceiptService) updateListTitle(tx *gorm.DB, listID uint, date time.Time) {
	if !date.IsZero() {
		var list models.ShoppingList
		if err := tx.First(&list, listID).Error; err == nil {
			dateSuffix := fmt.Sprintf(" (%s)", date.Format("2006-01-02"))
			if !strings.Contains(list.Title, dateSuffix) {
				list.Title += dateSuffix
				if err := tx.Save(&list).Error; err != nil {
					slog.Warn("Failed to update list title with date", "error", err)
				}
			}
		}
	}
}

func (s *ReceiptService) processReceiptItems(tx *gorm.DB, receiptID uint, listID uint, familyID uint, parsedItems []ai.ParsedReceiptItem, listItems []models.Item) error {
	for _, parsedItem := range parsedItems {
		// Create ReceiptItem
		receiptItem := models.ReceiptItem{
			ReceiptID:  receiptID,
			Name:       parsedItem.Name,
			Quantity:   parsedItem.Quantity,
			Unit:       parsedItem.Unit,
			Price:      parsedItem.Price,
			TotalPrice: parsedItem.TotalPrice,
		}
		if err := tx.Create(&receiptItem).Error; err != nil {
			return err
		}

		// Match with List Item
		matched := false
		for _, item := range listItems {
			if strings.EqualFold(item.Name, parsedItem.Name) && !item.IsBought {
				item.ReceiptItemID = &receiptItem.ID
				item.IsBought = true
				item.Price = parsedItem.Price
				item.Quantity = parsedItem.Quantity

				if err := tx.Save(&item).Error; err != nil {
					return err
				}
				matched = true
				s.updateItemFrequency(tx, familyID, item.Name, parsedItem.Price)
				break
			}
		}

		if !matched {
			var categoryID uint
			var cat models.Category
			if err := tx.Where("family_id = ?", familyID).Order("sort_order asc").First(&cat).Error; err == nil {
				categoryID = cat.ID
			}

			newItem := models.Item{
				Name:          parsedItem.Name,
				ListID:        listID,
				CategoryID:    categoryID,
				IsBought:      true,
				Price:         parsedItem.Price,
				Quantity:      parsedItem.Quantity,
				Unit:          parsedItem.Unit,
				ReceiptItemID: &receiptItem.ID,
			}
			if err := tx.Create(&newItem).Error; err != nil {
				slog.Error("failed to create new item from receipt", "name", newItem.Name, "error", err)
			}
			s.updateItemFrequency(tx, familyID, newItem.Name, parsedItem.Price)
		}
	}
	return nil
}

func (s *ReceiptService) recalculateListTotal(tx *gorm.DB, listID uint, familyID uint) error {
	var items []models.Item
	if err := tx.Where("list_id = ?", listID).Find(&items).Error; err != nil {
		return err
	}

	total := 0.0
	for _, item := range items {
		if item.IsBought {
			qty := item.Quantity
			if qty == 0 {
				qty = 1
			}
			total += item.Price * qty
		}
	}

	var list models.ShoppingList
	if err := tx.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		return err
	}

	if list.ActualAmount != total {
		list.ActualAmount = total
		if err := tx.Save(&list).Error; err != nil {
			return err
		}
	}
	return nil
}

// ProcessPendingReceipts finds 'new' receipts and tries to parse them
func (s *ReceiptService) ProcessPendingReceipts(ctx context.Context) error {
	if s.gemini == nil {
		return nil // Can't process without client
	}

	var pending []models.Receipt
	if err := s.db.Where("status = ?", "new").Find(&pending).Error; err != nil {
		return err
	}

	count := 0
	for _, r := range pending {
		// We need listID from the receipt model now.
		if r.ListID == nil {
			slog.Warn("Skipping pending receipt with no ListID", "id", r.ID)
			continue
		}

		if err := s.ProcessReceipt(ctx, r.ID, *r.ListID); err != nil {
			slog.Error("Failed to process pending receipt", "id", r.ID, "error", err)
			// Continue with next
		} else {
			count++
		}
	}

	if count > 0 {
		slog.Info("Processed pending receipts", "count", count)
	}
	return nil
}

func (s *ReceiptService) updateItemFrequency(tx *gorm.DB, familyID uint, name string, price float64) {
	var freq models.ItemFrequency
	if err := tx.Where("family_id = ? AND LOWER(item_name) = ?", familyID, strings.ToLower(name)).First(&freq).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			freq = models.ItemFrequency{
				FamilyID:  familyID,
				ItemName:  name,
				Frequency: 1,
				LastPrice: price,
			}
			tx.Create(&freq)
		}
	} else {
		freq.Frequency++
		freq.LastPrice = price
		tx.Save(&freq)
	}
}
