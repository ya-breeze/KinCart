package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"kincart/internal/ai"
	"kincart/internal/database"
	"kincart/internal/models"
	"kincart/internal/services"
)

const maxReceiptTextBytes = 100 * 1024 // 100 KB

// receiptSvc is the interface used by the upload handler (enables testing with mocks).
type receiptSvc interface {
	CreateReceipt(familyID uint, file *multipart.FileHeader) (*models.Receipt, error)
	CreateReceiptFromText(familyID uint, text string) (*models.Receipt, error)
	ProcessReceipt(ctx context.Context, receiptID uint, listID uint) error
}

// Helper to get service instance (in a real app, use dependency injection)
func getReceiptService(ctx context.Context) *services.ReceiptService {
	dataPath := os.Getenv("KINCART_DATA_PATH")
	if dataPath == "" {
		dataPath = "./kincart-data"
	}

	fileStorage := services.NewFileStorageService(dataPath)

	geminiKey := os.Getenv("GEMINI_API_KEY")
	var geminiClient services.ReceiptParser

	if geminiKey != "" {
		client, err := ai.NewGeminiClient(ctx)
		if err != nil {
			slog.Warn("Failed to init gemini client", "error", err)
			geminiClient = nil
		} else {
			geminiClient = client
		}
	}

	return services.NewReceiptService(database.DB, geminiClient, fileStorage, dataPath)
}

// UploadReceipt handles the receipt upload request.
// POST /api/lists/:id/receipts
func UploadReceipt(c *gin.Context) {
	svc := getReceiptService(c.Request.Context())
	uploadReceiptWith(c, svc)
}

// uploadReceiptWith is the testable core of UploadReceipt.
// It accepts either:
//   - application/json body with {"receipt_text": "..."}
//   - multipart/form-data with field "receipt" (image, PDF, or .txt)
func uploadReceiptWith(c *gin.Context, svc receiptSvc) {
	listIDStr := c.Param("id")
	listID, err := strconv.ParseUint(listIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid list ID"})
		return
	}

	var list struct {
		FamilyID uint
	}
	if dbErr := database.DB.Table("shopping_lists").Select("family_id").Where("id = ?", listID).First(&list).Error; dbErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "List not found"})
		return
	}

	// Determine request mode from Content-Type
	contentType := c.GetHeader("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)

	var receipt *models.Receipt

	if mediaType == "application/json" {
		// --- JSON paste mode ---
		var req struct {
			ReceiptText string `json:"receipt_text"`
		}
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
			return
		}

		if len(req.ReceiptText) > maxReceiptTextBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Receipt text exceeds 100KB limit"})
			return
		}

		if strings.TrimSpace(req.ReceiptText) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "receipt_text is required and must not be empty"})
			return
		}

		receipt, err = svc.CreateReceiptFromText(list.FamilyID, req.ReceiptText)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save receipt", "details": err.Error()})
			return
		}

	} else {
		// --- Multipart file upload mode ---
		file, fileErr := c.FormFile("receipt")
		if fileErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == ".txt" {
			// Validate and read .txt file
			if file.Size > maxReceiptTextBytes {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds 100KB limit"})
				return
			}

			src, openErr := file.Open()
			if openErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
				return
			}
			defer src.Close()

			data, readErr := io.ReadAll(src)
			if readErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
				return
			}

			// Strip UTF-8 BOM if present
			data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

			if !utf8.Valid(data) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "File must be valid UTF-8 encoded text"})
				return
			}

			if strings.TrimSpace(string(data)) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Receipt text is empty"})
				return
			}

			receipt, err = svc.CreateReceiptFromText(list.FamilyID, string(data))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save receipt", "details": err.Error()})
				return
			}

		} else {
			// Image / PDF upload (existing path)
			receipt, err = svc.CreateReceipt(list.FamilyID, file)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save receipt", "details": err.Error()})
				return
			}
		}
	}

	// Link receipt to the list so the background ticker can find it
	database.DB.Model(receipt).Updates(map[string]interface{}{"list_id": listID})

	// Process (synchronous; gracefully handles missing Gemini key)
	if err := svc.ProcessReceipt(c.Request.Context(), receipt.ID, uint(listID)); err != nil {
		if err.Error() == "gemini client not available" {
			c.JSON(http.StatusOK, gin.H{"message": "Receipt saved (queued for parsing)", "receipt_id": receipt.ID, "status": "queued"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Parsing failed", "details": err.Error()})
		return
	}

	// Read the actual status set by ProcessReceipt (may be "parsed" or "pending_review")
	status := "parsed"
	if receipt.ID != 0 {
		var updated models.Receipt
		if dbErr := database.DB.Select("status").First(&updated, receipt.ID).Error; dbErr == nil {
			status = updated.Status
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Receipt processed", "receipt_id": receipt.ID, "status": status})
}

// GetReceiptMatches returns the receipt with AI match suggestions for user review.
// GET /api/receipts/:id/matches
func GetReceiptMatches(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	receiptID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt ID"})
		return
	}

	svc := getReceiptService(c.Request.Context())
	resp, err := svc.GetReceiptMatches(uint(receiptID), familyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ConfirmReceiptItemMatch confirms or changes the match for a single receipt item.
// PATCH /api/receipts/:id/matches/:receipt_item_id
// Body: {"planned_item_id": 7}  or  {"planned_item_id": null}
func ConfirmReceiptItemMatch(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	receiptItemID, err := strconv.ParseUint(c.Param("receipt_item_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt item ID"})
		return
	}

	var body struct {
		PlannedItemID *uint `json:"planned_item_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	svc := getReceiptService(c.Request.Context())
	if err := svc.ConfirmMatch(c.Request.Context(), uint(receiptItemID), body.PlannedItemID, familyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Match confirmed"})
}

// DismissReceiptItem marks a receipt item as dismissed (not relevant to the list).
// POST /api/receipts/:id/matches/:receipt_item_id/dismiss
func DismissReceiptItem(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	receiptItemID, err := strconv.ParseUint(c.Param("receipt_item_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt item ID"})
		return
	}

	svc := getReceiptService(c.Request.Context())
	if err := svc.DismissReceiptItem(uint(receiptItemID), familyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item dismissed"})
}

// ConfirmAllMatches accepts all auto-matches and creates new items for unmatched receipt items.
// POST /api/receipts/:id/matches/confirm-all
func ConfirmAllMatches(c *gin.Context) {
	familyID := c.MustGet("family_id").(uint)
	receiptID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt ID"})
		return
	}

	svc := getReceiptService(c.Request.Context())
	if err := svc.ConfirmAllMatches(c.Request.Context(), uint(receiptID), familyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All matches confirmed"})
}

// GetReceiptFile serves the raw receipt file (image, PDF, or text).
// GET /api/receipts/:id/file
func GetReceiptFile(c *gin.Context) {
	dataPath := os.Getenv("KINCART_DATA_PATH")
	if dataPath == "" {
		dataPath = "./kincart-data"
	}
	getReceiptFileWith(c, dataPath)
}

// getReceiptFileWith is the testable core of GetReceiptFile.
func getReceiptFileWith(c *gin.Context, dataPath string) {
	familyID := c.MustGet("family_id").(uint)
	receiptIDStr := c.Param("id")
	receiptID, err := strconv.ParseUint(receiptIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt ID"})
		return
	}

	var receipt models.Receipt
	if dbErr := database.DB.Preload("Shop").
		Where("id = ? AND family_id = ?", receiptID, familyID).
		First(&receipt).Error; dbErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt not found"})
		return
	}

	rawDataPath, _ := filepath.Abs(dataPath)
	// EvalSymlinks resolves macOS /var → /private/var etc., keeping tests green on all platforms.
	absDataPath, err := filepath.EvalSymlinks(rawDataPath)
	if err != nil {
		absDataPath = rawDataPath
	}
	absFilePath := filepath.Join(absDataPath, receipt.ImagePath)

	// Path traversal guard (uses "/" intentionally — project runs on Linux/Docker)
	if !strings.HasPrefix(absFilePath, absDataPath+"/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	if _, statErr := os.Stat(absFilePath); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Error("Receipt file missing on disk", "path", absFilePath, "receipt_id", receipt.ID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Receipt file not found"})
		} else {
			slog.Error("Failed to stat receipt file", "path", absFilePath, "error", statErr, "receipt_id", receipt.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access receipt file"})
		}
		return
	}

	// Derive a friendly download filename
	ext := strings.TrimPrefix(filepath.Ext(receipt.ImagePath), ".")
	if ext == "" {
		ext = "bin"
	}
	date := receipt.Date.Format("2006-01-02")
	var filename string
	if receipt.Shop != nil {
		shopSlug := sanitizeSlug(receipt.Shop.Name)
		filename = "receipt-" + date + "-" + shopSlug + "." + ext
	} else {
		filename = fmt.Sprintf("receipt-%d.%s", receipt.ID, ext)
	}

	c.FileAttachment(absFilePath, filename)
}

// sanitizeSlug converts a shop name to a safe filename slug.
func sanitizeSlug(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
