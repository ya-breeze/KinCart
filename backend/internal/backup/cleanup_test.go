package backup

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kincart/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestTask(t *testing.T) (*Task, *gorm.DB, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Flyer{}, &models.FlyerPage{}, &models.FlyerItem{}))

	tmpDir := t.TempDir()
	task := &Task{
		logger: slog.Default(),
		db:     db,
	}
	return task, db, tmpDir
}

func createFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("img"), 0644))
	return path
}

func TestCleanupEligible_EndDateSet(t *testing.T) {
	task, db, dir := newTestTask(t)

	expiredEnd := time.Now().Add(-40 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: expiredEnd, CreatedAt: expiredEnd}
	require.NoError(t, db.Create(&flyer).Error)

	pagePath := createFile(t, dir, "page.jpg")
	page := models.FlyerPage{FlyerID: flyer.ID, LocalPath: pagePath, SourceURL: "http://example.com/p.jpg"}
	require.NoError(t, db.Create(&page).Error)

	itemPath := createFile(t, dir, "item.png")
	photoURL := "http://example.com/photo.jpg"
	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: itemPath, PhotoURL: photoURL, Name: "Milk"}
	require.NoError(t, db.Create(&item).Error)

	task.cleanupExpiredFlyerImages()

	// Files deleted
	_, err := os.Stat(pagePath)
	assert.True(t, os.IsNotExist(err), "page file should be deleted")
	_, err = os.Stat(itemPath)
	assert.True(t, os.IsNotExist(err), "item file should be deleted")

	// DB paths cleared
	var updatedPage models.FlyerPage
	require.NoError(t, db.First(&updatedPage, page.ID).Error)
	assert.Empty(t, updatedPage.LocalPath)

	var updatedItem models.FlyerItem
	require.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Empty(t, updatedItem.LocalPhotoPath)

	// PhotoURL preserved
	assert.Equal(t, photoURL, updatedItem.PhotoURL)
}

func TestCleanupEligible_EndDateZero_FallsBackToCreatedAt(t *testing.T) {
	task, db, dir := newTestTask(t)

	oldCreated := time.Now().Add(-40 * 24 * time.Hour)
	// EndDate is zero (not set)
	flyer := models.Flyer{ShopName: "testshop", CreatedAt: oldCreated}
	require.NoError(t, db.Create(&flyer).Error)

	itemPath := createFile(t, dir, "item.png")
	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: itemPath, Name: "Butter"}
	require.NoError(t, db.Create(&item).Error)

	task.cleanupExpiredFlyerImages()

	_, err := os.Stat(itemPath)
	assert.True(t, os.IsNotExist(err), "item file should be deleted")

	var updatedItem models.FlyerItem
	require.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Empty(t, updatedItem.LocalPhotoPath)
}

func TestCleanupNotEligible_RecentFlyer(t *testing.T) {
	task, db, dir := newTestTask(t)

	recentEnd := time.Now().Add(-10 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: recentEnd}
	require.NoError(t, db.Create(&flyer).Error)

	itemPath := createFile(t, dir, "item.png")
	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: itemPath, Name: "Cheese"}
	require.NoError(t, db.Create(&item).Error)

	task.cleanupExpiredFlyerImages()

	// File should still exist
	_, err := os.Stat(itemPath)
	assert.NoError(t, err, "file should not be deleted for recent flyer")

	var updatedItem models.FlyerItem
	require.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Equal(t, itemPath, updatedItem.LocalPhotoPath)
}

func TestCleanupIdempotent_MissingFile(t *testing.T) {
	task, db, _ := newTestTask(t)

	expiredEnd := time.Now().Add(-40 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: expiredEnd}
	require.NoError(t, db.Create(&flyer).Error)

	// Point to a file that doesn't exist
	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: "/tmp/nonexistent_kincart_test.png", Name: "Sugar"}
	require.NoError(t, db.Create(&item).Error)

	// Should not panic or error
	assert.NotPanics(t, func() { task.cleanupExpiredFlyerImages() })

	// DB path still cleared (ENOENT is treated as success)
	var updatedItem models.FlyerItem
	require.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Empty(t, updatedItem.LocalPhotoPath)
}

func TestCleanupFileDeleteFailure_PathNotCleared(t *testing.T) {
	task, db, dir := newTestTask(t)

	expiredEnd := time.Now().Add(-40 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: expiredEnd}
	require.NoError(t, db.Create(&flyer).Error)

	// Create a read-only directory so os.Remove will fail with EACCES
	roDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(roDir, 0755))
	protectedPath := filepath.Join(roDir, "item.png")
	require.NoError(t, os.WriteFile(protectedPath, []byte("img"), 0644))
	require.NoError(t, os.Chmod(roDir, 0555)) // remove write permission from dir
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: protectedPath, Name: "Locked"}
	require.NoError(t, db.Create(&item).Error)

	task.cleanupExpiredFlyerImages()

	// File must still exist (deletion failed)
	_, err := os.Stat(protectedPath)
	assert.NoError(t, err, "file should not be deleted when os.Remove fails")

	// DB path must NOT be cleared (so the next run can retry)
	var updatedItem models.FlyerItem
	require.NoError(t, db.Unscoped().First(&updatedItem, item.ID).Error)
	assert.Equal(t, protectedPath, updatedItem.LocalPhotoPath, "path should be preserved when deletion failed")
}

func TestCleanupPartialSuccess_OnlyClearedPathsUpdated(t *testing.T) {
	task, db, dir := newTestTask(t)

	expiredEnd := time.Now().Add(-40 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: expiredEnd}
	require.NoError(t, db.Create(&flyer).Error)

	// Page A: deletable
	pageAPath := createFile(t, dir, "page_a.jpg")
	pageA := models.FlyerPage{FlyerID: flyer.ID, LocalPath: pageAPath, SourceURL: "http://example.com/a.jpg"}
	require.NoError(t, db.Create(&pageA).Error)

	// Page B: locked directory — os.Remove will fail with EACCES
	roDir := filepath.Join(dir, "locked")
	require.NoError(t, os.MkdirAll(roDir, 0755))
	pageBPath := filepath.Join(roDir, "page_b.jpg")
	require.NoError(t, os.WriteFile(pageBPath, []byte("img"), 0644))
	require.NoError(t, os.Chmod(roDir, 0555))
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	pageB := models.FlyerPage{FlyerID: flyer.ID, LocalPath: pageBPath, SourceURL: "http://example.com/b.jpg"}
	require.NoError(t, db.Create(&pageB).Error)

	task.cleanupExpiredFlyerImages()

	// Page A: file gone, DB path cleared
	_, err := os.Stat(pageAPath)
	assert.True(t, os.IsNotExist(err), "page A file should be deleted")
	var updatedA models.FlyerPage
	require.NoError(t, db.First(&updatedA, pageA.ID).Error)
	assert.Empty(t, updatedA.LocalPath, "page A DB path should be cleared")

	// Page B: file still exists, DB path preserved for retry
	_, err = os.Stat(pageBPath)
	assert.NoError(t, err, "page B file should still exist")
	var updatedB models.FlyerPage
	require.NoError(t, db.First(&updatedB, pageB.ID).Error)
	assert.Equal(t, pageBPath, updatedB.LocalPath, "page B DB path should be preserved for next run")
}

func TestCleanupSoftDeletedItem_FileCleaned(t *testing.T) {
	task, db, dir := newTestTask(t)

	expiredEnd := time.Now().Add(-40 * 24 * time.Hour)
	flyer := models.Flyer{ShopName: "testshop", EndDate: expiredEnd}
	require.NoError(t, db.Create(&flyer).Error)

	itemPath := createFile(t, dir, "soft_deleted.png")
	item := models.FlyerItem{FlyerID: flyer.ID, LocalPhotoPath: itemPath, Name: "Ghost"}
	require.NoError(t, db.Create(&item).Error)

	// Soft-delete the item
	require.NoError(t, db.Delete(&item).Error)

	task.cleanupExpiredFlyerImages()

	// File must be deleted even though the item is soft-deleted
	_, err := os.Stat(itemPath)
	assert.True(t, os.IsNotExist(err), "soft-deleted item file should still be cleaned up")

	// DB path must be cleared on the soft-deleted row
	var updatedItem models.FlyerItem
	require.NoError(t, db.Unscoped().First(&updatedItem, item.ID).Error)
	assert.Empty(t, updatedItem.LocalPhotoPath)
}
