package flyers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"kincart/internal/models"

	"gorm.io/gorm"
)

type Manager struct {
	db      *gorm.DB
	fetcher *Fetcher
	parser  *Parser
}

func NewManager(db *gorm.DB, fetcher *Fetcher, parser *Parser) *Manager {
	return &Manager{
		db:      db,
		fetcher: fetcher,
		parser:  parser,
	}
}

func (m *Manager) ProcessNewFlyers(ctx context.Context, folder string, fetchAttachments bool, subjects []string) error {
	slog.Info("Starting flyer processing", "folder", folder, "attachments", fetchAttachments, "subjects", subjects)

	emails, err := m.fetcher.FetchRecentFlyers(folder, fetchAttachments, subjects)
	if err != nil {
		return fmt.Errorf("failed to fetch flyers: %w", err)
	}

	for _, email := range emails {
		if len(email.Attachments) == 0 {
			continue
		}

		slog.Info("Processing email flyer", "subject", email.Subject, "from", email.From)

		// Create a temporary directory for splitting results
		tempDir, err := os.MkdirTemp("", "flyer-parse-*")
		if err != nil {
			slog.Error("Failed to create temp dir", "error", err)
			continue
		}
		defer os.RemoveAll(tempDir)

		var allAttachments []Attachment
		for _, att := range email.Attachments {
			if att.ContentType == "application/pdf" {
				slog.Info("Splitting PDF flyer", "filename", att.Filename)
				pageFiles, err := SplitPDF(att.Data, tempDir)
				if err != nil {
					slog.Error("Failed to split PDF", "filename", att.Filename, "error", err)
					// Fallback: try parsing original if split fails (maybe Gemini can handle it)
					allAttachments = append(allAttachments, att)
					continue
				}
				for _, pf := range pageFiles {
					pData, err := os.ReadFile(pf)
					if err != nil {
						continue
					}
					allAttachments = append(allAttachments, Attachment{
						Filename:    filepath.Base(pf),
						ContentType: "image/png",
						Data:        pData,
					})
				}
			} else {
				allAttachments = append(allAttachments, att)
			}
		}

		// Parse each attachment (page) separately for better accuracy
		total := len(allAttachments)
		for i, att := range allAttachments {
			slog.Info("Parsing flyer page", "current", i+1, "total", total, "file", att.Filename)
			parsed, err := m.parser.ParseFlyer(ctx, []Attachment{att})
			if err != nil {
				slog.Error("Failed to parse flyer page", "current", i+1, "total", total, "file", att.Filename, "error", err)
				continue
			}

			if err := m.SaveParsedFlyer(parsed, att.Data); err != nil {
				slog.Error("Failed to save flyer", "shop", parsed.ShopName, "error", err)
				continue
			}
		}
	}

	return nil
}

func (m *Manager) SaveParsedFlyer(parsed *ParsedFlyer, imageData []byte) error {
	// Parse dates
	layout := "2006-01-02"
	startDate, _ := time.Parse(layout, parsed.StartDate)
	endDate, _ := time.Parse(layout, parsed.EndDate)

	flyer := models.Flyer{
		ShopName:  parsed.ShopName,
		StartDate: startDate,
		EndDate:   endDate,
		ParsedAt:  time.Now(),
	}

	outputDir := "data/flyer_items"
	for _, pi := range parsed.Items {
		var localPath string
		// Currently we only support cropping from images (not PDFs)
		if imageData != nil && len(pi.BoundingBox) == 4 && len(imageData) > 4 && string(imageData[:4]) != "%PDF" {
			path, err := CropItem(imageData, pi.BoundingBox, outputDir, pi.Name)
			if err != nil {
				slog.Error("Failed to crop item", "name", pi.Name, "error", err)
			} else {
				localPath = path
			}
		}

		flyer.Items = append(flyer.Items, models.FlyerItem{
			Name:           pi.Name,
			Price:          pi.Price,
			Quantity:       pi.Quantity,
			LocalPhotoPath: localPath,
		})
	}

	if err := m.db.Create(&flyer).Error; err != nil {
		return fmt.Errorf("failed to save flyer to db: %w", err)
	}

	slog.Info("Saved flyer to database", "shop", flyer.ShopName, "items", len(flyer.Items))
	return nil
}
