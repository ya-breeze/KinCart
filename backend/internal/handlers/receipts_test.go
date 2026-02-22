package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"kincart/internal/database"
	"kincart/internal/models"

	coremodels "github.com/ya-breeze/kin-core/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockReceiptSvc is a test double for receiptSvc.
type mockReceiptSvc struct {
	createReceiptFunc     func(familyID uint, file *multipart.FileHeader) (*models.Receipt, error)
	createReceiptTextFunc func(familyID uint, text string) (*models.Receipt, error)
	processReceiptFunc    func(ctx context.Context, receiptID uint, listID uint) error
}

func (m *mockReceiptSvc) CreateReceipt(familyID uint, file *multipart.FileHeader) (*models.Receipt, error) {
	if m.createReceiptFunc != nil {
		return m.createReceiptFunc(familyID, file)
	}
	return &models.Receipt{}, nil
}

func (m *mockReceiptSvc) CreateReceiptFromText(familyID uint, text string) (*models.Receipt, error) {
	if m.createReceiptTextFunc != nil {
		return m.createReceiptTextFunc(familyID, text)
	}
	return &models.Receipt{}, nil
}

func (m *mockReceiptSvc) ProcessReceipt(ctx context.Context, receiptID uint, listID uint) error {
	if m.processReceiptFunc != nil {
		return m.processReceiptFunc(ctx, receiptID, listID)
	}
	return nil
}

func setupReceiptTestDB() uint {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	database.DB.AutoMigrate(&models.ShoppingList{}, &models.Item{}, &models.Family{}, &models.Receipt{}, &models.ReceiptItem{})

	family := models.Family{Family: coremodels.Family{Name: "Test Family"}}
	database.DB.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
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
		createReceiptTextFunc: func(familyID uint, text string) (*models.Receipt, error) {
			return &models.Receipt{}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uint, listID uint) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Lidl\nTotal: 10.00\nMilk 2.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), strings.NewReader(body))
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
		processReceiptFunc: func(ctx context.Context, receiptID uint, listID uint) error {
			return nil
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Lidl\nTotal: 5.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUploadReceipt_InvalidJSON(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{}
	r := newReceiptRouter(svc)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), strings.NewReader("{invalid json"))
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
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), strings.NewReader(body))
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

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

// --- Multipart file tests ---

func buildMultipartRequest(t *testing.T, listID uint, filename string, content []byte) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("receipt", filename)
	assert.NoError(t, err)
	_, err = part.Write(content)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), buf)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadReceipt_MultipartTxtSuccess(t *testing.T) {
	listID := setupReceiptTestDB()

	var capturedText string
	svc := &mockReceiptSvc{
		createReceiptTextFunc: func(familyID uint, text string) (*models.Receipt, error) {
			capturedText = text
			return &models.Receipt{}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uint, listID uint) error {
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
		createReceiptFunc: func(familyID uint, file *multipart.FileHeader) (*models.Receipt, error) {
			imageCalled = true
			return &models.Receipt{}, nil
		},
		processReceiptFunc: func(ctx context.Context, receiptID uint, listID uint) error {
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
	req, _ := http.NewRequest(http.MethodPost, "/lists/99999/receipts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUploadReceipt_GeminiUnavailableQueues(t *testing.T) {
	listID := setupReceiptTestDB()

	svc := &mockReceiptSvc{
		processReceiptFunc: func(ctx context.Context, receiptID uint, listID uint) error {
			return fmt.Errorf("gemini client not available")
		},
	}

	r := newReceiptRouter(svc)

	body := `{"receipt_text": "Store: Test\nTotal: 5.00"}`
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/lists/%d/receipts", listID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "queued", resp["status"])
}
