package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"kincart/internal/database"
	"kincart/internal/models"
	"kincart/internal/services"

	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockReceiptSvc is a test double for receiptSvc.
type mockReceiptSvc struct {
	createReceiptFunc     func(familyID uuid.UUID, file *multipart.FileHeader) (*models.Receipt, error)
	createReceiptTextFunc func(familyID uuid.UUID, text string) (*models.Receipt, error)
	processReceiptFunc    func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error
}

func (m *mockReceiptSvc) CreateReceipt(familyID uuid.UUID, file *multipart.FileHeader) (*models.Receipt, error) {
	if m.createReceiptFunc != nil {
		return m.createReceiptFunc(familyID, file)
	}
	return &models.Receipt{TenantModel: coremodels.TenantModel{ID: uuid.New()}}, nil
}

func (m *mockReceiptSvc) CreateReceiptFromText(familyID uuid.UUID, text string) (*models.Receipt, error) {
	if m.createReceiptTextFunc != nil {
		return m.createReceiptTextFunc(familyID, text)
	}
	return &models.Receipt{TenantModel: coremodels.TenantModel{ID: uuid.New()}}, nil
}

func (m *mockReceiptSvc) ProcessReceipt(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
	if m.processReceiptFunc != nil {
		return m.processReceiptFunc(ctx, receiptID, listID)
	}
	return nil
}

func setupReceiptTestDB() uuid.UUID {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	database.DB.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Receipt{}, &models.ReceiptItem{})

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "Test Family"}}
	database.DB.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: family.ID},
		Title:       "Test List",
	}
	database.DB.Create(&list)

	return list.ID
}

func newReceiptRouter(svc receiptSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/lists/:id/receipts", func(c *gin.Context) {
		uploadReceiptWith(c, svc)
	})
	return r
}

// --- JSON paste mode tests ---

func TestUploadReceipt_JSONSuccess(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{
		createReceiptTextFunc: func(familyID uuid.UUID, text string) (*models.Receipt, error) {
			return &models.Receipt{TenantModel: coremodels.TenantModel{ID: uuid.New()}}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Lidl\nTotal: 10.00\nMilk 2.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "parsed", resp["status"])
}

func TestUploadReceipt_JSONWithCharset(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{
		processReceiptFunc: func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Lidl\nTotal: 5.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUploadReceipt_InvalidJSON(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUploadReceipt_EmptyReceiptText(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	for _, body := range []string{
		`{"receipt_text": ""}`,
		`{"receipt_text": "   "}`,
		`{"receipt_text": "\n\t\n"}`,
	} {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "expected 400 for body: %s", body)
	}
}

func TestUploadReceipt_OversizedPastedText(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	bigText := strings.Repeat("A", maxReceiptTextBytes+1)
	body, _ := json.Marshal(map[string]string{"receipt_text": bigText})

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

// --- Multipart file tests ---

func buildMultipartRequest(t *testing.T, listID uuid.UUID, filename string, content []byte) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("receipt", filename)
	assert.NoError(t, err)
	_, err = part.Write(content)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), buf)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadReceipt_MultipartTxtSuccess(t *testing.T) {
	listID := setupReceiptTestDB()

	var capturedText string
	svc := &mockReceiptSvc{
		createReceiptTextFunc: func(familyID uuid.UUID, text string) (*models.Receipt, error) {
			capturedText = text
			return &models.Receipt{TenantModel: coremodels.TenantModel{ID: uuid.New()}}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)
	content := []byte("Store: Lidl\nMilk 1,99\nBread 2,49")
	req := buildMultipartRequest(t, listID, "receipt.txt", content)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, string(content), capturedText)
}

func TestUploadReceipt_MultipartTxtInvalidUTF8(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	// Build invalid UTF-8 bytes
	invalidUTF8 := []byte("Store: Test\n\xff\xfe invalid bytes")
	assert.False(t, utf8.Valid(invalidUTF8))

	req := buildMultipartRequest(t, listID, "receipt.txt", invalidUTF8)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "UTF-8")
}

func TestUploadReceipt_MultipartImageUnchanged(t *testing.T) {
	listID := setupReceiptTestDB()

	var imageCalled bool
	svc := &mockReceiptSvc{
		createReceiptFunc: func(familyID uuid.UUID, file *multipart.FileHeader) (*models.Receipt, error) {
			imageCalled = true
			return &models.Receipt{TenantModel: coremodels.TenantModel{ID: uuid.New()}}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)
	// Fake JPEG content (starts with FF D8 FF — JPEG magic bytes)
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}
	req := buildMultipartRequest(t, listID, "receipt.jpg", fakeJPEG)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, imageCalled, "expected CreateReceipt (image path) to be called")
}

func TestUploadReceipt_ListNotFound(t *testing.T) {
	setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	body := `{"receipt_text": "some text"}`
	nonExistentID := uuid.New()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", nonExistentID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUploadReceipt_GeminiUnavailableQueues(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{
		processReceiptFunc: func(ctx context.Context, receiptID uuid.UUID, listID uuid.UUID) error {
			return services.ErrGeminiUnavailable
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Test\nTotal: 5.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%s/receipts", listID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "queued", resp["status"])
}

// --- GetReceiptFile tests ---

func setupReceiptFileTestDB(t *testing.T) uuid.UUID {
	t.Helper()
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("failed to open test db")
	}
	database.DB.AutoMigrate(
		&models.Item{}, &models.Family{},
		&models.Receipt{}, &models.ReceiptItem{}, &models.Shop{},
	)

	family := models.Family{Family: coremodels.Family{ID: uuid.New(), Name: "File Test Family"}}
	database.DB.Create(&family)

	return family.ID
}

func newReceiptFileRouterWithFamily(dataPath string, familyID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/receipts/:id/file", func(c *gin.Context) {
		c.Set("family_id", familyID)
		getReceiptFileWith(c, dataPath)
	})
	return r
}

func TestGetReceiptFile_Success(t *testing.T) {
	familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	imagePath := fmt.Sprintf("families/%s/receipts/2026/03/receipt.jpg", familyID.String())
	fullPath := filepath.Join(tmpDir, imagePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("fake image data"), 0644); err != nil {
		t.Fatal(err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ImagePath:   imagePath,
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%s/file", receipt.ID.String()), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fake image data", w.Body.String())
}

func TestGetReceiptFile_NotFound(t *testing.T) {
	familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%s/file", uuid.New().String()), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReceiptFile_WrongFamily(t *testing.T) {
	familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	imagePath := fmt.Sprintf("families/%s/receipts/2026/03/receipt.jpg", familyID.String())
	fullPath := filepath.Join(tmpDir, imagePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ImagePath:   imagePath,
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	// Request as a different family
	otherFamilyID := uuid.New()
	r := newReceiptFileRouterWithFamily(tmpDir, otherFamilyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%s/file", receipt.ID.String()), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReceiptFile_MissingFile(t *testing.T) {
	familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{ID: uuid.New(), FamilyID: familyID},
		ImagePath:   "families/test/receipts/2026/03/missing.jpg",
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%s/file", receipt.ID.String()), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
