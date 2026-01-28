package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"kincart/internal/database"
	"kincart/internal/flyers"

	"github.com/gin-gonic/gin"
)

func ParseFlyer(c *gin.Context) {

	file, header, err := c.Request.FormFile("flyer")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get flyer file", "details": err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read flyer file", "details": err.Error()})
		return
	}

	manager := getFlyerManager(c)
	if manager == nil {
		return
	}

	att := flyers.Attachment{
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Data:        data,
	}

	shopName := c.Query("shop")
	if shopName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shop name is mandatory"})
		return
	}

	if err := manager.ProcessAttachment(context.Background(), att, shopName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Flyer processing failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flyer processing completed"})
}

func DownloadFlyers(c *gin.Context) {
	manager := getFlyerManager(c)
	if manager == nil {
		return
	}

	// Start background task
	go func() {
		ctx := context.Background()
		flyers.UpdateJobStatus(database.DB, flyers.FlyerDownloadJobName)
		if err := manager.FetchAndProcessFlyers(ctx); err != nil {
			slog.Error("Background flyer download failed", "error", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Flyer download task started in background"})
}

func getFlyerManager(c *gin.Context) *flyers.Manager {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gemini API key not configured"})
		return nil
	}

	parser, err := flyers.NewParser(geminiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize parser", "details": err.Error()})
		return nil
	}

	manager := flyers.NewManager(database.DB, parser)

	// Set output directory for cropped images
	flyerItemsPath := os.Getenv("FLYER_ITEMS_PATH")
	if flyerItemsPath == "" {
		uploadsPath := os.Getenv("UPLOADS_PATH")
		if uploadsPath == "" {
			uploadsPath = "./uploads"
		}
		flyerItemsPath = filepath.Join(uploadsPath, "flyer_items")
	}
	manager.OutputDir = flyerItemsPath
	return manager
}
