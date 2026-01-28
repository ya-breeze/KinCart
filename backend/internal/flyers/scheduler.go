package flyers

import (
	"context"
	"log/slog"
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
	var status models.JobStatus

	err := db.Where("name = ?", FlyerDownloadJobName).First(&status).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		slog.Error("Failed to check job status", "error", err)
		return
	}

	if err == nil {
		// Check if it was started earlier than 12h ago
		if time.Since(status.LastRun) < 12*time.Hour {
			slog.Info("Flyer download job ran recently, skipping immediate start", "last_run", status.LastRun)
			return
		}
	}

	slog.Info("Starting scheduled flyer download job")

	// Update last run time immediately to prevent concurrent triggers
	UpdateJobStatus(db, FlyerDownloadJobName)

	ctx := context.Background()
	if err := manager.FetchAndProcessFlyers(ctx); err != nil {
		slog.Error("Scheduled flyer download failed", "error", err)
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
