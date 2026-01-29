package flyers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

func (m *Manager) ProcessAttachment(ctx context.Context, att Attachment, shopName string) error {
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

		if err := m.SaveParsedFlyer(parsed, a.Data, shopName, "", "", 0); err != nil {
			slog.Error("Failed to save flyer", "shop", shopName, "error", err)
			continue
		}
	}

	return nil
}

func (m *Manager) FetchAndProcessFlyers(ctx context.Context) error {
	if err := m.DownloadNewFlyers(ctx); err != nil {
		slog.Error("Failed to download new flyers", "error", err)
	}
	return m.ProcessPendingPages(ctx)
}

func (m *Manager) DownloadNewFlyers(ctx context.Context) error {
	crawler := NewCrawler()
	uploadsPath := os.Getenv("UPLOADS_PATH")
	if uploadsPath == "" {
		uploadsPath = "./uploads"
	}
	baseDir := filepath.Join(uploadsPath, "flyer_pages")

	delay := 500 * time.Millisecond

	for shopName := range Retailers {
		slog.Info("Fetching flyer URLs", "shop", shopName)
		flyersList, err := crawler.FetchFlyerURLs(shopName)
		if err != nil {
			slog.Error("Failed to fetch flyer URLs", "shop", shopName, "error", err)
			continue
		}

		for _, f := range flyersList {
			var flyer models.Flyer
			err := m.db.Where("url = ?", f.URL).First(&flyer).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				continue
			}

			if err == gorm.ErrRecordNotFound {
				slog.Info("Found new flyer", "title", f.Title, "url", f.URL)
				flyer = models.Flyer{
					ShopName: shopName,
					URL:      f.URL,
					ParsedAt: time.Now(), // This will be updated later when items are added
				}
				if err = m.db.Create(&flyer).Error; err != nil {
					slog.Error("Failed to create flyer record", "error", err)
					continue
				}
			}

			// Check if pages are already downloaded
			var pageCount int64
			m.db.Model(&models.FlyerPage{}).Where("flyer_id = ?", flyer.ID).Count(&pageCount)
			if pageCount > 0 {
				continue
			}

			slog.Info("Downloading pages for flyer", "shop", shopName, "url", f.URL)
			images, err := crawler.FetchFlyerImages(f.URL, delay)
			if err != nil {
				slog.Error("Failed to fetch flyer images", "url", f.URL, "error", err)
				continue
			}

			for i, imgURL := range images {
				localFilename := fmt.Sprintf("%d_page_%d.jpg", flyer.ID, i+1)
				shopDir := filepath.Join(baseDir, shopName)
				localPath := filepath.Join(shopDir, localFilename)

				if err := os.MkdirAll(shopDir, 0755); err != nil {
					slog.Error("Failed to create shop directory", "dir", shopDir, "error", err)
					continue
				}

				if err := crawler.DownloadImage(imgURL, localPath); err != nil {
					slog.Error("Failed to download page image", "url", imgURL, "error", err)
					continue
				}

				page := models.FlyerPage{
					FlyerID:   flyer.ID,
					SourceURL: imgURL,
					LocalPath: localPath,
				}
				if err := m.db.Create(&page).Error; err != nil {
					slog.Error("Failed to save page record", "error", err)
				}
			}
		}
	}
	return nil
}

func (m *Manager) ProcessPendingPages(ctx context.Context) error {
	var pages []models.FlyerPage
	// Process pages that are not parsed and have less than 3 retries
	err := m.db.Where("is_parsed = ? AND retries < ?", false, 3).Find(&pages).Error
	if err != nil {
		return fmt.Errorf("failed to fetch pending pages: %w", err)
	}

	if len(pages) == 0 {
		return nil
	}

	slog.Info("Processing pending flyer pages", "count", len(pages))

	for _, page := range pages {
		var flyer models.Flyer
		if err := m.db.First(&flyer, page.FlyerID).Error; err != nil {
			slog.Error("Flyer not found for page", "page_id", page.ID, "flyer_id", page.FlyerID)
			continue
		}

		slog.Info("Parsing flyer page", "page_id", page.ID, "path", page.LocalPath)
		data, err := os.ReadFile(page.LocalPath)
		if err != nil {
			slog.Error("Failed to read page file", "path", page.LocalPath, "error", err)
			continue
		}

		att := Attachment{
			Filename:    filepath.Base(page.LocalPath),
			ContentType: "image/jpeg",
			Data:        data,
		}

		parsed, err := m.parser.ParseFlyer(ctx, []Attachment{att})
		if err != nil {
			slog.Error("Failed to parse flyer page", "page_id", page.ID, "error", err)
			m.db.Model(&page).Updates(map[string]interface{}{
				"retries":    page.Retries + 1,
				"last_error": err.Error(),
			})
			continue
		}

		if err := m.SaveParsedFlyer(parsed, data, flyer.ShopName, flyer.URL, page.SourceURL, page.ID); err != nil {
			slog.Error("Failed to save flyer items", "page_id", page.ID, "error", err)
			m.db.Model(&page).Update("last_error", err.Error())
			continue
		}

		// Mark as parsed
		m.db.Model(&page).Update("is_parsed", true)
	}

	return nil
}

func (m *Manager) SaveParsedFlyer(parsed *ParsedFlyer, imageData []byte, shopName string, flyerURL string, photoURL string, pageID uint) error {
	// Parse dates
	layout := "2006-01-02"
	startDate, _ := time.Parse(layout, parsed.StartDate)
	endDate, _ := time.Parse(layout, parsed.EndDate)

	var flyer models.Flyer
	err := m.db.Where("url = ?", flyerURL).First(&flyer).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing flyer: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		flyer = models.Flyer{
			ShopName:  shopName,
			URL:       flyerURL,
			StartDate: startDate,
			EndDate:   endDate,
			ParsedAt:  time.Now(),
		}
		if err := m.db.Create(&flyer).Error; err != nil {
			return fmt.Errorf("failed to create flyer: %w", err)
		}
	} else {
		// Update dates if they were not set (e.g. created by DownloadNewFlyers without dates)
		updates := make(map[string]interface{})
		if flyer.StartDate.IsZero() && !startDate.IsZero() {
			updates["start_date"] = startDate
		}
		if flyer.EndDate.IsZero() && !endDate.IsZero() {
			updates["end_date"] = endDate
		}
		if len(updates) > 0 {
			m.db.Model(&flyer).Updates(updates)
		}
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

		itemStartDate, err := time.Parse(layout, pi.StartDate)
		if err != nil {
			itemStartDate = startDate
		}
		itemEndDate, err := time.Parse(layout, pi.EndDate)
		if err != nil {
			itemEndDate = endDate
		}

		flyerItem := models.FlyerItem{
			FlyerID:        flyer.ID,
			FlyerPageID:    pageID,
			Name:           pi.Name,
			Price:          pi.Price,
			OriginalPrice:  pi.OriginalPrice,
			Quantity:       pi.Quantity,
			StartDate:      itemStartDate,
			EndDate:        itemEndDate,
			PhotoURL:       photoURL,
			LocalPhotoPath: localPath,
			Categories:     strings.Join(pi.Categories, ", "),
			Keywords:       strings.Join(pi.Keywords, ", "),
		}

		if err := m.db.Create(&flyerItem).Error; err != nil {
			slog.Error("Failed to save flyer item", "name", pi.Name, "error", err)
		}
	}

	slog.Info("Processed flyer items", "shop", flyer.ShopName, "items", len(parsed.Items))
	return nil
}
