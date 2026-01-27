package main

import (
	"log/slog"
	"os"

	"kincart/internal/database"
	"kincart/internal/handlers"
	"kincart/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/subosito/gotenv"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	gotenv.Load()
	database.InitDB()

	// Start token blacklist cleanup routine
	middleware.CleanupBlacklist()

	r := gin.Default()

	// Limit multipart form memory to 10MB (matches our file size limit)
	r.MaxMultipartMemory = 10 << 20 // 10 MB

	// CORS Middleware with secure origin validation
	r.Use(middleware.CORSMiddleware())

	api := r.Group("/api")
	{
		api.POST("/auth/login", middleware.LoginRateLimiter(), handlers.Login)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
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

			protected.GET("/shops", handlers.GetShops)
			protected.POST("/shops", handlers.CreateShop)
			protected.PATCH("/shops/:id", handlers.UpdateShop)
			protected.DELETE("/shops/:id", handlers.DeleteShop)
			protected.GET("/shops/:id/order", handlers.GetShopCategoryOrder)
			protected.PATCH("/shops/:id/order", handlers.SetShopCategoryOrder)
		}

		// Internal routes (blocked by Nginx)
		internal := api.Group("/internal")
		{
			internal.POST("/flyers/parse", handlers.ParseFlyer)
		}
	}

	uploadsPath := os.Getenv("UPLOADS_PATH")
	if uploadsPath == "" {
		uploadsPath = "./uploads"
	}

	// Apply security middleware to uploads route
	uploadsGroup := r.Group("/uploads")
	uploadsGroup.Use(middleware.UploadSecurityMiddleware())
	uploadsGroup.Static("/", uploadsPath)

	slog.Info("Server starting", "port", 8080, "uploads_path", uploadsPath)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
