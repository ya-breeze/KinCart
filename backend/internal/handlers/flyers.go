package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"kincart/internal/database"
	"kincart/internal/flyers"

	"github.com/gin-gonic/gin"
)

func ParseFlyer(c *gin.Context) {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gemini API key not configured"})
		return
	}

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

	parser, err := flyers.NewParser(geminiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize parser", "details": err.Error()})
		return
	}

	manager := flyers.NewManager(database.DB, parser)

	// Set output directory for cropped images
	uploadsPath := os.Getenv("UPLOADS_PATH")
	if uploadsPath == "" {
		uploadsPath = "./uploads"
	}
	manager.OutputDir = filepath.Join(uploadsPath, "flyer_items")

	att := flyers.Attachment{
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Data:        data,
	}

	if err := manager.ProcessAttachment(context.Background(), att); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Flyer processing failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flyer processing completed"})
}

func GetFlyers(c *gin.Context) {
	// ... implementation ...
}
