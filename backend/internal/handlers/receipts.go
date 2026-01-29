package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"kincart/internal/ai"
	"kincart/internal/database"
	"kincart/internal/services"
)

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
			// Log error but proceed without gemini?
			// If key is set but invalid, it's better to log and maybe fail or fallback.
			// Let's fallback to nil client for robustness.
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

	// Process File
	file, err := c.FormFile("receipt")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	service := getReceiptService(c.Request.Context())

	// 1. Save & Create Receipt
	receipt, err := service.CreateReceipt(list.FamilyID, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save receipt", "details": err.Error()})
		return
	}

	// Update ListID (since CreateReceipt doesn't take it yet)
	// We need to link it early so background job can find it
	database.DB.Model(receipt).Updates(map[string]interface{}{"list_id": listID})

	// 2. Process (Async?)
	if err := service.ProcessReceipt(c.Request.Context(), receipt.ID, uint(listID)); err != nil {
		// If error is "gemini client not available", it's fine, just queued.
		if err.Error() == "gemini client not available" {
			c.JSON(http.StatusOK, gin.H{"message": "Receipt saved (queued for parsing)", "receipt_id": receipt.ID, "status": "queued"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Parsing failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Receipt processed", "receipt_id": receipt.ID, "status": "parsed"})
}
