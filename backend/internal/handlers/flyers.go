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
	"kincart/internal/models"
	"time"

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

func GetFlyerItems(c *gin.Context) {
	query := c.Query("q")
	shop := c.Query("shop")
	activity := c.Query("activity") // "now", "future", "all" (default "now")

	db := database.DB.Table("flyer_items").
		Select("flyer_items.*, flyers.shop_name").
		Joins("JOIN flyers ON flyers.id = flyer_items.flyer_id")

	now := time.Now().Format("2006-01-02")

	// 1. Filter by shop
	if shop != "" {
		db = db.Where("LOWER(flyers.shop_name) = LOWER(?)", shop)
	}

	// 2. Filter by search query
	if query != "" {
		q := "%" + query + "%"
		db = db.Where("(flyer_items.name LIKE ? OR flyer_items.categories LIKE ? OR flyer_items.keywords LIKE ?)", q, q, q)
	}

	// 3. Filter by activity/dates
	switch activity {
	case "future":
		db = db.Where("date(flyer_items.start_date) > ?", now)
	case "all":
		// Show everything that is not outdated
		db = db.Where("date(flyer_items.end_date) >= ?", now)
	case "now", "":
		// Default: currently active
		db = db.Where("date(flyer_items.start_date) <= ? AND date(flyer_items.end_date) >= ?", now, now)
	}

	var items []models.FlyerItem
	if err := db.Order("date(flyer_items.end_date) ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch flyer items", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func GetFlyerShops(c *gin.Context) {
	var shops []string
	if err := database.DB.Model(&models.Flyer{}).Distinct().Pluck("shop_name", &shops).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shops", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shops)
}
