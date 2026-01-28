package flyers

import (
	"context"
	"log/slog"
	"os"
	"time"

	"kincart/internal/models"

	"gorm.io/gorm"
)

const FlyerDownloadJobName = "flyer_download"

func StartScheduler(db *gorm.DB, manager *Manager) {
	go func() {
		// First run check
		checkAndRun(db, manager)

		// Daily rotation
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			checkAndRun(db, manager)
		}
	}()
}

func checkAndRun(db *gorm.DB, manager *Manager) {
	ctx := context.Background()

	// 1. Try to download new flyers, but respect cooldown
	fetchDelay := 12 * time.Hour
	if delayEnv := os.Getenv("FLYER_FETCH_DELAY_HOURS"); delayEnv != "" {
		if hours, err := time.ParseDuration(delayEnv + "h"); err == nil {
			fetchDelay = hours
		}
	}

	var status models.JobStatus
	err := db.Where("name = ?", FlyerDownloadJobName).First(&status).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		slog.Error("Failed to check job status", "error", err)
	} else {
		canDownload := true
		if err == nil {
			// Check if it was started earlier than configured delay ago
			if time.Since(status.LastRun) < fetchDelay {
				slog.Info("Flyer download job ran recently, skipping download", "last_run", status.LastRun, "delay", fetchDelay)
				canDownload = false
			}
		}

		if canDownload {
			slog.Info("Starting scheduled flyer download job")
			UpdateJobStatus(db, FlyerDownloadJobName)
			if err := manager.DownloadNewFlyers(ctx); err != nil {
				slog.Error("Scheduled flyer download failed", "error", err)
			}
		}
	}

	// 2. Always process pending pages (no cooldown for parsing)
	slog.Info("Checking for pending flyer pages to process")
	if err := manager.ProcessPendingPages(ctx); err != nil {
		slog.Error("Scheduled flyer processing failed", "error", err)
	}
}

func UpdateJobStatus(db *gorm.DB, name string) {
	var status models.JobStatus
	err := db.Where("name = ?", name).First(&status).Error
	if err == gorm.ErrRecordNotFound {
		status = models.JobStatus{
			Name:    name,
			LastRun: time.Now(),
		}
		db.Create(&status)
	} else if err == nil {
		db.Model(&status).Update("last_run", time.Now())
	}
}
