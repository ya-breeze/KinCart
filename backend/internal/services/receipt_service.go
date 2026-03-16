package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"kincart/internal/ai"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"
)

const autoMatchThreshold = 90 // confidence >= this → auto-accept

// ReceiptParser is the AI client interface used by the service.
type ReceiptParser interface {
	ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
	ParseReceiptText(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error)
	MatchReceiptItems(ctx context.Context, receiptItems []string, plannedItems []string) (*ai.MatchResult, error)
}

// MatchSuggestionItem is the JSON element stored in ReceiptItem.SuggestedItems.
type MatchSuggestionItem struct {
	ItemID     uint   `json:"item_id"`
	ItemName   string `json:"item_name"`
	Confidence int    `json:"confidence"`
}

// Response types returned by GetReceiptMatches.

type PlannedItemRef struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ReceiptItemMatch struct {
	ReceiptItemID uint                  `json:"receipt_item_id"`
	ReceiptName   string                `json:"receipt_name"`
	Quantity      float64               `json:"quantity"`
	Price         float64               `json:"price"`
	TotalPrice    float64               `json:"total_price"`
	MatchStatus   string                `json:"match_status"`
	Confidence    int                   `json:"confidence"`
	MatchedItem   *PlannedItemRef       `json:"matched_item"`
	Suggestions   []MatchSuggestionItem `json:"suggestions"`
	IsExtra       bool                  `json:"is_extra"`
}

type ReceiptMatchResponse struct {
	ReceiptID             uint               `json:"receipt_id"`
	Status                string             `json:"status"`
	ShopName              string             `json:"shop_name,omitempty"`
	Date                  string             `json:"date"`
	Total                 float64            `json:"total"`
	Items                 []ReceiptItemMatch  `json:"items"`
	UnmatchedPlannedItems []PlannedItemRef   `json:"unmatched_planned_items"`
}

// receiptItemMatchPlan holds pre-computed match info for one parsed receipt item.
type receiptItemMatchPlan struct {
	PlannedItemID *uint
	MatchStatus   string // "auto" or "unmatched"
	Confidence    int
	Suggestions   []MatchSuggestionItem
}

type ReceiptService struct {
	db           *gorm.DB
	gemini       ReceiptParser
	fileStorage  *FileStorageService
	receiptsPath string
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
	path, err := s.fileStorage.SaveReceipt(familyID, file)
	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: familyID},
		ImagePath:   path,
		Date:        time.Now(),
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		return nil, err
	}

	return &receipt, nil
}

// CreateReceiptFromText saves the plain text as a .txt file and creates a Receipt DB record.
func (s *ReceiptService) CreateReceiptFromText(familyID uint, text string) (*models.Receipt, error) {
	path, err := s.fileStorage.SaveReceiptText(familyID, text)
	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: familyID},
		ImagePath:   path,
		Date:        time.Now(),
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

	if s.gemini == nil {
		return fmt.Errorf("gemini client not available")
	}

	// 1. Get list items for context and matching
	var listItems []models.Item
	if err := s.db.Where("list_id = ?", listID).Find(&listItems).Error; err != nil {
		return fmt.Errorf("failed to fetch list items: %w", err)
	}

	knownItemNames := make([]string, len(listItems))
	for i, item := range listItems {
		knownItemNames[i] = item.Name
	}

	// 2. Parse with Gemini — branch on .txt vs image/PDF
	var parsed *ai.ParsedReceipt
	var parseErr error

	if strings.HasSuffix(strings.ToLower(receipt.ImagePath), ".txt") {
		fullPath := filepath.Join(s.receiptsPath, receipt.ImagePath)
		textContent, err := os.ReadFile(fullPath)
		if err != nil {
			s.db.Model(&receipt).Update("status", "error")
			return fmt.Errorf("failed to read text receipt: %w", err)
		}
		parsed, parseErr = s.gemini.ParseReceiptText(ctx, string(textContent), knownItemNames)
	} else {
		fullPath := filepath.Join(s.receiptsPath, receipt.ImagePath)
		parsed, parseErr = s.gemini.ParseReceipt(ctx, fullPath, knownItemNames)
	}

	if parseErr != nil {
		s.db.Model(&receipt).Update("status", "error")
		return fmt.Errorf("gemini parsing failed: %w", parseErr)
	}

	// 3. Pre-compute item matches outside the transaction (AI call is network I/O)
	matchPlans := s.buildItemMatches(ctx, receipt.FamilyID, listItems, parsed.Items)

	// 4. Transaction to apply everything
	return s.db.Transaction(func(tx *gorm.DB) error {
		date, _ := time.Parse("2006-01-02", parsed.Date)
		receipt.Date = date
		receipt.Total = parsed.Total

		if parsed.StoreName != "" {
			shopID, err := s.findOrCreateShop(tx, receipt.FamilyID, parsed.StoreName)
			if err != nil {
				return err
			}
			receipt.ShopID = shopID
		}

		s.updateListTitle(tx, listID, receipt.Date)

		needsReview, err := s.applyItemMatches(tx, receipt.ID, listID, receipt.FamilyID, parsed.Items, matchPlans, listItems)
		if err != nil {
			return err
		}

		// Check if unbought planned items exist (also triggers pending_review)
		if !needsReview {
			matchedIDs := collectMatchedItemIDs(matchPlans)
			for _, item := range listItems {
				if !item.IsBought && !matchedIDs[item.ID] {
					needsReview = true
					break
				}
			}
		}

		if needsReview {
			receipt.Status = "pending_review"
		} else {
			receipt.Status = "parsed"
		}

		if err := tx.Save(&receipt).Error; err != nil {
			return err
		}

		return s.recalculateListTotal(tx, listID, receipt.FamilyID)
	})
}

// buildItemMatches computes how each parsed receipt item should be matched.
// It first checks ItemAlias, then falls back to AI for unresolved items.
// Returns a slice parallel to parsedItems.
func (s *ReceiptService) buildItemMatches(ctx context.Context, familyID uint, listItems []models.Item, parsedItems []ai.ParsedReceiptItem) []receiptItemMatchPlan {
	plans := make([]receiptItemMatchPlan, len(parsedItems))
	usedPlannedIDs := map[uint]bool{}

	// Build a lookup: lowercase name → Item
	plannedByName := map[string]models.Item{}
	for _, item := range listItems {
		if !item.IsBought {
			plannedByName[strings.ToLower(item.Name)] = item
		}
	}

	// Step 1: Check ItemAlias for auto-matches
	unresolvedIdxs := []int{}
	for i, parsed := range parsedItems {
		var aliases []models.ItemAlias
		s.db.Where("family_id = ? AND LOWER(receipt_name) = ?", familyID, strings.ToLower(parsed.Name)).Find(&aliases)

		matched := false
		for _, alias := range aliases {
			item, ok := plannedByName[strings.ToLower(alias.PlannedName)]
			if ok && !usedPlannedIDs[item.ID] {
				plans[i] = receiptItemMatchPlan{
					PlannedItemID: &item.ID,
					MatchStatus:   "auto",
					Confidence:    100,
					Suggestions:   []MatchSuggestionItem{{ItemID: item.ID, ItemName: item.Name, Confidence: 100}},
				}
				usedPlannedIDs[item.ID] = true
				matched = true
				break
			}
		}
		if !matched {
			unresolvedIdxs = append(unresolvedIdxs, i)
		}
	}

	if len(unresolvedIdxs) == 0 || ctx.Err() != nil {
		return plans
	}

	// Step 2: Call AI for unresolved items
	unresolvedNames := make([]string, len(unresolvedIdxs))
	for j, idx := range unresolvedIdxs {
		unresolvedNames[j] = parsedItems[idx].Name
	}

	plannedNames := make([]string, 0, len(plannedByName))
	for name := range plannedByName {
		if !usedPlannedIDs[plannedByName[name].ID] {
			plannedNames = append(plannedNames, plannedByName[name].Name) // use original case
		}
	}

	aiResult, err := s.gemini.MatchReceiptItems(ctx, unresolvedNames, plannedNames)
	if err != nil {
		slog.Warn("AI item matching failed, leaving items unmatched", "error", err)
		for _, idx := range unresolvedIdxs {
			plans[idx] = receiptItemMatchPlan{
				MatchStatus: "unmatched",
				Suggestions: []MatchSuggestionItem{},
			}
		}
		return plans
	}

	// Build a lookup from AI results: receipt name (lower) → suggestion list
	aiByName := map[string][]MatchSuggestionItem{}
	aiAutoMatch := map[string]*uint{} // receipt name (lower) → planned item ID if high confidence
	for _, sug := range aiResult.Suggestions {
		key := strings.ToLower(sug.ReceiptItemName)
		var suggestions []MatchSuggestionItem
		for _, m := range sug.Matches {
			item, ok := plannedByName[strings.ToLower(m.PlannedItemName)]
			if !ok {
				continue
			}
			suggestions = append(suggestions, MatchSuggestionItem{
				ItemID:     item.ID,
				ItemName:   item.Name,
				Confidence: m.Confidence,
			})
		}
		aiByName[key] = suggestions

		// Check for auto-match candidate (highest confidence ≥ threshold, not yet used)
		if len(suggestions) > 0 && suggestions[0].Confidence >= autoMatchThreshold {
			id := suggestions[0].ItemID
			if !usedPlannedIDs[id] {
				aiAutoMatch[key] = &id
			}
		}
	}

	// Apply AI results to unresolved items
	for _, idx := range unresolvedIdxs {
		key := strings.ToLower(parsedItems[idx].Name)
		suggestions := aiByName[key]
		if suggestions == nil {
			suggestions = []MatchSuggestionItem{}
		}

		if autoID, ok := aiAutoMatch[key]; ok && !usedPlannedIDs[*autoID] {
			confidence := suggestions[0].Confidence
			plans[idx] = receiptItemMatchPlan{
				PlannedItemID: autoID,
				MatchStatus:   "auto",
				Confidence:    confidence,
				Suggestions:   suggestions,
			}
			usedPlannedIDs[*autoID] = true
		} else {
			plans[idx] = receiptItemMatchPlan{
				MatchStatus: "unmatched",
				Suggestions: suggestions,
			}
		}
	}

	return plans
}

// collectMatchedItemIDs returns a set of planned item IDs that were auto-matched.
func collectMatchedItemIDs(plans []receiptItemMatchPlan) map[uint]bool {
	ids := map[uint]bool{}
	for _, p := range plans {
		if p.PlannedItemID != nil {
			ids[*p.PlannedItemID] = true
		}
	}
	return ids
}

// applyItemMatches creates ReceiptItem records and links/creates planned items.
// Returns true if the receipt needs manual review.
func (s *ReceiptService) applyItemMatches(tx *gorm.DB, receiptID uint, listID uint, familyID uint, parsedItems []ai.ParsedReceiptItem, plans []receiptItemMatchPlan, listItems []models.Item) (bool, error) {
	// Build Item lookup by ID for quick access
	itemByID := map[uint]models.Item{}
	for _, item := range listItems {
		itemByID[item.ID] = item
	}

	// First category for new unmatched items
	var defaultCategoryID uint
	var cat models.Category
	if err := tx.Where("family_id = ?", familyID).Order("sort_order asc").First(&cat).Error; err == nil {
		defaultCategoryID = cat.ID
	}

	needsReview := false

	for i, parsedItem := range parsedItems {
		plan := plans[i]

		sugJSON := "[]"
		if b, err := json.Marshal(plan.Suggestions); err == nil {
			sugJSON = string(b)
		}

		receiptItem := models.ReceiptItem{
			ReceiptID:      receiptID,
			Name:           parsedItem.Name,
			Quantity:       parsedItem.Quantity,
			Unit:           parsedItem.Unit,
			Price:          parsedItem.Price,
			TotalPrice:     parsedItem.TotalPrice,
			MatchStatus:    plan.MatchStatus,
			Confidence:     plan.Confidence,
			SuggestedItems: sugJSON,
		}

		if plan.MatchStatus == "auto" && plan.PlannedItemID != nil {
			// Auto-match: link and mark bought
			receiptItem.MatchedItemID = plan.PlannedItemID

			item := itemByID[*plan.PlannedItemID]
			item.ReceiptItemID = nil // set after create
			item.IsBought = true
			item.Price = parsedItem.Price
			item.Quantity = parsedItem.Quantity

			if err := tx.Create(&receiptItem).Error; err != nil {
				return false, err
			}
			receiptItem.MatchedItemID = plan.PlannedItemID
			item.ReceiptItemID = &receiptItem.ID
			if err := tx.Save(&item).Error; err != nil {
				return false, err
			}
			s.updateItemFrequency(tx, familyID, item.Name, parsedItem.Price)
		} else {
			// Unmatched — store suggestions for user review
			needsReview = true

			if err := tx.Create(&receiptItem).Error; err != nil {
				return false, err
			}
		}
	}

	return needsReview, nil
}

// GetReceiptMatches returns the receipt with AI match suggestions for user review.
func (s *ReceiptService) GetReceiptMatches(receiptID uint, familyID uint) (*ReceiptMatchResponse, error) {
	var receipt models.Receipt
	if err := s.db.Preload("Items").Preload("Shop").
		Where("id = ? AND family_id = ?", receiptID, familyID).
		First(&receipt).Error; err != nil {
		return nil, fmt.Errorf("receipt not found: %w", err)
	}

	if receipt.ListID == nil {
		return nil, fmt.Errorf("receipt has no associated list")
	}

	// Load all planned items from the list
	var listItems []models.Item
	s.db.Where("list_id = ?", *receipt.ListID).Find(&listItems)

	// Build lookup of matched planned item IDs
	matchedIDs := map[uint]bool{}
	for _, ri := range receipt.Items {
		if ri.MatchedItemID != nil {
			matchedIDs[*ri.MatchedItemID] = true
		}
	}

	// Build item lookup
	itemByID := map[uint]models.Item{}
	for _, item := range listItems {
		itemByID[item.ID] = item
	}

	// Build response items
	items := make([]ReceiptItemMatch, 0, len(receipt.Items))
	for _, ri := range receipt.Items {
		var suggestions []MatchSuggestionItem
		if ri.SuggestedItems != "" {
			_ = json.Unmarshal([]byte(ri.SuggestedItems), &suggestions)
		}
		if suggestions == nil {
			suggestions = []MatchSuggestionItem{}
		}

		var matchedItem *PlannedItemRef
		if ri.MatchedItemID != nil {
			if item, ok := itemByID[*ri.MatchedItemID]; ok {
				matchedItem = &PlannedItemRef{ID: item.ID, Name: item.Name}
			}
		}

		isExtra := ri.MatchStatus == "unmatched" && len(suggestions) == 0

		items = append(items, ReceiptItemMatch{
			ReceiptItemID: ri.ID,
			ReceiptName:   ri.Name,
			Quantity:      ri.Quantity,
			Price:         ri.Price,
			TotalPrice:    ri.TotalPrice,
			MatchStatus:   ri.MatchStatus,
			Confidence:    ri.Confidence,
			MatchedItem:   matchedItem,
			Suggestions:   suggestions,
			IsExtra:       isExtra,
		})
	}

	// Collect unmatched planned items (not bought and not matched to any receipt item)
	var unmatched []PlannedItemRef
	for _, item := range listItems {
		if !item.IsBought && !matchedIDs[item.ID] {
			unmatched = append(unmatched, PlannedItemRef{ID: item.ID, Name: item.Name})
		}
	}
	if unmatched == nil {
		unmatched = []PlannedItemRef{}
	}

	shopName := ""
	if receipt.Shop != nil {
		shopName = receipt.Shop.Name
	}

	return &ReceiptMatchResponse{
		ReceiptID:             receipt.ID,
		Status:                receipt.Status,
		ShopName:              shopName,
		Date:                  receipt.Date.Format("2006-01-02"),
		Total:                 receipt.Total,
		Items:                 items,
		UnmatchedPlannedItems: unmatched,
	}, nil
}

// ConfirmMatch confirms or updates the match for a single receipt item.
// If plannedItemID is nil, a new list item is created from the receipt item data.
func (s *ReceiptService) ConfirmMatch(ctx context.Context, receiptItemID uint, plannedItemID *uint, familyID uint) error {
	// Load receipt item and verify ownership via receipt
	var receiptItem models.ReceiptItem
	if err := s.db.First(&receiptItem, receiptItemID).Error; err != nil {
		return fmt.Errorf("receipt item not found: %w", err)
	}

	var receipt models.Receipt
	if err := s.db.Where("id = ? AND family_id = ?", receiptItem.ReceiptID, familyID).First(&receipt).Error; err != nil {
		return fmt.Errorf("receipt not found or access denied: %w", err)
	}
	if receipt.ListID == nil {
		return fmt.Errorf("receipt has no associated list")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if plannedItemID != nil {
			// Link to existing planned item
			var item models.Item
			if err := tx.Where("id = ? AND family_id = ?", *plannedItemID, familyID).First(&item).Error; err != nil {
				return fmt.Errorf("planned item not found: %w", err)
			}

			item.ReceiptItemID = &receiptItemID
			item.IsBought = true
			item.Price = receiptItem.Price
			item.Quantity = receiptItem.Quantity
			if err := tx.Save(&item).Error; err != nil {
				return err
			}

			// Upsert ItemAlias for future auto-matching
			s.upsertItemAlias(tx, familyID, item.Name, receiptItem.Name, receiptItem.Price, receipt.ShopID)
			s.updateItemFrequency(tx, familyID, item.Name, receiptItem.Price)

			receiptItem.MatchedItemID = plannedItemID
			receiptItem.MatchStatus = "confirmed"
		} else {
			// Create new list item from receipt data (extra/impulse buy)
			var defaultCategoryID uint
			var cat models.Category
			if err := tx.Where("family_id = ?", familyID).Order("sort_order asc").First(&cat).Error; err == nil {
				defaultCategoryID = cat.ID
			}

			newItem := models.Item{
				TenantModel:   coremodels.TenantModel{FamilyID: familyID},
				Name:          receiptItem.Name,
				ListID:        *receipt.ListID,
				CategoryID:    defaultCategoryID,
				IsBought:      true,
				Price:         receiptItem.Price,
				Quantity:      receiptItem.Quantity,
				Unit:          receiptItem.Unit,
				ReceiptItemID: &receiptItemID,
			}
			if err := tx.Create(&newItem).Error; err != nil {
				return err
			}

			// Self-alias so it's recognized in future receipts
			s.upsertItemAlias(tx, familyID, receiptItem.Name, receiptItem.Name, receiptItem.Price, receipt.ShopID)
			s.updateItemFrequency(tx, familyID, receiptItem.Name, receiptItem.Price)

			receiptItem.MatchedItemID = &newItem.ID
			receiptItem.MatchStatus = "confirmed"
		}

		if err := tx.Save(&receiptItem).Error; err != nil {
			return err
		}

		s.checkAndUpdateReceiptStatus(tx, receipt.ID, *receipt.ListID, familyID)
		return s.recalculateListTotal(tx, *receipt.ListID, familyID)
	})
}

// DismissReceiptItem marks a receipt item as dismissed — not relevant to the list.
func (s *ReceiptService) DismissReceiptItem(receiptItemID uint, familyID uint) error {
	var receiptItem models.ReceiptItem
	if err := s.db.First(&receiptItem, receiptItemID).Error; err != nil {
		return fmt.Errorf("receipt item not found: %w", err)
	}

	var receipt models.Receipt
	if err := s.db.Where("id = ? AND family_id = ?", receiptItem.ReceiptID, familyID).First(&receipt).Error; err != nil {
		return fmt.Errorf("receipt not found or access denied: %w", err)
	}

	receiptItem.MatchStatus = "dismissed"
	if err := s.db.Save(&receiptItem).Error; err != nil {
		return err
	}

	if receipt.ListID != nil {
		s.checkAndUpdateReceiptStatus(s.db, receipt.ID, *receipt.ListID, familyID)
	}
	return nil
}

// ConfirmAllMatches accepts all current auto-matches, creates new items for
// unmatched receipt items, and leaves unbought planned items unchanged.
func (s *ReceiptService) ConfirmAllMatches(ctx context.Context, receiptID uint, familyID uint) error {
	var receipt models.Receipt
	if err := s.db.Preload("Items").Where("id = ? AND family_id = ?", receiptID, familyID).First(&receipt).Error; err != nil {
		return fmt.Errorf("receipt not found: %w", err)
	}
	if receipt.ListID == nil {
		return fmt.Errorf("receipt has no associated list")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var defaultCategoryID uint
		var cat models.Category
		if err := tx.Where("family_id = ?", familyID).Order("sort_order asc").First(&cat).Error; err == nil {
			defaultCategoryID = cat.ID
		}

		for _, ri := range receipt.Items {
			switch ri.MatchStatus {
			case "auto":
				// Already applied — just mark as confirmed
				ri.MatchStatus = "confirmed"
				tx.Save(&ri)
			case "unmatched":
				// Create new list item (extra buy)
				newItem := models.Item{
					TenantModel:   coremodels.TenantModel{FamilyID: familyID},
					Name:          ri.Name,
					ListID:        *receipt.ListID,
					CategoryID:    defaultCategoryID,
					IsBought:      true,
					Price:         ri.Price,
					Quantity:      ri.Quantity,
					Unit:          ri.Unit,
					ReceiptItemID: &ri.ID,
				}
				if err := tx.Create(&newItem).Error; err != nil {
					slog.Error("failed to create item from receipt in confirm-all", "name", ri.Name, "error", err)
					continue
				}
				s.upsertItemAlias(tx, familyID, ri.Name, ri.Name, ri.Price, receipt.ShopID)
				s.updateItemFrequency(tx, familyID, ri.Name, ri.Price)

				ri.MatchedItemID = &newItem.ID
				ri.MatchStatus = "confirmed"
				tx.Save(&ri)
			case "dismissed":
				// Already handled — skip
			}
		}

		tx.Model(&receipt).Update("status", "parsed")
		return s.recalculateListTotal(tx, *receipt.ListID, familyID)
	})
}

// checkAndUpdateReceiptStatus sets receipt to "parsed" when all items are resolved.
func (s *ReceiptService) checkAndUpdateReceiptStatus(db *gorm.DB, receiptID uint, listID uint, familyID uint) {
	var pendingCount int64
	db.Model(&models.ReceiptItem{}).
		Where("receipt_id = ? AND match_status = 'unmatched'", receiptID).
		Count(&pendingCount)

	if pendingCount == 0 {
		// Also check for unbought planned items
		var receipt models.Receipt
		db.First(&receipt, receiptID)
		var unmatchedPlanned int64
		db.Model(&models.Item{}).
			Where("list_id = ? AND is_bought = false", listID).
			Count(&unmatchedPlanned)
		if unmatchedPlanned == 0 {
			db.Model(&models.Receipt{}).Where("id = ?", receiptID).Update("status", "parsed")
		}
	}
}

// upsertItemAlias creates or increments a planned_name → receipt_name mapping.
func (s *ReceiptService) upsertItemAlias(tx *gorm.DB, familyID uint, plannedName string, receiptName string, price float64, shopID *uint) {
	var alias models.ItemAlias
	q := tx.Where("family_id = ? AND LOWER(planned_name) = ? AND LOWER(receipt_name) = ?",
		familyID, strings.ToLower(plannedName), strings.ToLower(receiptName))
	if shopID != nil {
		q = q.Where("shop_id = ?", *shopID)
	} else {
		q = q.Where("shop_id IS NULL")
	}

	if err := q.First(&alias).Error; err != nil {
		// Create new
		alias = models.ItemAlias{
			FamilyID:      familyID,
			PlannedName:   plannedName,
			ReceiptName:   receiptName,
			ShopID:        shopID,
			LastPrice:     price,
			PurchaseCount: 1,
			LastUsedAt:    time.Now(),
		}
		tx.Create(&alias)
	} else {
		alias.PurchaseCount++
		alias.LastPrice = price
		alias.LastUsedAt = time.Now()
		tx.Save(&alias)
	}
}

func (s *ReceiptService) findOrCreateShop(tx *gorm.DB, familyID uint, storeName string) (*uint, error) {
	var shop models.Shop
	if err := tx.Where("LOWER(name) = ? AND family_id = ?", strings.ToLower(storeName), familyID).First(&shop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			shop = models.Shop{
				TenantModel: coremodels.TenantModel{FamilyID: familyID},
				Name:        storeName,
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

// ProcessPendingReceipts finds 'new' receipts and tries to parse them.
func (s *ReceiptService) ProcessPendingReceipts(ctx context.Context) error {
	if s.gemini == nil {
		return nil
	}

	var pending []models.Receipt
	if err := s.db.Where("status = ?", "new").Find(&pending).Error; err != nil {
		return err
	}

	count := 0
	for _, r := range pending {
		if r.ListID == nil {
			slog.Warn("Skipping pending receipt with no ListID", "id", r.ID)
			continue
		}

		if err := s.ProcessReceipt(ctx, r.ID, *r.ListID); err != nil {
			slog.Error("Failed to process pending receipt", "id", r.ID, "error", err)
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
