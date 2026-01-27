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
	db        *gorm.DB
	parser    *Parser
	OutputDir string
}

func NewManager(db *gorm.DB, parser *Parser) *Manager {
	return &Manager{
		db:        db,
		parser:    parser,
		OutputDir: "data/flyer_items", // Default for CLI
	}
}

func (m *Manager) ProcessAttachment(ctx context.Context, att Attachment) error {
	// Create a temporary directory for splitting results
	tempDir, err := os.MkdirTemp("", "flyer-parse-upload-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var attachments []Attachment
	if att.ContentType == "application/pdf" {
		slog.Info("Splitting uploaded PDF flyer", "filename", att.Filename)
		pageFiles, err := SplitPDF(att.Data, tempDir)
		if err != nil {
			return fmt.Errorf("failed to split PDF: %w", err)
		}
		for _, pf := range pageFiles {
			pData, err := os.ReadFile(pf)
			if err != nil {
				continue
			}
			attachments = append(attachments, Attachment{
				Filename:    filepath.Base(pf),
				ContentType: "image/png",
				Data:        pData,
			})
		}
	} else {
		attachments = append(attachments, att)
	}

	total := len(attachments)
	for i, a := range attachments {
		slog.Info("Parsing flyer attachment", "current", i+1, "total", total, "file", a.Filename)
		parsed, err := m.parser.ParseFlyer(ctx, []Attachment{a})
		if err != nil {
			slog.Error("Failed to parse flyer attachment", "current", i+1, "total", total, "file", a.Filename, "error", err)
			continue
		}

		if err := m.SaveParsedFlyer(parsed, a.Data); err != nil {
			slog.Error("Failed to save flyer", "shop", parsed.ShopName, "error", err)
			continue
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

	outputDir := m.OutputDir
	if outputDir == "" {
		outputDir = "data/flyer_items"
	}
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
