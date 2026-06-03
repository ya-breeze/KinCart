package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"kincart/internal/models"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver for VACUUM INTO
	"gorm.io/gorm"
)

const (
	backupDateFormat = "2006-01-02"
	backupPrefix     = "kincart-backup-"
	backupSuffix     = ".tar.gz"
	backupsDirName   = "kincart-backups"
	defaultMaxCount  = 10
	defaultInterval  = 24 * time.Hour
	startupDelay     = 30 * time.Second
)

// Task creates daily .tar.gz backups of the database, uploads, and flyer items.
type Task struct {
	logger         *slog.Logger
	db             *gorm.DB
	dbPath         string
	uploadsPath    string
	flyerItemsPath string
	backupDir      string
	interval       time.Duration
	maxCount       int
}

func NewTask(logger *slog.Logger, db *gorm.DB, dbPath, uploadsPath, flyerItemsPath, dataPath string) *Task {
	interval := defaultInterval
	if v := os.Getenv("KINCART_BACKUP_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		} else {
			logger.Warn("Invalid KINCART_BACKUP_INTERVAL, using 24h", "value", v, "error", err)
		}
	}

	maxCount := defaultMaxCount
	if v := os.Getenv("KINCART_BACKUP_MAX_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxCount = n
		}
	}

	return &Task{
		logger:         logger,
		db:             db,
		dbPath:         dbPath,
		uploadsPath:    uploadsPath,
		flyerItemsPath: flyerItemsPath,
		backupDir:      filepath.Join(dataPath, backupsDirName),
		interval:       interval,
		maxCount:       maxCount,
	}
}

// Start launches the background goroutine. It waits 30s on startup, then runs on the ticker.
func (t *Task) Start(ctx context.Context) {
	go func() {
		select {
		case <-time.After(startupDelay):
		case <-ctx.Done():
			return
		}
		t.run()
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.run()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// cleanupExpiredFlyerImages deletes local image files for flyers that expired
// more than 30 days ago and clears the corresponding DB path columns.
// Effective expiry = EndDate if set, else CreatedAt. Idempotent: missing files
// are logged as warnings and the DB column is cleared regardless.
func (t *Task) cleanupExpiredFlyerImages() {
	if t.db == nil {
		return
	}

	threshold := time.Now().Add(-30 * 24 * time.Hour)
	// Dates before year 2000 are treated as unset (zero value from GORM).
	cutoff := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	var flyers []models.Flyer
	if err := t.db.Where(
		"(end_date > ? AND end_date < ?) OR (end_date <= ? AND created_at < ?)",
		cutoff, threshold, cutoff, threshold,
	).Find(&flyers).Error; err != nil {
		t.logger.Error("cleanup: failed to query eligible flyers", "error", err)
		return
	}

	if len(flyers) == 0 {
		t.logger.Info("cleanup: no expired flyers eligible for image cleanup")
		return
	}

	t.logger.Info("cleanup: starting image expiry", "flyers", len(flyers))

	for _, flyer := range flyers {
		// --- FlyerPage source files ---
		var pages []models.FlyerPage
		if err := t.db.Where("flyer_id = ? AND local_path != ''", flyer.ID).Find(&pages).Error; err != nil {
			t.logger.Error("cleanup: failed to query pages", "flyer_id", flyer.ID, "error", err)
		} else {
			for _, page := range pages {
				if err := os.Remove(page.LocalPath); err != nil && !os.IsNotExist(err) {
					t.logger.Warn("cleanup: failed to delete page file", "path", page.LocalPath, "error", err)
				}
			}
			if err := t.db.Model(&models.FlyerPage{}).Where("flyer_id = ?", flyer.ID).Update("local_path", "").Error; err != nil {
				t.logger.Error("cleanup: failed to clear page paths in DB", "flyer_id", flyer.ID, "error", err)
			}
		}

		// --- FlyerItem crop files ---
		var items []models.FlyerItem
		if err := t.db.Where("flyer_id = ? AND local_photo_path != ''", flyer.ID).Find(&items).Error; err != nil {
			t.logger.Error("cleanup: failed to query items", "flyer_id", flyer.ID, "error", err)
		} else {
			for _, item := range items {
				if err := os.Remove(item.LocalPhotoPath); err != nil && !os.IsNotExist(err) {
					t.logger.Warn("cleanup: failed to delete item file", "path", item.LocalPhotoPath, "error", err)
				}
			}
			if err := t.db.Model(&models.FlyerItem{}).Where("flyer_id = ?", flyer.ID).Update("local_photo_path", "").Error; err != nil {
				t.logger.Error("cleanup: failed to clear item paths in DB", "flyer_id", flyer.ID, "error", err)
			}
		}

		t.logger.Info("cleanup: processed flyer", "flyer_id", flyer.ID, "shop", flyer.ShopName, "pages", len(pages), "items", len(items))
	}

	t.logger.Info("cleanup: image expiry complete", "flyers_processed", len(flyers))
}

func (t *Task) run() {
	today := time.Now().Format(backupDateFormat)

	if err := os.MkdirAll(t.backupDir, 0o750); err != nil {
		t.logger.Error("backup: failed to create backup directory", "error", err)
		return
	}

	// Clean up any stale temp files left by a previous crash.
	if entries, err := os.ReadDir(t.backupDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") {
				_ = os.Remove(filepath.Join(t.backupDir, e.Name()))
			}
		}
	}

	archivePath := filepath.Join(t.backupDir, backupPrefix+today+backupSuffix)
	if _, err := os.Stat(archivePath); err == nil {
		t.logger.Info("backup: today's backup already exists, skipping", "date", today)
		return
	}

	t.logger.Info("backup: starting", "date", today)

	t.cleanupExpiredFlyerImages()

	tmpDB := archivePath + ".db.tmp"
	defer os.Remove(tmpDB) //nolint:errcheck

	if err := vacuumInto(t.dbPath, tmpDB); err != nil {
		t.logger.Error("backup: VACUUM INTO failed", "error", err)
		return
	}

	tmpArchive := archivePath + ".tmp"
	defer os.Remove(tmpArchive) //nolint:errcheck

	if err := t.createArchive(tmpArchive, tmpDB); err != nil {
		t.logger.Error("backup: failed to create archive", "error", err)
		return
	}

	if err := os.Rename(tmpArchive, archivePath); err != nil {
		t.logger.Error("backup: failed to finalize archive", "error", err)
		return
	}

	t.logger.Info("backup: completed", "file", filepath.Base(archivePath))

	if err := t.pruneBackups(); err != nil {
		t.logger.Error("backup: pruning failed", "error", err)
	}
}

// vacuumInto executes VACUUM INTO using a fresh database/sql connection.
// This produces an atomic, consistent copy without requiring GORM access.
func vacuumInto(src, dst string) error {
	db, err := sql.Open("sqlite3", src)
	if err != nil {
		return fmt.Errorf("open source db: %w", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.Exec("VACUUM INTO ?", dst); err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	return nil
}

func (t *Task) createArchive(archivePath, tmpDB string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer f.Close() //nolint:errcheck

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	if err := addFileToTar(tw, tmpDB, "kincart.db"); err != nil {
		return fmt.Errorf("add db: %w", err)
	}
	if _, err := os.Stat(t.uploadsPath); err == nil {
		if err := addDirToTar(tw, t.uploadsPath, "uploads"); err != nil {
			return fmt.Errorf("add uploads: %w", err)
		}
	}
	// Only include flyer_items separately if it is not nested inside uploads
	// (default dev layout puts it at uploads/flyer_items, which is already covered).
	if rel, err := filepath.Rel(t.uploadsPath, t.flyerItemsPath); err != nil || strings.HasPrefix(rel, "..") {
		if _, err := os.Stat(t.flyerItemsPath); err == nil {
			if err := addDirToTar(tw, t.flyerItemsPath, "flyer_items"); err != nil {
				return fmt.Errorf("add flyer_items: %w", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("finalize tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("finalize gzip: %w", err)
	}
	return nil
}

func addFileToTar(tw *tar.Writer, srcPath, archiveName string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	info, err := f.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    archiveName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}
	if err = tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

func addDirToTar(tw *tar.Writer, srcDir, archiveBase string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		archivePath := filepath.Join(archiveBase, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = archivePath
		if d.IsDir() {
			hdr.Name += "/"
		}
		if err = tw.WriteHeader(hdr); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
		_, err = io.Copy(tw, f)
		return err
	})
}

func (t *Task) pruneBackups() error {
	entries, err := os.ReadDir(t.backupDir)
	if err != nil {
		return fmt.Errorf("read backup dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() &&
			len(n) > len(backupPrefix)+len(backupSuffix) &&
			n[:len(backupPrefix)] == backupPrefix &&
			n[len(n)-len(backupSuffix):] == backupSuffix {
			names = append(names, n)
		}
	}

	sort.Strings(names) // lexicographic == chronological for YYYY-MM-DD names

	for len(names) > t.maxCount {
		oldest := names[0]
		names = names[1:]
		if err := os.Remove(filepath.Join(t.backupDir, oldest)); err != nil {
			t.logger.Warn("backup: failed to delete old backup", "file", oldest, "error", err)
		} else {
			t.logger.Info("backup: deleted old backup", "file", oldest)
		}
	}
	return nil
}
