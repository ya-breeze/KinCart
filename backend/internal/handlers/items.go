package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kincart/internal/ai"
	"kincart/internal/database"
	"kincart/internal/models"
	"kincart/internal/services"
	"kincart/internal/utils"
)

// enforceBoughtAbsentExclusivity applies the "bought wins" invariant to an item
// being created. Creation handlers bind a full models.Item from JSON, so a client
// can name both flags in one payload; UpdateItem's guard never sees those requests.
func enforceBoughtAbsentExclusivity(item *models.Item) {
	if item.IsBought {
		item.IsAbsent = false
	}
}

func AddItemToList(c *gin.Context) {
	listID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	// Verify list ownership
	var list models.ShoppingList
	if err := database.DB.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	var item models.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.TenantModel.ID = uuid.New()
	item.TenantModel.FamilyID = familyID
	item.ListID = list.ID
	enforceBoughtAbsentExclusivity(&item)

	// Verify category ownership if provided
	if err := validateItemsFamily([]models.Item{item}, familyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item"})
		return
	}

	// Increment frequency
	var freq models.ItemFrequency
	result := database.DB.Where("family_id = ? AND LOWER(item_name) = LOWER(?)", familyID, item.Name).First(&freq)
	if result.Error != nil {
		// New item
		freq = models.ItemFrequency{
			FamilyID:  familyID,
			ItemName:  item.Name,
			Frequency: 1,
		}
		database.DB.Create(&freq)
	} else if !freq.IsHidden {
		// Update existing (skip if user has hidden this item)
		database.DB.Model(&freq).Update("frequency", freq.Frequency+1)
	}

	c.JSON(http.StatusCreated, item)
}

func UpdateItem(c *gin.Context) {
	itemID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	var item models.Item
	if err := database.DB.Joins("JOIN shopping_lists ON shopping_lists.id = items.list_id").
		Where("items.id = ? AND shopping_lists.family_id = ?", itemID, familyID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CategoryID if present in update
	if val, ok := updateData["category_id"]; ok && val != nil {
		catIDStr, isStr := val.(string)
		if isStr && catIDStr != "" && catIDStr != "00000000-0000-0000-0000-000000000000" {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID format"})
				return
			}
			var cat models.Category
			if err := database.DB.Where("id = ? AND family_id = ?", catID, familyID).First(&cat).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
				return
			}
		}
	}

	// If item is linked to a flyer, protect certain fields
	if item.FlyerItemID != nil {
		willUnlink := false
		if val, ok := updateData["flyer_item_id"]; ok && val == nil {
			willUnlink = true
		}

		if !willUnlink {
			delete(updateData, "name")
			delete(updateData, "price")
			delete(updateData, "local_photo_path")
		}
	}

	// Enforce is_bought / is_absent exclusivity; bought wins.
	//
	// Evaluate the state this patch would *produce* rather than judging the patch
	// in isolation: a request that clears is_bought while setting is_absent lands
	// on a legal state and must be allowed, even though it mentions is_absent on a
	// currently-bought item.
	//
	// Read as "present and boolean". A plain type assertion would treat a
	// non-bool (e.g. {"is_bought": 1}) as "field not set" and skip the checks
	// below, while GORM still wrote the coerced value -- leaving a row holding
	// both flags. Reject the malformed input instead.
	boolField := func(key string) (value bool, present bool, wellFormed bool) {
		raw, exists := updateData[key]
		if !exists {
			return false, false, true
		}
		b, isBool := raw.(bool)
		if !isBool {
			return false, true, false
		}
		return b, true, true
	}

	patchBought, setsBought, boughtWellFormed := boolField("is_bought")
	patchAbsent, setsAbsent, absentWellFormed := boolField("is_absent")
	if !boughtWellFormed || !absentWellFormed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_bought and is_absent must be booleans"})
		return
	}

	resultBought := item.IsBought
	if setsBought {
		resultBought = patchBought
	}
	resultAbsent := item.IsAbsent
	if setsAbsent {
		resultAbsent = patchAbsent
	}

	if resultBought && resultAbsent {
		if setsBought && patchBought {
			// Bought wins. Clear absent in the same Updates call so there is no
			// window in which the row holds both flags.
			updateData["is_absent"] = false
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A bought item cannot be marked absent"})
			return
		}
	}

	if err := database.DB.Model(&item).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func DeleteItem(c *gin.Context) {
	itemID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	var item models.Item
	if err := database.DB.Joins("JOIN shopping_lists ON shopping_lists.id = items.list_id").
		Where("items.id = ? AND shopping_lists.family_id = ?", itemID, familyID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Use explicit WHERE string to avoid GORM skipping zero UUID primary key
	database.DB.Where("id = ?", itemID).Delete(&models.Item{})
	c.Status(http.StatusNoContent)
}

const (
	// MaxFileSize is 10MB
	MaxFileSize = 10 << 20
)

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// validateImageFile performs security checks on uploaded files
func validateImageFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of 10MB")
	}

	if file.Size == 0 {
		return fmt.Errorf("file is empty")
	}

	// Open file to read header
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file")
	}
	defer src.Close()

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file")
	}

	// Detect MIME type from content
	mimeType := http.DetectContentType(buffer[:n])

	// Validate MIME type
	if _, ok := allowedMimeTypes[mimeType]; !ok {
		return fmt.Errorf("invalid file type: %s. Only JPEG, PNG, and WebP images are allowed", mimeType)
	}

	return nil
}

// generateSecureFilename creates a cryptographically secure random filename
func generateSecureFilename(itemID string, mimeType string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	randomStr := hex.EncodeToString(randomBytes)
	ext := allowedMimeTypes[mimeType]
	filename := fmt.Sprintf("%s_%d_%s%s", itemID, time.Now().Unix(), randomStr, ext)

	return filename, nil
}

// sanitizePath prevents path traversal attacks
func sanitizePath(basePath, filename string) (string, error) {
	cleanFilename := filepath.Base(filepath.Clean(filename))

	if strings.Contains(cleanFilename, "..") || strings.ContainsAny(cleanFilename, "/\\") {
		return "", fmt.Errorf("invalid filename")
	}

	fullPath := filepath.Join(basePath, cleanFilename)

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", err
	}

	absFull, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFull, absBase) {
		return "", fmt.Errorf("path traversal attempt detected")
	}

	return fullPath, nil
}

func AddItemPhoto(c *gin.Context) {
	itemID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	var item models.Item
	if err := database.DB.Joins("JOIN shopping_lists ON shopping_lists.id = items.list_id").
		Where("items.id = ? AND shopping_lists.family_id = ?", itemID, familyID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if item.FlyerItemID != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change photo of a flyer-linked item"})
		return
	}

	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No photo uploaded"})
		return
	}

	// Validate file
	if err = validateImageFile(file); err != nil {
		slog.Warn("Invalid file upload attempt", "ip", c.ClientIP(), "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Detect MIME type again for filename generation
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	defer src.Close()

	buffer := make([]byte, 512)
	n, _ := src.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])

	src.Close()

	// Generate secure filename
	filename, err := generateSecureFilename(itemID, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate filename"})
		return
	}

	uploadsPath := os.Getenv("UPLOADS_PATH")
	if uploadsPath == "" {
		uploadsPath = "uploads"
	}

	// 3-level sharding
	itemsBaseDir := filepath.Join(uploadsPath, "items")
	itemsDir := utils.GetShardDir(itemsBaseDir, filename)
	if err = os.MkdirAll(itemsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Sanitize path to prevent traversal
	savePath, err := sanitizePath(itemsDir, filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save photo"})
		return
	}

	// Delete old photo if exists
	if item.LocalPhotoPath != "" {
		oldPath := filepath.Join(uploadsPath, strings.TrimPrefix(item.LocalPhotoPath, "/uploads/"))
		os.Remove(oldPath) // Ignore errors, file might not exist
	}

	// Store path with sharding
	item.LocalPhotoPath = utils.GetShardedPath("/uploads/items", filename)
	if err := database.DB.Where("id = ?", itemID).Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save photo metadata"})
		return
	}

	c.JSON(http.StatusOK, item)
}

type frequentItemVariant struct {
	ReceiptName string  `json:"receipt_name"`
	ShopName    string  `json:"shop_name,omitempty"`
	LastPrice   float64 `json:"last_price"`
	Count       int     `json:"count"`
	LastUsed    string  `json:"last_used"`
}

type frequentItemResponse struct {
	ID        uint                  `json:"id"`
	ItemName  string                `json:"item_name"`
	Frequency int                   `json:"frequency"`
	LastPrice float64               `json:"last_price"`
	Variants  []frequentItemVariant `json:"variants"`
}

func GetFrequentItems(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var freqItems []models.ItemFrequency
	if err := database.DB.Where("family_id = ? AND frequency >= 2 AND is_hidden = ?", familyID, false).Order("frequency DESC").Limit(10).Find(&freqItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch frequent items"})
		return
	}

	// Enrich with alias variants
	result := make([]frequentItemResponse, 0, len(freqItems))
	for _, fi := range freqItems {
		var aliases []models.ItemAlias
		database.DB.Preload("Shop").
			Where("family_id = ? AND LOWER(planned_name) = ?", familyID, strings.ToLower(fi.ItemName)).
			Order("purchase_count DESC").
			Find(&aliases)

		variants := make([]frequentItemVariant, 0, len(aliases))
		for _, a := range aliases {
			shopName := ""
			if a.Shop != nil {
				shopName = a.Shop.Name
			}
			variants = append(variants, frequentItemVariant{
				ReceiptName: a.ReceiptName,
				ShopName:    shopName,
				LastPrice:   a.LastPrice,
				Count:       a.PurchaseCount,
				LastUsed:    a.LastUsedAt.Format("2006-01-02"),
			})
		}

		result = append(result, frequentItemResponse{
			ID:        fi.ID,
			ItemName:  fi.ItemName,
			Frequency: fi.Frequency,
			LastPrice: fi.LastPrice,
			Variants:  variants,
		})
	}

	c.JSON(http.StatusOK, result)
}

func DeleteFrequentItem(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var freq models.ItemFrequency
	if err := database.DB.Where("id = ? AND family_id = ?", id, familyID).First(&freq).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	if err := database.DB.Model(&freq).Update("is_hidden", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hide item"})
		return
	}
	c.Status(http.StatusNoContent)
}

func GetHiddenFrequentItems(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var freqItems []models.ItemFrequency
	if err := database.DB.Where("family_id = ? AND is_hidden = ?", familyID, true).Order("item_name ASC").Find(&freqItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hidden items"})
		return
	}

	result := make([]frequentItemResponse, 0, len(freqItems))
	for _, fi := range freqItems {
		result = append(result, frequentItemResponse{
			ID:        fi.ID,
			ItemName:  fi.ItemName,
			Frequency: fi.Frequency,
			LastPrice: fi.LastPrice,
			Variants:  []frequentItemVariant{},
		})
	}

	c.JSON(http.StatusOK, result)
}

func RestoreFrequentItem(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var freq models.ItemFrequency
	if err := database.DB.Where("id = ? AND family_id = ?", id, familyID).First(&freq).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	if err := database.DB.Model(&freq).Update("is_hidden", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore item"})
		return
	}
	c.Status(http.StatusNoContent)
}

type itemSuggestionVariant struct {
	AliasID     uint    `json:"alias_id"`
	ReceiptName string  `json:"receipt_name"`
	ShopName    string  `json:"shop_name,omitempty"`
	ShopID      *string `json:"shop_id,omitempty"`
	LastPrice   float64 `json:"last_price"`
	Count       int     `json:"count"`
	LastUsed    string  `json:"last_used"`
}

type itemSuggestion struct {
	PlannedName string                  `json:"planned_name"`
	Variants    []itemSuggestionVariant `json:"variants"`
}

type parseTextRequest struct {
	Text   string `json:"text" binding:"required"`
	ShopID string `json:"shop_id"`
}

type parsedItemVariant struct {
	ReceiptName string  `json:"receipt_name"`
	ShopName    string  `json:"shop_name,omitempty"`
	LastPrice   float64 `json:"last_price"`
	Count       int     `json:"count"`
}

type parsedItemResult struct {
	ai.ParsedShoppingItem
	SuggestedPrice float64             `json:"suggested_price,omitempty"`
	AliasCount     int                 `json:"alias_count,omitempty"`
	Variants       []parsedItemVariant `json:"variants,omitempty"`
}

func ParseListText(c *gin.Context) {
	listID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	var list models.ShoppingList
	if err := database.DB.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	var req parseTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The model may only pick from the family's own category names — they are
	// user-created and frequently not in English, so a free-text guess would match
	// nothing. No categories means no category property in the schema at all.
	var categories []models.Category
	if err := database.DB.Where("family_id = ?", familyID).Order("sort_order asc").Find(&categories).Error; err != nil {
		slog.Warn("Could not load categories for parse enrichment", "error", err)
	}
	categoryNames := make([]string, 0, len(categories))
	for _, cat := range categories {
		categoryNames = append(categoryNames, cat.Name)
	}

	var parsedItems []ai.ParsedShoppingItem
	geminiClient, err := ai.NewGeminiClient(c.Request.Context())
	if err != nil {
		slog.Info("Gemini unavailable, using fallback parser", "reason", err)
		parsedItems = ai.ParseShoppingTextFallback(req.Text)
	} else {
		parsedItems, err = geminiClient.ParseShoppingText(c.Request.Context(), req.Text, categoryNames)
		if err != nil {
			slog.Warn("Gemini parsing failed, using fallback parser", "error", err)
			parsedItems = ai.ParseShoppingTextFallback(req.Text)
		}
	}

	// Batch-load aliases for all parsed names (avoids N+1 queries)
	names := make([]string, len(parsedItems))
	for i, item := range parsedItems {
		names[i] = strings.ToLower(item.Name)
	}

	var aliases []models.ItemAlias
	if len(names) > 0 {
		database.DB.Preload("Shop").
			Where("family_id = ? AND LOWER(planned_name) IN ?", familyID, names).
			Order("purchase_count DESC").
			Find(&aliases)
	}

	byName := make(map[string][]models.ItemAlias)
	for _, a := range aliases {
		key := strings.ToLower(a.PlannedName)
		byName[key] = append(byName[key], a)
	}

	var shopID *uuid.UUID
	if req.ShopID != "" {
		if parsed, err := uuid.Parse(req.ShopID); err == nil {
			shopID = &parsed
		}
	}

	results := make([]parsedItemResult, len(parsedItems))
	for i, item := range parsedItems {
		matched := byName[strings.ToLower(item.Name)]
		result := parsedItemResult{ParsedShoppingItem: item, AliasCount: len(matched)}
		if len(matched) > 0 {
			// Reorder so shop-preferred alias is first (becomes default variant)
			if shopID != nil {
				for j, a := range matched {
					if a.ShopID != nil && *a.ShopID == *shopID {
						matched[0], matched[j] = matched[j], matched[0]
						break
					}
				}
			}
			result.SuggestedPrice = matched[0].LastPrice
			result.Variants = make([]parsedItemVariant, len(matched))
			for j, a := range matched {
				shopName := ""
				if a.Shop != nil {
					shopName = a.Shop.Name
				}
				result.Variants[j] = parsedItemVariant{
					ReceiptName: a.ReceiptName,
					ShopName:    shopName,
					LastPrice:   a.LastPrice,
					Count:       a.PurchaseCount,
				}
			}
		}
		results[i] = result
	}

	c.JSON(http.StatusOK, results)
}

func BulkAddItems(c *gin.Context) {
	listID := c.Param("id")
	familyID := c.MustGet("family_id").(uuid.UUID)

	var list models.ShoppingList
	if err := database.DB.Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	var items []models.Item
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No items provided"})
		return
	}

	for i := range items {
		items[i].TenantModel.ID = uuid.New()
		items[i].TenantModel.FamilyID = familyID
		items[i].ListID = list.ID
		enforceBoughtAbsentExclusivity(&items[i])
	}

	if err := database.DB.Create(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add items"})
		return
	}

	for _, item := range items {
		var freq models.ItemFrequency
		result := database.DB.Where("family_id = ? AND LOWER(item_name) = LOWER(?)", familyID, item.Name).First(&freq)
		if result.Error != nil {
			freq = models.ItemFrequency{FamilyID: familyID, ItemName: item.Name, Frequency: 1}
			database.DB.Create(&freq)
		} else if !freq.IsHidden {
			database.DB.Model(&freq).Update("frequency", freq.Frequency+1)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"created": len(items), "items": items})
}

func GetItemSuggestions(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		c.JSON(http.StatusOK, []itemSuggestion{})
		return
	}

	var aliases []models.ItemAlias
	if err := database.DB.Preload("Shop").
		Where("family_id = ? AND planned_name_lower LIKE ?", familyID, strings.ToLower(q)+"%").
		Order("purchase_count DESC").
		Limit(20).
		Find(&aliases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suggestions"})
		return
	}

	// Group by planned_name_lower (case-insensitive) preserving order; display name from first match
	seen := make(map[string]int)
	result := make([]itemSuggestion, 0)
	for _, a := range aliases {
		shopName := ""
		if a.Shop != nil {
			shopName = a.Shop.Name
		}
		var shopIDStr *string
		if a.ShopID != nil {
			s := a.ShopID.String()
			shopIDStr = &s
		}
		variant := itemSuggestionVariant{
			AliasID:     a.ID,
			ReceiptName: a.ReceiptName,
			ShopName:    shopName,
			ShopID:      shopIDStr,
			LastPrice:   a.LastPrice,
			Count:       a.PurchaseCount,
			LastUsed:    a.LastUsedAt.Format("2006-01-02"),
		}
		key := a.PlannedNameLower
		if key == "" {
			key = strings.ToLower(a.PlannedName)
		}
		if idx, ok := seen[key]; ok {
			result[idx].Variants = append(result[idx].Variants, variant)
		} else {
			seen[key] = len(result)
			result = append(result, itemSuggestion{
				PlannedName: a.PlannedName,
				Variants:    []itemSuggestionVariant{variant},
			})
		}
	}

	c.JSON(http.StatusOK, result)
}

type linkAliasRequest struct {
	PlannedItemID *string `json:"planned_item_id"` // UUID of existing list item → deleted after alias creation
	PlannedName   *string `json:"planned_name"`    // free-text → alias created only, no deletion
	ReceiptItemID string  `json:"receipt_item_id" binding:"required"`
}

// LinkItemAsAlias creates an ItemAlias mapping a planned name to a receipt/store name,
// then deletes the planned list item if planned_item_id was provided.
// POST /api/items/link-alias
func LinkItemAsAlias(c *gin.Context) {
	familyID := c.MustGet("family_id").(uuid.UUID)

	var body linkAliasRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate: exactly one of planned_item_id or planned_name must be set
	if body.PlannedItemID != nil && body.PlannedName != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide either planned_item_id or planned_name, not both"})
		return
	}
	if body.PlannedItemID == nil && body.PlannedName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "planned_item_id or planned_name is required"})
		return
	}

	// Validate planned_name if provided
	var plannedName string
	if body.PlannedName != nil {
		plannedName = strings.TrimSpace(*body.PlannedName)
		if plannedName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "planned_name must not be empty"})
			return
		}
	}

	// Parse receipt item UUID
	scannedItemID, err := uuid.Parse(body.ReceiptItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid receipt_item_id"})
		return
	}

	// Load the scanned (receipt-side) Item — scope via shopping_lists since items.family_id may be zero
	var scannedItem models.Item
	if err = database.DB.Joins("JOIN shopping_lists ON shopping_lists.id = items.list_id").
		Where("items.id = ? AND shopping_lists.family_id = ?", scannedItemID, familyID).First(&scannedItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "receipt item not found"})
		return
	}

	// Load the planned Item if planned_item_id was provided
	var plannedItem models.Item
	if body.PlannedItemID != nil {
		var plannedItemID uuid.UUID
		plannedItemID, err = uuid.Parse(*body.PlannedItemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid planned_item_id"})
			return
		}
		if err = database.DB.Joins("JOIN shopping_lists ON shopping_lists.id = items.list_id").
			Where("items.id = ? AND shopping_lists.family_id = ?", plannedItemID, familyID).First(&plannedItem).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "planned item not found"})
			return
		}
		plannedName = plannedItem.Name
	}

	// Best-effort shop resolution from the receipt chain; failures fall back to nil shop_id
	var shopID *uuid.UUID
	if scannedItem.ReceiptItemID != nil {
		var ri models.ReceiptItem
		if err = database.DB.First(&ri, *scannedItem.ReceiptItemID).Error; err != nil {
			slog.Debug("Could not load receipt item for shop resolution", "receipt_item_id", *scannedItem.ReceiptItemID, "error", err)
		} else {
			var receipt models.Receipt
			if err = database.DB.Where("id = ? AND family_id = ?", ri.ReceiptID, familyID).First(&receipt).Error; err != nil {
				slog.Debug("Could not load receipt for shop resolution", "receipt_id", ri.ReceiptID, "error", err)
			} else {
				shopID = receipt.ShopID
			}
		}
	}

	// Upsert alias
	// The scanned item is the one actually bought, so its unit/category are what
	// history should remember for this name.
	alias, err := services.UpsertItemAlias(database.DB, familyID, plannedName, scannedItem.Name, scannedItem.Price, shopID,
		scannedItem.Unit, services.CategoryIDPtr(scannedItem.CategoryID))
	if err != nil {
		slog.Error("Failed to upsert item alias", "planned", plannedName, "receipt", scannedItem.Name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alias"})
		return
	}

	// Delete planned item if one was supplied (done after upsert so alias survives a delete failure)
	if body.PlannedItemID != nil {
		if err := database.DB.Delete(&plannedItem).Error; err != nil {
			slog.Error("Failed to delete planned item after alias creation", "item_id", plannedItem.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "alias created but failed to remove planned item"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"alias": alias})
}
