package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

type FileStorageService struct {
	BaseDir string
}

func NewFileStorageService(baseDir string) *FileStorageService {
	return &FileStorageService{BaseDir: baseDir}
}

// SaveReceipt saves a receipt file to families/{familyID}/receipts/YYYY/MM/filename
func (s *FileStorageService) SaveReceipt(familyID uint, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	now := time.Now()
	relDir := filepath.Join("families", fmt.Sprintf("%d", familyID), "receipts", now.Format("2006"), now.Format("01"))
	fullDir := filepath.Join(s.BaseDir, relDir)

	if mkdirErr := os.MkdirAll(fullDir, 0755); mkdirErr != nil {
		return "", mkdirErr
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", now.Format("20060102_150405"), ext)
	fullPath := filepath.Join(fullDir, filename)
	relPath := filepath.Join(relDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return relPath, nil
}

// SaveReceiptText saves plain-text receipt content to families/{familyID}/receipts/YYYY/MM/{timestamp}.txt.
// Returns the relative path (same pattern as SaveReceipt).
func (s *FileStorageService) SaveReceiptText(familyID uint, text string) (string, error) {
	now := time.Now()
	relDir := filepath.Join("families", fmt.Sprintf("%d", familyID), "receipts", now.Format("2006"), now.Format("01"))
	fullDir := filepath.Join(s.BaseDir, relDir)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// Use nanosecond sub-second component to avoid collisions under concurrent uploads
	filename := fmt.Sprintf("%s_%d.txt", now.Format("20060102_150405"), now.UnixNano()%1_000_000_000)
	fullPath := filepath.Join(fullDir, filename)
	relPath := filepath.Join(relDir, filename)

	if err := os.WriteFile(fullPath, []byte(text), 0644); err != nil {
		return "", err
	}

	return relPath, nil
}
