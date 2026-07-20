package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kincart/internal/ai"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"
)

const (
	autoMatchThreshold   = 90 // confidence >= this → auto-accept
	matchStatusAuto      = "auto"
	matchStatusConfirmed = "confirmed"
	matchStatusUnmatched = "unmatched"
)

// ErrGeminiUnavailable is returned when the Gemini AI client is not configured.
var ErrGeminiUnavailable = fmt.Errorf("gemini client not available")

// ErrReceiptItemNotFound is returned when a receipt item cannot be found.
var ErrReceiptItemNotFound = fmt.Errorf("receipt item not found")

// ErrReceiptNotFound is returned when a receipt cannot be found or access is denied.
var ErrReceiptNotFound = fmt.Errorf("receipt not found or access denied")

// ErrPlannedItemNotFound is returned when a planned item cannot be found on the list.
var ErrPlannedItemNotFound = fmt.Errorf("planned item not found or not on this list")

// ErrNoAssociatedList is returned when a receipt has no linked list.
var ErrNoAssociatedList = fmt.Errorf("receipt has no associated list")

// ReceiptParser is the AI client interface used by the service.
type ReceiptParser interface {
	ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ai.ParsedReceipt, error)
	ParseReceiptText(ctx context.Context, receiptText string, knownItems []string) (*ai.ParsedReceipt, error)
	MatchReceiptItems(ctx context.Context, receiptItems []string, plannedItems []string) (*ai.MatchResult, error)
	SuggestItemDefaults(ctx context.Context, name string, categories []string) (ai.SuggestedItemDefaults, error)
}

// MatchSuggestionItem is the JSON element stored in ReceiptItem.SuggestedItems.
type MatchSuggestionItem struct {
	ItemID     uuid.UUID `json:"item_id"`
	ItemName   string    `json:"item_name"`
	Confidence int       `json:"confidence"`
}

// Response types returned by GetReceiptMatches.

type PlannedItemRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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
	ReceiptID             uuid.UUID          `json:"receipt_id"`
	Status                string             `json:"status"`
	ShopName              string             `json:"shop_name,omitempty"`
	Date                  string             `json:"date"`
	Total                 float64            `json:"total"`
	Items                 []ReceiptItemMatch `json:"items"`
	UnmatchedPlannedItems []PlannedItemRef   `json:"unmatched_planned_items"`
	AlreadyBoughtItems    []PlannedItemRef   `json:"already_bought_items"`
}

// receiptItemMatchPlan holds pre-computed match info for one parsed receipt item.
type receiptItemMatchPlan struct {
	PlannedItemID *uuid.UUID
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

func (s *ReceiptService) CreateReceipt(familyID uuid.UUID, file *multipart.FileHeader) (*models.Receipt, error) {
	path, err := s.fileStorage.SaveReceipt(familyID, file)
	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ImagePath:   path,
		Date:        time.Now(),
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		return nil, err
	}

	return &receipt, nil
}

// CreateReceiptFromText saves the plain text as a .txt file and creates a Receipt DB record.
func (s *ReceiptService) CreateReceiptFromText(familyID uuid.UUID, text string) (*models.Receipt, error) {
	path, err := s.fileStorage.SaveReceiptText(familyID, text)
	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ImagePath:   path,
		Date:        time.Now(),
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		return nil, err
	}

	return &receipt, nil
}

func (s *ReceiptService) ProcessReceipt(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
	var receipt models.Receipt
	if err := s.db.Preload("Items").First(&receipt, "id = ?", receiptID).Error; err != nil {
		return fmt.Errorf("receipt not found: %w", err)
	}

	if s.gemini == nil {
		return ErrGeminiUnavailable
	}

	// 1. Get list items for context and matching — scope by family_id for tenant isolation
	var listItems []models.Item
	if err := s.db.Where("list_id = ? AND family_id = ?", listID, receipt.FamilyID).Find(&listItems).Error; err != nil {
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

	normalizePackItems(parsed)

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

		needsReview, err := s.applyItemMatches(tx, receipt.ID, receipt.FamilyID, parsed.Items, matchPlans, listItems)
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
//
//nolint:gocognit,gocyclo
func (s *ReceiptService) buildItemMatches(ctx context.Context, familyID uuid.UUID, listItems []models.Item, parsedItems []ai.ParsedReceiptItem) []receiptItemMatchPlan {
	plans := make([]receiptItemMatchPlan, len(parsedItems))
	usedPlannedIDs := map[uuid.UUID]bool{}

	// Build a lookup: lowercase name → Item.
	// Include items not yet claimed by any receipt item (ReceiptItemID == nil),
	// even if already bought — this handles the case where the user manually ticked
	// items during shopping before uploading the receipt.
	plannedByName := map[string]models.Item{}
	for _, item := range listItems {
		if item.ReceiptItemID == nil {
			plannedByName[strings.ToLower(item.Name)] = item
		}
	}

	// Step 1: Batch-load all ItemAliases for this family to avoid N+1 queries
	receiptNames := make([]string, len(parsedItems))
	for i, p := range parsedItems {
		receiptNames[i] = strings.ToLower(p.Name)
	}
	var allAliases []models.ItemAlias
	s.db.Where("family_id = ? AND LOWER(receipt_name) IN ?", familyID, receiptNames).Find(&allAliases)

	// Group aliases by lowercase receipt name
	aliasesByReceiptName := map[string][]models.ItemAlias{}
	for _, a := range allAliases {
		key := strings.ToLower(a.ReceiptName)
		aliasesByReceiptName[key] = append(aliasesByReceiptName[key], a)
	}

	// Check aliases for auto-matches
	unresolvedIdxs := []int{}
	for i, parsed := range parsedItems {
		aliases := aliasesByReceiptName[strings.ToLower(parsed.Name)]

		matched := false
		for _, alias := range aliases {
			item, ok := plannedByName[strings.ToLower(alias.PlannedName)]
			if ok && !usedPlannedIDs[item.ID] {
				plans[i] = receiptItemMatchPlan{
					PlannedItemID: &item.ID,
					MatchStatus:   matchStatusAuto,
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

	if len(unresolvedIdxs) == 0 {
		return plans
	}

	// If context was canceled or no AI client is available, mark remaining as unmatched
	if ctx.Err() != nil || s.gemini == nil {
		for _, idx := range unresolvedIdxs {
			plans[idx] = receiptItemMatchPlan{
				MatchStatus: matchStatusUnmatched,
				Suggestions: []MatchSuggestionItem{},
			}
		}
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
				MatchStatus: matchStatusUnmatched,
				Suggestions: []MatchSuggestionItem{},
			}
		}
		return plans
	}

	// Build a lookup from AI results: receipt name (lower) → suggestion list
	aiByName := map[string][]MatchSuggestionItem{}
	aiAutoMatch := map[string]*uuid.UUID{} // receipt name (lower) → planned item ID if high confidence
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
		// Sort by confidence descending so suggestions[0] is the best candidate
		sort.Slice(suggestions, func(a, b int) bool {
			return suggestions[a].Confidence > suggestions[b].Confidence
		})
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
				MatchStatus:   matchStatusAuto,
				Confidence:    confidence,
				Suggestions:   suggestions,
			}
			usedPlannedIDs[*autoID] = true
		} else {
			plans[idx] = receiptItemMatchPlan{
				MatchStatus: matchStatusUnmatched,
				Suggestions: suggestions,
			}
		}
	}

	return plans
}

// collectMatchedItemIDs returns a set of planned item IDs that were auto-matched.
func collectMatchedItemIDs(plans []receiptItemMatchPlan) map[uuid.UUID]bool {
	ids := map[uuid.UUID]bool{}
	for _, p := range plans {
		if p.PlannedItemID != nil {
			ids[*p.PlannedItemID] = true
		}
	}
	return ids
}

// applyItemMatches creates ReceiptItem records and links/creates planned items.
// Returns true if the receipt needs manual review.
func (s *ReceiptService) applyItemMatches(tx *gorm.DB, receiptID uuid.UUID, familyID uuid.UUID, parsedItems []ai.ParsedReceiptItem, plans []receiptItemMatchPlan, listItems []models.Item) (bool, error) {
	// Build Item lookup by ID for quick access
	itemByID := map[uuid.UUID]models.Item{}
	for _, item := range listItems {
		itemByID[item.ID] = item
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

		if plan.MatchStatus == matchStatusAuto && plan.PlannedItemID != nil {
			// Auto-match: link and mark bought
			item := itemByID[*plan.PlannedItemID]
			item.IsBought = true
			// The receipt proves it was bought, so a shopper's earlier "not
			// found" is stale. Bought and absent are mutually exclusive.
			item.IsAbsent = false
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
func (s *ReceiptService) GetReceiptMatches(receiptID uuid.UUID, familyID uuid.UUID) (*ReceiptMatchResponse, error) {
	var receipt models.Receipt
	if err := s.db.Preload("Items").Preload("Shop").
		Where("id = ? AND family_id = ?", receiptID, familyID).
		First(&receipt).Error; err != nil {
		return nil, fmt.Errorf("receipt not found: %w", err)
	}

	if receipt.ListID == nil {
		return nil, fmt.Errorf("receipt has no associated list")
	}

	// Load all planned items from the list — scope by family_id for tenant isolation
	var listItems []models.Item
	if err := s.db.Where("list_id = ? AND family_id = ?", *receipt.ListID, familyID).Find(&listItems).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch list items: %w", err)
	}

	// Build lookup of matched planned item IDs
	matchedIDs := map[uuid.UUID]bool{}
	for _, ri := range receipt.Items {
		if ri.MatchedItemID != nil {
			matchedIDs[*ri.MatchedItemID] = true
		}
	}

	// Build item lookup
	itemByID := map[uuid.UUID]models.Item{}
	for _, item := range listItems {
		itemByID[item.ID] = item
	}

	// Build response items
	items := make([]ReceiptItemMatch, 0, len(receipt.Items))
	for _, ri := range receipt.Items {
		var suggestions []MatchSuggestionItem
		if ri.SuggestedItems != "" {
			if err := json.Unmarshal([]byte(ri.SuggestedItems), &suggestions); err != nil {
				slog.Warn("Failed to unmarshal suggested items", "receipt_item_id", ri.ID, "error", err)
			}
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

		isExtra := ri.MatchStatus == matchStatusUnmatched && len(suggestions) == 0

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

	// Collect already-bought items not yet linked to any receipt item.
	// These are available as link targets so users can connect receipt items
	// to items they manually ticked before uploading the receipt.
	var alreadyBought []PlannedItemRef
	for _, item := range listItems {
		if item.IsBought && item.ReceiptItemID == nil && !matchedIDs[item.ID] {
			alreadyBought = append(alreadyBought, PlannedItemRef{ID: item.ID, Name: item.Name})
		}
	}
	if alreadyBought == nil {
		alreadyBought = []PlannedItemRef{}
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
		AlreadyBoughtItems:    alreadyBought,
	}, nil
}

// ConfirmMatch confirms or updates the match for a single receipt item.
// If plannedItemID is nil and the item was previously matched, it reverts to "unmatched".
// If plannedItemID is nil and the item was not matched, a new list item is created.
func (s *ReceiptService) ConfirmMatch(ctx context.Context, receiptItemID uint, plannedItemID *uuid.UUID, familyID uuid.UUID) error {
	// Load receipt item and verify ownership via receipt
	var receiptItem models.ReceiptItem
	if err := s.db.First(&receiptItem, receiptItemID).Error; err != nil {
		return fmt.Errorf("%w: %v", ErrReceiptItemNotFound, err)
	}

	var receipt models.Receipt
	if err := s.db.Where("id = ? AND family_id = ?", receiptItem.ReceiptID, familyID).First(&receipt).Error; err != nil {
		return fmt.Errorf("%w: %v", ErrReceiptNotFound, err)
	}
	if receipt.ListID == nil {
		return ErrNoAssociatedList
	}

	wasPreviouslyMatched := receiptItem.MatchedItemID != nil

	// A new list item is created only when linking to no planned item and this
	// receipt item was not already matched (the `default` switch branch below).
	// Resolve its defaults only in that case — otherwise linking to an existing
	// planned item, or unmatching, would pay for a history read and a possible
	// synchronous Gemini call whose result is discarded. Resolved before the
	// transaction so neither the history read nor the AI fallback holds the write lock.
	var newUnit string
	var newCategoryID uuid.UUID
	willCreateNewItem := plannedItemID == nil && !wasPreviouslyMatched
	if willCreateNewItem {
		newUnit, newCategoryID = s.resolveNewReceiptItemDefaults(ctx, familyID, receiptItem.Name, receipt.ShopID, receiptItem.Unit)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Clear any previous association when re-matching
		if receiptItem.MatchedItemID != nil {
			var oldItem models.Item
			if err := tx.Where("id = ? AND family_id = ?", *receiptItem.MatchedItemID, familyID).First(&oldItem).Error; err == nil {
				if oldItem.IsReceiptCreated {
					// Item was created by this receipt match (Add as new) — delete it to avoid phantom accumulation
					if err := tx.Delete(&oldItem).Error; err != nil {
						return fmt.Errorf("failed to delete orphaned receipt item: %w", err)
					}
				} else {
					// Pre-existing planned item — just unlink it
					oldItem.ReceiptItemID = nil
					oldItem.IsBought = false
					if err := tx.Save(&oldItem).Error; err != nil {
						return fmt.Errorf("failed to clear previous match: %w", err)
					}
				}
			}
			receiptItem.MatchedItemID = nil
		}

		switch {
		case plannedItemID != nil:
			// Link to existing planned item — verify it belongs to the receipt's list
			var item models.Item
			if err := tx.Where("id = ? AND family_id = ? AND list_id = ?", *plannedItemID, familyID, *receipt.ListID).First(&item).Error; err != nil {
				return fmt.Errorf("%w: %v", ErrPlannedItemNotFound, err)
			}

			item.ReceiptItemID = &receiptItemID
			item.IsBought = true
			// See auto-match above: a confirmed purchase clears absent.
			item.IsAbsent = false
			item.Price = receiptItem.Price
			item.Quantity = receiptItem.Quantity
			if err := tx.Save(&item).Error; err != nil {
				return err
			}

			// Upsert ItemAlias for future auto-matching
			s.upsertItemAlias(tx, familyID, item.Name, receiptItem.Name, receiptItem.Price, receipt.ShopID, item.Unit, CategoryIDPtr(item.CategoryID))
			s.updateItemFrequency(tx, familyID, item.Name, receiptItem.Price)

			receiptItem.MatchedItemID = plannedItemID
			receiptItem.MatchStatus = matchStatusConfirmed
		case wasPreviouslyMatched:
			// Unmatch: revert to "unmatched" so user can pick a different match
			receiptItem.MatchStatus = matchStatusUnmatched
		default:
			// Create new list item from receipt data (extra/impulse buy). Category
			// and unit come from history/AI (resolved above), not the first category
			// by sort order.
			newItem := models.Item{
				TenantModel:      coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
				Name:             receiptItem.Name,
				ListID:           *receipt.ListID,
				CategoryID:       newCategoryID,
				IsBought:         true,
				Price:            receiptItem.Price,
				Quantity:         receiptItem.Quantity,
				Unit:             newUnit,
				ReceiptItemID:    &receiptItemID,
				IsReceiptCreated: true,
			}
			if err := tx.Create(&newItem).Error; err != nil {
				return err
			}

			// Self-alias so it's recognized in future receipts
			s.upsertItemAlias(tx, familyID, receiptItem.Name, receiptItem.Name, receiptItem.Price, receipt.ShopID, newItem.Unit, CategoryIDPtr(newItem.CategoryID))
			s.updateItemFrequency(tx, familyID, receiptItem.Name, receiptItem.Price)

			receiptItem.MatchedItemID = &newItem.ID
			receiptItem.MatchStatus = matchStatusConfirmed
		}

		if err := tx.Save(&receiptItem).Error; err != nil {
			return err
		}

		s.checkAndUpdateReceiptStatus(tx, receipt.ID, *receipt.ListID, familyID)
		return s.recalculateListTotal(tx, *receipt.ListID, familyID)
	})
}

// DismissReceiptItem marks a receipt item as dismissed — not relevant to the list.
func (s *ReceiptService) DismissReceiptItem(receiptItemID uint, familyID uuid.UUID) error {
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
func (s *ReceiptService) ConfirmAllMatches(ctx context.Context, receiptID uuid.UUID, familyID uuid.UUID) error {
	var receipt models.Receipt
	if err := s.db.Preload("Items").Where("id = ? AND family_id = ?", receiptID, familyID).First(&receipt).Error; err != nil {
		return fmt.Errorf("receipt not found: %w", err)
	}
	if receipt.ListID == nil {
		return fmt.Errorf("receipt has no associated list")
	}

	// Resolve defaults for every item we may create, before the transaction, so the
	// history reads and AI fallbacks stay off the write lock. Keyed by receipt item ID.
	type itemDefaults struct {
		unit       string
		categoryID uuid.UUID
	}
	resolved := make(map[uint]itemDefaults)
	for _, ri := range receipt.Items {
		if ri.MatchStatus == matchStatusUnmatched {
			unit, categoryID := s.resolveNewReceiptItemDefaults(ctx, familyID, ri.Name, receipt.ShopID, ri.Unit)
			resolved[ri.ID] = itemDefaults{unit: unit, categoryID: categoryID}
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, ri := range receipt.Items {
			switch ri.MatchStatus {
			case matchStatusAuto:
				// Already applied — just mark as confirmed
				ri.MatchStatus = matchStatusConfirmed
				if err := tx.Save(&ri).Error; err != nil {
					return fmt.Errorf("failed to confirm auto-match for %q: %w", ri.Name, err)
				}
			case matchStatusUnmatched:
				// Create new list item (extra buy). Category/unit from history/AI
				// (resolved above), not the first category by sort order.
				def := resolved[ri.ID]
				newItem := models.Item{
					TenantModel:      coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
					Name:             ri.Name,
					ListID:           *receipt.ListID,
					CategoryID:       def.categoryID,
					IsBought:         true,
					Price:            ri.Price,
					Quantity:         ri.Quantity,
					Unit:             def.unit,
					ReceiptItemID:    &ri.ID,
					IsReceiptCreated: true,
				}
				if err := tx.Create(&newItem).Error; err != nil {
					return fmt.Errorf("failed to create item from receipt in confirm-all for %q: %w", ri.Name, err)
				}
				s.upsertItemAlias(tx, familyID, ri.Name, ri.Name, ri.Price, receipt.ShopID, newItem.Unit, CategoryIDPtr(newItem.CategoryID))
				s.updateItemFrequency(tx, familyID, ri.Name, ri.Price)

				ri.MatchedItemID = &newItem.ID
				ri.MatchStatus = matchStatusConfirmed
				if err := tx.Save(&ri).Error; err != nil {
					return fmt.Errorf("failed to save confirmed receipt item %q: %w", ri.Name, err)
				}
			case "dismissed":
				// Already handled — skip
			}
		}

		tx.Model(&receipt).Update("status", "parsed")
		return s.recalculateListTotal(tx, *receipt.ListID, familyID)
	})
}

// checkAndUpdateReceiptStatus sets receipt to "parsed" when all items are resolved.
func (s *ReceiptService) checkAndUpdateReceiptStatus(db *gorm.DB, receiptID uuid.UUID, listID uuid.UUID, familyID uuid.UUID) {
	var pendingCount int64
	db.Model(&models.ReceiptItem{}).
		Where("receipt_id = ? AND match_status = 'unmatched'", receiptID).
		Count(&pendingCount)

	if pendingCount == 0 {
		var receipt models.Receipt
		db.First(&receipt, "id = ?", receiptID)
		var unmatchedPlanned int64
		db.Model(&models.Item{}).
			Where("list_id = ? AND family_id = ? AND is_bought = false", listID, familyID).
			Count(&unmatchedPlanned)
		if unmatchedPlanned == 0 {
			db.Model(&models.Receipt{}).Where("id = ?", receiptID).Update("status", "parsed")
		}
	}
}

// UpsertItemAlias creates or increments a planned_name → receipt_name mapping for a family.
// Returns the resulting alias record.
//
// unit and categoryID record what the item was bought as, so later additions of the
// same name can be prefilled. Latest wins, but an empty unit or nil categoryID never
// erases a value already recorded — a caller that simply does not know (an item with
// no category set, say) must not wipe out history gathered from earlier purchases.
func UpsertItemAlias(tx *gorm.DB, familyID uuid.UUID, plannedName, receiptName string, price float64, shopID *uuid.UUID, unit string, categoryID *uuid.UUID) (*models.ItemAlias, error) {
	plannedLower := strings.ToLower(plannedName)
	receiptLower := strings.ToLower(receiptName)

	var alias models.ItemAlias
	q := tx.Where("family_id = ? AND planned_name_lower = ? AND receipt_name_lower = ?",
		familyID, plannedLower, receiptLower)
	if shopID != nil {
		q = q.Where("shop_id = ?", *shopID)
	} else {
		q = q.Where("shop_id IS NULL")
	}

	if err := q.First(&alias).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		alias = models.ItemAlias{
			FamilyID:         familyID,
			PlannedName:      plannedName,
			PlannedNameLower: plannedLower,
			ReceiptName:      receiptName,
			ReceiptNameLower: receiptLower,
			ShopID:           shopID,
			LastPrice:        price,
			Unit:             unit,
			CategoryID:       categoryID,
			PurchaseCount:    1,
			LastUsedAt:       time.Now(),
		}
		if err := tx.Create(&alias).Error; err != nil {
			return nil, err
		}
		return &alias, nil
	}
	alias.PurchaseCount++
	alias.LastPrice = price
	alias.LastUsedAt = time.Now()
	// Latest wins, but only when the caller actually knows: an empty unit or nil
	// category leaves whatever earlier purchases recorded intact.
	if unit != "" {
		alias.Unit = unit
	}
	if categoryID != nil {
		alias.CategoryID = categoryID
	}
	if err := tx.Save(&alias).Error; err != nil {
		return nil, err
	}
	return &alias, nil
}

// resolveNewReceiptItemDefaults decides the unit and category for a list item being
// created from an unmatched receipt item, replacing the old "first category by
// sort_order" default.
//
// Called BEFORE the write transaction, deliberately. The creation sites run inside
// ConfirmMatch / ConfirmAllMatches — user-triggered, not the background ticker the
// design first assumed — so both the history read and any AI call must stay off the
// SQLite write lock. Resolving here and passing the result in keeps the transaction
// free of network I/O.
//
// Unit: the receipt's own parsed unit is real data about what was bought and wins;
// history/AI only fills a blank. Category: receipt items carry none, so it is
// history first, then a constrained AI guess, then uncategorized (uuid.Nil).
func (s *ReceiptService) resolveNewReceiptItemDefaults(ctx context.Context, familyID uuid.UUID,
	name string, shopID *uuid.UUID, receiptUnit string) (unit string, categoryID uuid.UUID) {
	unit = receiptUnit

	defaults, err := ResolveItemDefaults(ctx, s.db, familyID, name, shopID)
	if err != nil {
		slog.Warn("Could not resolve receipt item history defaults", "name", name, "error", err)
	}
	if unit == "" {
		unit = defaults.Unit
	}
	if defaults.CategoryID != nil {
		return unit, *defaults.CategoryID
	}

	// History missed. A constrained AI guess is allowed here (unlike the synchronous
	// manual-add paths) because the receipt review flow tolerates a brief pause, and
	// this runs outside the transaction. Guarded so a nil client is a plain skip.
	if s.gemini == nil {
		return unit, uuid.Nil
	}
	categories, err := LoadFamilyCategories(ctx, s.db, familyID)
	if err != nil {
		slog.Warn("Could not load categories for receipt AI categorize", "error", err)
		return unit, uuid.Nil
	}
	suggestion, err := s.gemini.SuggestItemDefaults(ctx, name, CategoryNames(categories))
	if err != nil {
		slog.Warn("AI categorize failed for receipt item", "name", name, "error", err)
		return unit, uuid.Nil
	}
	if unit == "" {
		unit = suggestion.Unit
	}
	if matched := MatchCategoryName(categories, suggestion.Category); matched != nil {
		return unit, *matched
	}
	return unit, uuid.Nil
}

// CategoryIDPtr converts an Item's non-pointer CategoryID into the nullable form
// ItemAlias stores, mapping the zero UUID ("uncategorized") to nil so it is recorded
// as "unknown" rather than as a category that does not exist.
func CategoryIDPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// upsertItemAlias is the internal wrapper used by receipt processing; swallows errors with a warning.
func (s *ReceiptService) upsertItemAlias(tx *gorm.DB, familyID uuid.UUID, plannedName string, receiptName string, price float64, shopID *uuid.UUID, unit string, categoryID *uuid.UUID) {
	if _, err := UpsertItemAlias(tx, familyID, plannedName, receiptName, price, shopID, unit, categoryID); err != nil {
		slog.Warn("Failed to upsert item alias", "planned", plannedName, "receipt", receiptName, "error", err)
	}
}

func (s *ReceiptService) findOrCreateShop(tx *gorm.DB, familyID uuid.UUID, storeName string) (*uuid.UUID, error) {
	var shop models.Shop
	if err := tx.Where("LOWER(name) = ? AND family_id = ?", strings.ToLower(storeName), familyID).First(&shop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			shop = models.Shop{
				TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
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

func (s *ReceiptService) updateListTitle(tx *gorm.DB, listID uuid.UUID, date time.Time) {
	if !date.IsZero() {
		var list models.ShoppingList
		if err := tx.First(&list, "id = ?", listID).Error; err == nil {
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

func (s *ReceiptService) recalculateListTotal(tx *gorm.DB, listID uuid.UUID, familyID uuid.UUID) error {
	var items []models.Item
	if err := tx.Where("list_id = ? AND family_id = ?", listID, familyID).Find(&items).Error; err != nil {
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

func (s *ReceiptService) updateItemFrequency(tx *gorm.DB, familyID uuid.UUID, name string, price float64) {
	var freq models.ItemFrequency
	if err := tx.Where("family_id = ? AND LOWER(item_name) = ?", familyID, strings.ToLower(name)).First(&freq).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			freq = models.ItemFrequency{
				FamilyID:  familyID,
				ItemName:  name,
				Frequency: 1,
				LastPrice: price,
			}
			if err := tx.Create(&freq).Error; err != nil {
				slog.Warn("Failed to create item frequency", "name", name, "error", err)
			}
		}
	} else if !freq.IsHidden {
		freq.Frequency++
		freq.LastPrice = price
		if err := tx.Save(&freq).Error; err != nil {
			slog.Warn("Failed to update item frequency", "name", name, "error", err)
		}
	}
}

// packPattern matches unambiguous multi-pack indicators in item names:
// "6×150g", "6x150g", "3-pack", "3 pack".
//
// Intentionally excluded patterns (handled by the AI prompt instead):
//   - "4+2" (promo packs): `\d+\+\d+` also matches supplement ratios like "Omega 3+6"
//   - "10ks" (Czech "kusy" = pieces): also appears in product names like "Magne B6 60ks"
//     where it denotes capsule count, not a multi-unit pack
var packPattern = regexp.MustCompile(`(?i)\b\d+\s*[×x]\s*\d+|\b\d+\s*-?\s*pack\b`)

// normalizePackItems collapses multi-pack lines that Gemini may have left with
// quantity > 1 and a per-unit price. When the item name contains an unambiguous pack
// indicator and quantity > 1, we treat the whole pack as 1 unit: price = total_price,
// unit = "pack". TotalPrice is intentionally left unchanged — after normalization the
// invariant price * quantity == total_price still holds (pack_total * 1 == pack_total).
func normalizePackItems(parsed *ai.ParsedReceipt) {
	for i := range parsed.Items {
		item := &parsed.Items[i]
		if item.Quantity <= 1 {
			continue
		}
		if packPattern.MatchString(item.Name) && item.TotalPrice > 0 {
			item.Price = item.TotalPrice
			item.Quantity = 1
			item.Unit = "pack"
		}
	}
}
