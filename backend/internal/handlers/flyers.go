package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kincart/internal/database"
	"kincart/internal/flyers"
	"kincart/internal/models"
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

	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))

	// Validate and cap pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 24
	}

	offset := (page - 1) * limit

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

	// Get total count before applying pagination
	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count items", "details": err.Error()})
		return
	}

	// Apply pagination
	var items []models.FlyerItem
	if err := db.Order("date(flyer_items.end_date) ASC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch flyer items", "details": err.Error()})
		return
	}

	// Calculate pagination metadata
	totalPages := (totalCount + int64(limit) - 1) / int64(limit)
	hasMore := offset+len(items) < int(totalCount)

	// Add cache headers (5 minutes)
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("Vary", "Authorization")

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

func GetFlyerShops(c *gin.Context) {
	var shops []string
	if err := database.DB.Model(&models.Flyer{}).Distinct().Pluck("shop_name", &shops).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shops", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shops)
}

func GetFlyerStats(c *gin.Context) {
	var stats struct {
		TotalFlyers int64 `json:"total_flyers"`
		TotalPages  int64 `json:"total_pages"`
		ParsedPages int64 `json:"parsed_pages"`
		ErrorPages  int64 `json:"error_pages"`
		TotalItems  int64 `json:"total_items"`
		History     []struct {
			Date   string `json:"date"`
			Total  int64  `json:"total"`
			Parsed int64  `json:"parsed"`
			Errors int64  `json:"errors"`
		} `json:"history"`
	}

	database.DB.Model(&models.Flyer{}).Count(&stats.TotalFlyers)
	database.DB.Model(&models.FlyerPage{}).Count(&stats.TotalPages)
	database.DB.Model(&models.FlyerPage{}).Where("is_parsed = ?", true).Count(&stats.ParsedPages)
	database.DB.Model(&models.FlyerPage{}).Where("last_error != ?", "").Count(&stats.ErrorPages)
	database.DB.Model(&models.FlyerItem{}).Count(&stats.TotalItems)

	// Fetch last 14 days of history
	database.DB.Raw(`
		SELECT 
			DATE(updated_at) as date,
			COUNT(*) as total,
			SUM(CASE WHEN is_parsed = 1 THEN 1 ELSE 0 END) as parsed,
			SUM(CASE WHEN last_error != '' THEN 1 ELSE 0 END) as errors
		FROM flyer_pages
		WHERE updated_at > ?
		GROUP BY DATE(updated_at)
		ORDER BY DATE(updated_at) ASC
	`, time.Now().AddDate(0, 0, -14)).Scan(&stats.History)

	c.JSON(http.StatusOK, stats)
}

func GetFlyers(c *gin.Context) {
	var flyers []models.Flyer
	if err := database.DB.Order("created_at DESC").Find(&flyers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch flyers", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flyers)
}

func GetFlyerPages(c *gin.Context) {
	flyerID := c.Query("flyer_id")
	isParsed := c.Query("is_parsed")
	hasError := c.Query("has_error")
	date := c.Query("date")

	db := database.DB.Table("flyer_pages").
		Select("flyer_pages.*, flyers.shop_name").
		Joins("JOIN flyers ON flyers.id = flyer_pages.flyer_id")

	if date != "" {
		db = db.Where("DATE(flyer_pages.updated_at) = ?", date)
	}
	if flyerID != "" {
		db = db.Where("flyer_pages.flyer_id = ?", flyerID)
	}
	if isParsed == "true" {
		db = db.Where("flyer_pages.is_parsed = ?", true)
	} else if isParsed == "false" {
		db = db.Where("flyer_pages.is_parsed = ?", false)
	}
	if hasError == "true" {
		db = db.Where("flyer_pages.last_error != ?", "")
	}

	var pages []models.FlyerPage
	if err := db.Order("flyer_pages.updated_at DESC").Find(&pages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch flyer pages", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pages)
}

func GetFlyerItemsDetailed(c *gin.Context) {
	date := c.Query("date")
	db := database.DB.Model(&models.FlyerItem{})

	if date != "" {
		db = db.Where("DATE(created_at) = ?", date)
	}

	var items []models.FlyerItem
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func GetFlyerActivityStats(c *gin.Context) {
	var activityStats struct {
		LatestDate string `json:"latest_date"`
	}

	database.DB.Raw(`
		SELECT MAX(date_raw) FROM (
			SELECT DATE(updated_at) as date_raw FROM flyer_pages
			UNION
			SELECT DATE(created_at) as date_raw FROM flyer_items
		) t
	`).Scan(&activityStats.LatestDate)

	c.JSON(http.StatusOK, activityStats)
}

func GetFlyerActivity(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date parameter is required"})
		return
	}

	var activity struct {
		Pages []models.FlyerPage `json:"pages"`
		Items []models.FlyerItem `json:"items"`
	}

	if err := database.DB.Table("flyer_pages").
		Select("flyer_pages.*, flyers.shop_name").
		Joins("JOIN flyers ON flyers.id = flyer_pages.flyer_id").
		Where("DATE(flyer_pages.updated_at) = ?", date).
		Order("flyer_pages.updated_at DESC").
		Find(&activity.Pages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch daily activity", "details": err.Error()})
		return
	}

	// Also items created on that day
	if err := database.DB.Where("DATE(created_at) = ?", date).Order("created_at DESC").Find(&activity.Items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch daily items", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, activity)
}
