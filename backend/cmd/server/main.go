package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"kincart/internal/ai"
	"kincart/internal/backup"
	"kincart/internal/database"
	"kincart/internal/flyers"
	"kincart/internal/handlers"
	"kincart/internal/middleware"
	"kincart/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/subosito/gotenv"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	_ = gotenv.Load() // .env file is optional
	database.InitDB()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/kincart.db"
	}
	uploadsPath := os.Getenv("UPLOADS_PATH")
	if uploadsPath == "" {
		uploadsPath = "./uploads"
	}
	flyerItemsPath := os.Getenv("FLYER_ITEMS_PATH")
	if flyerItemsPath == "" {
		flyerItemsPath = filepath.Join(uploadsPath, "flyer_items")
	}
	dataPath := os.Getenv("KINCART_DATA_PATH")
	if dataPath == "" {
		dataPath = "./kincart-data"
	}

	backup.NewTask(logger, dbPath, uploadsPath, flyerItemsPath, dataPath).Start(ctx)

	// Start token cleanup routine (blacklist + refresh tokens)
	middleware.CleanupTokens(database.DB)

	// Initialize Flyer Manager and start scheduler (disabled only if ENABLE_FLYER_SCHEDULER=false)
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if os.Getenv("ENABLE_FLYER_SCHEDULER") != "false" {
		if geminiKey == "" {
			slog.Info("Flyer scheduler skipped — GEMINI_API_KEY not set")
		} else {
			parser, err := flyers.NewParser(geminiKey)
			if err != nil {
				slog.Error("Failed to initialize flyer parser", "error", err)
			} else {
				manager := flyers.NewManager(database.DB, parser)
				manager.OutputDir = flyerItemsPath
				flyers.StartScheduler(database.DB, manager)
			}
		}
	}

	// Start Receipt Processing Scheduler every 10 minutes (disabled only if ENABLE_RECEIPT_SCHEDULER=false)
	if os.Getenv("ENABLE_RECEIPT_SCHEDULER") != "false" {
		if geminiKey == "" {
			slog.Info("Receipt scheduler skipped — GEMINI_API_KEY not set")
		} else {
			go func() {
				ticker := time.NewTicker(10 * time.Minute)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
					case <-ctx.Done():
						return
					}

					gemClient, err := ai.NewGeminiClient(ctx)
					if err != nil {
						slog.Error("Failed to init Gemini for background receipts", "error", err)
						continue
					}

					fileStorage := services.NewFileStorageService(dataPath)
					svc := services.NewReceiptService(database.DB, gemClient, fileStorage, dataPath)

					if err := svc.ProcessPendingReceipts(ctx); err != nil {
						slog.Error("Background receipt processing error", "error", err)
					}
				}
			}()
		}
	}

	r := gin.Default()

	// Limit multipart form memory to 10MB (matches our file size limit)
	r.MaxMultipartMemory = 10 << 20 // 10 MB

	// CORS Middleware with secure origin validation
	r.Use(middleware.CORSMiddleware())

	api := r.Group("/api")
	{
		api.POST("/auth/login", middleware.LoginRateLimiter(), handlers.Login)
		api.POST("/auth/refresh", handlers.Refresh)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(database.DB))
		{
			protected.GET("/auth/me", handlers.GetMe)
			protected.POST("/auth/logout", handlers.Logout)

			protected.GET("/lists", handlers.GetLists)
			protected.GET("/lists/:id", handlers.GetList)
			protected.POST("/lists", handlers.CreateList)
			protected.PATCH("/lists/:id", handlers.UpdateList)
			protected.POST("/lists/:id/duplicate", handlers.DuplicateList)
			protected.DELETE("/lists/:id", handlers.DeleteList)

			protected.POST("/lists/:id/items", handlers.AddItemToList)
			protected.POST("/lists/:id/parse-text", handlers.ParseListText)
			protected.POST("/lists/:id/items/bulk", handlers.BulkAddItems)
			protected.POST("/lists/:id/receipts", handlers.UploadReceipt)
			protected.GET("/receipts/:id/file", handlers.GetReceiptFile)
			protected.GET("/receipts/:id/matches", handlers.GetReceiptMatches)
			protected.PATCH("/receipts/:id/matches/:receipt_item_id", handlers.ConfirmReceiptItemMatch)
			protected.POST("/receipts/:id/matches/:receipt_item_id/dismiss", handlers.DismissReceiptItem)
			protected.POST("/receipts/:id/matches/confirm-all", handlers.ConfirmAllMatches)
			protected.PATCH("/items/:id", handlers.UpdateItem)
			protected.DELETE("/items/:id", handlers.DeleteItem)

			protected.POST("/items/:id/photo", handlers.AddItemPhoto)

			protected.GET("/categories", handlers.GetCategories)
			protected.POST("/categories", handlers.CreateCategory)
			protected.PATCH("/categories/:id", handlers.UpdateCategory)
			protected.DELETE("/categories/:id", handlers.DeleteCategory)
			protected.PATCH("/categories/reorder", handlers.ReorderCategories)

			protected.GET("/family/config", handlers.GetFamilyConfig)
			protected.PATCH("/family/config", handlers.UpdateFamilyConfig)
			protected.GET("/family/frequent-items", handlers.GetFrequentItems)
			protected.GET("/family/item-suggestions", handlers.GetItemSuggestions)
			protected.GET("/family/aliases", handlers.GetAliases)
			protected.POST("/family/aliases", handlers.CreateAlias)
			protected.PATCH("/family/aliases/:id", handlers.UpdateAlias)
			protected.DELETE("/family/aliases/:id", handlers.DeleteAlias)

			protected.GET("/shops", handlers.GetShops)
			protected.POST("/shops", handlers.CreateShop)
			protected.PATCH("/shops/:id", handlers.UpdateShop)
			protected.DELETE("/shops/:id", handlers.DeleteShop)
			protected.GET("/shops/:id/order", handlers.GetShopCategoryOrder)
			protected.PATCH("/shops/:id/order", handlers.SetShopCategoryOrder)

			protected.GET("/flyers/items", handlers.GetFlyerItems)
			protected.GET("/flyers/items/history", handlers.GetFlyerItemHistory)
			protected.GET("/flyers/shops", handlers.GetFlyerShops)
			protected.GET("/flyers/stats", handlers.GetFlyerStats)
			protected.GET("/flyers/activity-stats", handlers.GetFlyerActivityStats)
			protected.GET("/flyers", handlers.GetFlyers)
			protected.GET("/flyers/pages", handlers.GetFlyerPages)
			protected.GET("/flyers/activity", handlers.GetFlyerActivity)
			protected.GET("/flyers/items-detailed", handlers.GetFlyerItemsDetailed)
		}

		// Internal routes (blocked by Nginx)
		internal := api.Group("/internal")
		{
			internal.POST("/flyers/parse", handlers.ParseFlyer)
			internal.POST("/flyers/download", handlers.DownloadFlyers)
		}
	}

	// Apply security middleware to uploads route
	uploadsGroup := r.Group("/uploads")
	uploadsGroup.Use(middleware.UploadSecurityMiddleware())
	uploadsGroup.Static("/", uploadsPath)

	// Also serve flyer items. If they are in /data/flyer_items, serve them there.
	// This matches the absolute paths stored in the DB by some legacy code or docker configs.
	if strings.HasPrefix(flyerItemsPath, "/data") {
		dataGroup := r.Group("/data/flyer_items")
		dataGroup.Use(middleware.UploadSecurityMiddleware())
		dataGroup.Static("/", flyerItemsPath)
	} else {
		// Default to serving under /uploads/flyer_items if not specified as /data
		flyerGroup := r.Group("/uploads/flyer_items")
		flyerGroup.Use(middleware.UploadSecurityMiddleware())
		flyerGroup.Static("/", flyerItemsPath)
	}

	slog.Info("Server starting", "port", 8080, "uploads_path", uploadsPath)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
