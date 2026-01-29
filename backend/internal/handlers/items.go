package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"kincart/internal/database"
	"kincart/internal/models"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AddItemToList(c *gin.Context) {
	listID := c.Param("id")
	familyID := c.MustGet("family_id").(uint)

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
	familyID := c.MustGet("family_id").(uint)

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
		catID := uint(0)
		switch v := val.(type) {
		case float64:
			catID = uint(v)
		case int:
			catID = uint(v)
		}

		if catID != 0 {
			var cat models.Category
			if err := database.DB.Where("id = ? AND family_id = ?", catID, familyID).First(&cat).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
				return
			}
		}
	}

	// If item is linked to a flyer, protect certain fields
	// Unless the user is unlinking the item (by setting flyer_item_id to null)
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
	familyID := c.MustGet("family_id").(uint)

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
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Convert to hex string
	randomStr := hex.EncodeToString(randomBytes)

	// Get extension from MIME type
	ext := allowedMimeTypes[mimeType]

	// Create filename: itemID_timestamp_random.ext
	filename := fmt.Sprintf("%s_%d_%s%s", itemID, time.Now().Unix(), randomStr, ext)

	return filename, nil
}

// sanitizePath prevents path traversal attacks
func sanitizePath(basePath, filename string) (string, error) {
	// Clean the filename to remove any path components
	cleanFilename := filepath.Base(filepath.Clean(filename))

	// Ensure filename doesn't contain path separators
	if strings.Contains(cleanFilename, "..") || strings.ContainsAny(cleanFilename, "/\\") {
		return "", fmt.Errorf("invalid filename")
	}

	// Join with base path and clean
	fullPath := filepath.Join(basePath, cleanFilename)

	// Verify the result is still within basePath
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
	familyID := c.MustGet("family_id").(uint)

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
	if err := validateImageFile(file); err != nil {
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

	// Reset file pointer
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

	itemsDir := filepath.Join(uploadsPath, "items")
	if err := os.MkdirAll(itemsDir, 0755); err != nil {
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

	// Use relative path for storage
	item.LocalPhotoPath = "/uploads/items/" + filename
	database.DB.Save(&item)

	c.JSON(http.StatusOK, item)
}

func GetFrequentItems(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)

	var items []models.ItemFrequency
	if err := database.DB.Where("family_id = ?", familyID).Order("frequency DESC").Limit(20).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch frequent items"})
		return
	}

	c.JSON(http.StatusOK, items)
}
