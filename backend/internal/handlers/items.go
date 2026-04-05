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

	"kincart/internal/database"
	"kincart/internal/models"
	"kincart/internal/utils"
)

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

	item.ListID = list.ID

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
	result := database.DB.Where("family_id = ? AND item_name = ?", familyID, item.Name).First(&freq)
	if result.Error != nil {
		// New item
		freq = models.ItemFrequency{
			FamilyID:  familyID,
			ItemName:  item.Name,
			Frequency: 1,
		}
		database.DB.Create(&freq)
	} else {
		// Update existing
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

	database.DB.Delete(&item)
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
	database.DB.Save(&item)

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
	if err := database.DB.Where("family_id = ?", familyID).Order("frequency DESC").Limit(20).Find(&freqItems).Error; err != nil {
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
