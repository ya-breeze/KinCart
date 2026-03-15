# Receipt Viewer Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a receipt viewer drawer to ListDetail that lets users view receipt images/text and parsed items, and download the original file.

**Architecture:** Split the existing receipt button into upload (existing) and view (new). A new `ReceiptViewerModal` component fetches receipt files via a new protected backend endpoint that serves files with family-scoped auth. The Receipt model gains a `Shop` association for name display.

**Tech Stack:** Go/Gin, GORM/SQLite, React 19/Vite, vanilla CSS, Vitest/RTL, Go testing/testify

---

## Chunk 1: Backend

### Task 1: Add Shop association to Receipt model

**Files:**
- Modify: `backend/internal/models/models.go`

- [ ] **Step 1.1: Add `Shop *Shop` field to Receipt struct**

In `backend/internal/models/models.go`, the Receipt struct currently ends at `Items []ReceiptItem`. Add the Shop association field after `ShopID`:

```go
type Receipt struct {
    coremodels.TenantModel
    ListID    *uint         `json:"list_id"`
    ShopID    *uint         `json:"shop_id"`
    Shop      *Shop         `gorm:"foreignKey:ShopID" json:"shop"`
    Date      time.Time     `json:"date"`
    Total     float64       `json:"total"`
    ImagePath string        `json:"image_path"`
    Status    string        `gorm:"default:'new'" json:"status"`
    Items     []ReceiptItem `gorm:"foreignKey:ReceiptID" json:"items"`
}
```

- [ ] **Step 1.2: Verify the project builds**

```bash
cd /Users/ek/work/KinCart/backend && go build ./...
```
Expected: no errors.

---

### Task 2: Update GetList preloading

**Files:**
- Modify: `backend/internal/handlers/lists.go`

- [ ] **Step 2.1: Add receipt sub-preloads to GetList**

In `backend/internal/handlers/lists.go`, `GetList` (line ~34) currently has:
```go
if err := database.DB.Preload("Items").Preload("Receipts").Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
```

Change to:
```go
if err := database.DB.Preload("Items").Preload("Receipts").Preload("Receipts.Items").Preload("Receipts.Shop").Where("id = ? AND family_id = ?", listID, familyID).First(&list).Error; err != nil {
```

`GetLists` does NOT need this change — it is a summary endpoint (receipt count only) and adding shop preloads there would regress performance on the Dashboard.

- [ ] **Step 2.2: Verify build**

```bash
cd /Users/ek/work/KinCart/backend && go build ./...
```
Expected: no errors.

---

### Task 3: Add GetReceiptFile handler

**Files:**
- Modify: `backend/internal/handlers/receipts.go`
- Modify: `backend/internal/handlers/receipts_test.go`

- [ ] **Step 3.1: Write the failing tests first**

Append to `backend/internal/handlers/receipts_test.go`:

```go
// --- GetReceiptFile tests ---

func setupReceiptFileTestDB(t *testing.T) (listID uint, familyID uint) {
	t.Helper()
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("failed to open test db")
	}
	database.DB.AutoMigrate(
		&models.ShoppingList{}, &models.Item{}, &models.Family{},
		&models.Receipt{}, &models.ReceiptItem{}, &models.Shop{},
	)

	family := models.Family{Family: coremodels.Family{Name: "File Test Family"}}
	database.DB.Create(&family)

	list := models.ShoppingList{
		TenantModel: coremodels.TenantModel{FamilyID: family.ID},
		Title:       "File Test List",
	}
	database.DB.Create(&list)

	return list.ID, family.ID
}

func newReceiptFileRouterWithFamily(dataPath string, familyID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/receipts/:id/file", func(c *gin.Context) {
		c.Set("family_id", familyID)
		getReceiptFileWith(c, dataPath)
	})
	return r
}

func TestGetReceiptFile_Success(t *testing.T) {
	_, familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	imagePath := fmt.Sprintf("families/%d/receipts/2026/03/receipt.jpg", familyID)
	fullPath := filepath.Join(tmpDir, imagePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("fake image data"), 0644); err != nil {
		t.Fatal(err)
	}

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: familyID},
		ImagePath:   imagePath,
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%d/file", receipt.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fake image data", w.Body.String())
}

func TestGetReceiptFile_NotFound(t *testing.T) {
	_, familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/receipts/99999/file", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReceiptFile_WrongFamily(t *testing.T) {
	_, familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	imagePath := fmt.Sprintf("families/%d/receipts/2026/03/receipt.jpg", familyID)
	fullPath := filepath.Join(tmpDir, imagePath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte("data"), 0644)

	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: familyID},
		ImagePath:   imagePath,
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	// Request as a different family
	r := newReceiptFileRouterWithFamily(tmpDir, familyID+99)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%d/file", receipt.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReceiptFile_MissingFile(t *testing.T) {
	_, familyID := setupReceiptFileTestDB(t)

	tmpDir := t.TempDir()
	receipt := models.Receipt{
		TenantModel: coremodels.TenantModel{FamilyID: familyID},
		ImagePath:   "families/1/receipts/2026/03/missing.jpg",
		Status:      "parsed",
	}
	database.DB.Create(&receipt)

	r := newReceiptFileRouterWithFamily(tmpDir, familyID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/receipts/%d/file", receipt.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

You'll also need these imports added to the test file if not already present:
```go
"os"
"path/filepath"
```

- [ ] **Step 3.2: Run tests to verify they fail**

```bash
cd /Users/ek/work/KinCart/backend && go test ./internal/handlers -run TestGetReceiptFile -v
```
Expected: FAIL — `getReceiptFileWith` undefined.

- [ ] **Step 3.3: Implement GetReceiptFile handler**

Append to `backend/internal/handlers/receipts.go`:

```go
// GetReceiptFile serves the raw receipt file (image, PDF, or text).
// GET /api/receipts/:id/file
func GetReceiptFile(c *gin.Context) {
	dataPath := os.Getenv("KINCART_DATA_PATH")
	if dataPath == "" {
		dataPath = "./kincart-data"
	}
	getReceiptFileWith(c, dataPath)
}

// getReceiptFileWith is the testable core of GetReceiptFile.
func getReceiptFileWith(c *gin.Context, dataPath string) {
	familyID := c.MustGet("family_id").(uint)
	receiptIDStr := c.Param("id")
	receiptID, err := strconv.ParseUint(receiptIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt ID"})
		return
	}

	var receipt models.Receipt
	if dbErr := database.DB.Preload("Shop").
		Where("id = ? AND family_id = ?", receiptID, familyID).
		First(&receipt).Error; dbErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt not found"})
		return
	}

	rawDataPath, _ := filepath.Abs(dataPath)
	rawFilePath, _ := filepath.Abs(filepath.Join(dataPath, receipt.ImagePath))
	// EvalSymlinks resolves macOS /var → /private/var etc., keeping tests green on all platforms.
	// Fall back to the raw absolute path if the path doesn't exist yet (e.g. missing file) —
	// the os.Stat check below handles the not-found case.
	absDataPath, err := filepath.EvalSymlinks(rawDataPath)
	if err != nil {
		absDataPath = rawDataPath
	}
	absFilePath, err := filepath.EvalSymlinks(rawFilePath)
	if err != nil {
		absFilePath = rawFilePath
	}

	// Path traversal guard (uses "/" intentionally — project runs on Linux/Docker)
	if !strings.HasPrefix(absFilePath, absDataPath+"/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	if _, statErr := os.Stat(absFilePath); os.IsNotExist(statErr) {
		slog.Error("Receipt file missing on disk", "path", absFilePath, "receipt_id", receipt.ID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt file not found"})
		return
	}

	// Derive a friendly download filename
	ext := strings.TrimPrefix(filepath.Ext(receipt.ImagePath), ".")
	if ext == "" {
		ext = "bin"
	}
	date := receipt.Date.Format("2006-01-02")
	var filename string
	if receipt.Shop != nil {
		shopSlug := strings.ToLower(strings.ReplaceAll(receipt.Shop.Name, " ", "-"))
		filename = "receipt-" + date + "-" + shopSlug + "." + ext
	} else {
		filename = fmt.Sprintf("receipt-%d.%s", receipt.ID, ext)
	}

	c.FileAttachment(absFilePath, filename)
}
```

Add `"fmt"` to the imports in `receipts.go` if not already present (it isn't — add it).

- [ ] **Step 3.4: Run tests to verify they pass**

```bash
cd /Users/ek/work/KinCart/backend && go test ./internal/handlers -run TestGetReceiptFile -v
```
Expected: all 4 tests PASS.

---

### Task 4: Register route

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 4.1: Add route to protected group**

In `backend/cmd/server/main.go`, find the protected group where `POST /lists/:id/receipts` is registered. Add the new GET route immediately after:

```go
protected.POST("/lists/:id/receipts", handlers.UploadReceipt)
protected.GET("/receipts/:id/file", handlers.GetReceiptFile)
```

- [ ] **Step 4.2: Run full backend test suite**

```bash
cd /Users/ek/work/KinCart/backend && go test ./...
```
Expected: all tests pass.

- [ ] **Step 4.3: Run make to verify lint and build**

```bash
cd /Users/ek/work/KinCart && make build && make lint-backend
```
Expected: no errors.

---

## Chunk 2: Frontend

### Task 5: Split receipt button in ListDetail

**Files:**
- Modify: `frontend/src/pages/ListDetail.jsx`

- [ ] **Step 5.1: Add isReceiptViewerOpen state**

In `ListDetail.jsx`, find line 29 where `isReceiptModalOpen` is declared:
```js
const [isReceiptModalOpen, setIsReceiptModalOpen] = useState(false);
```
Add the new state immediately after:
```js
const [isReceiptViewerOpen, setIsReceiptViewerOpen] = useState(false);
```

- [ ] **Step 5.2: Split the receipt button (lines 402–436)**

Replace the single button:
```jsx
<button
    onClick={() => setIsReceiptModalOpen(true)}
    className="card"
    style={{
        padding: '0.4rem',
        borderRadius: '50%',
        color: 'var(--primary)',
        border: '1px solid var(--border)',
        marginLeft: 'auto',
        position: 'relative',
        minHeight: 'unset'
    }}
    title="Upload Receipt"
>
    <Receipt size={18} />
    {list.receipts?.length > 0 && (
        <span style={{
            position: 'absolute',
            top: '-3px',
            right: '-3px',
            background: 'var(--primary)',
            color: 'white',
            fontSize: '0.55rem',
            fontWeight: 'bold',
            width: '14px',
            height: '14px',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
        }}>
            {list.receipts.length}
        </span>
    )}
</button>
```

With two separate controls:
```jsx
<button
    onClick={() => setIsReceiptModalOpen(true)}
    className="card"
    style={{
        padding: '0.4rem',
        borderRadius: '50%',
        color: 'var(--primary)',
        border: '1px solid var(--border)',
        marginLeft: 'auto',
        position: 'relative',
        minHeight: 'unset'
    }}
    title="Upload Receipt"
>
    <Receipt size={18} />
</button>
{list.receipts?.length > 0 && (
    <button
        data-testid="view-receipts-btn"
        onClick={() => setIsReceiptViewerOpen(true)}
        className="card"
        style={{
            padding: '0.4rem',
            borderRadius: '50%',
            color: 'var(--primary)',
            border: '1px solid var(--border)',
            position: 'relative',
            minHeight: 'unset'
        }}
        title="View Receipts"
    >
        <Receipt size={18} />
        <span style={{
            position: 'absolute',
            top: '-3px',
            right: '-3px',
            background: 'var(--primary)',
            color: 'white',
            fontSize: '0.55rem',
            fontWeight: 'bold',
            width: '14px',
            height: '14px',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
        }}>
            {list.receipts.length}
        </span>
    </button>
)}
```

- [ ] **Step 5.3: Verify frontend builds**

```bash
cd /Users/ek/work/KinCart/frontend && npm run build
```
Expected: build succeeds. (The import and render of `ReceiptViewerModal` are added in Task 5b, after the component is created in Task 6.)

---

### Task 6: Create ReceiptViewerModal component

**Files:**
- Create: `frontend/src/components/ReceiptViewerModal.jsx`
- Create: `frontend/src/components/ReceiptViewerModal.css`

- [ ] **Step 6.1: Create the CSS file**

Create `frontend/src/components/ReceiptViewerModal.css`:

```css
.receipt-viewer-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1000;
    display: flex;
    align-items: flex-end;
    justify-content: center;
}

.receipt-viewer-modal {
    background: var(--card-bg, #1e1e2e);
    border-radius: 12px 12px 0 0;
    width: 100%;
    max-height: 90vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
}

.receipt-viewer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem;
    border-bottom: 1px solid var(--border, #333);
    position: sticky;
    top: 0;
    background: var(--card-bg, #1e1e2e);
    z-index: 1;
}

.receipt-viewer-header h3 {
    margin: 0;
    font-size: 1rem;
}

.receipt-viewer-header button {
    background: none;
    border: none;
    color: var(--text, #fff);
    cursor: pointer;
    font-size: 0.9rem;
    padding: 0.25rem 0.5rem;
}

.receipt-download-btn {
    background: var(--primary, #6c6af6) !important;
    border-radius: 6px !important;
    padding: 0.3rem 0.75rem !important;
}

/* List view */
.receipt-list {
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.receipt-list-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
    border-radius: 8px;
    background: var(--bg, #13131f);
    cursor: pointer;
    border: 1px solid var(--border, #333);
}

.receipt-list-item:hover {
    border-color: var(--primary, #6c6af6);
}

.receipt-list-icon {
    font-size: 1.5rem;
    flex-shrink: 0;
}

.receipt-list-info {
    flex: 1;
    font-size: 0.85rem;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
}

.receipt-list-info .receipt-shop {
    color: var(--text-muted, #888);
    font-size: 0.8rem;
}

.receipt-list-info .receipt-total {
    color: var(--primary, #6c6af6);
    font-weight: 600;
}

.receipt-status-badge {
    font-size: 0.7rem;
    padding: 0.15rem 0.4rem;
    border-radius: 4px;
    font-weight: 600;
    flex-shrink: 0;
}

.receipt-status-parsed { background: #1a3a1a; color: #4ade80; }
.receipt-status-new    { background: #2a2a1a; color: #facc15; }
.receipt-status-error  { background: #3a1a1a; color: #f87171; }

/* Detail view */
.receipt-detail {
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.receipt-detail-file {
    background: var(--bg, #13131f);
    border-radius: 8px;
    overflow: hidden;
    min-height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.receipt-detail-file img {
    width: 100%;
    height: auto;
    display: block;
}

.receipt-text {
    padding: 1rem;
    font-size: 0.8rem;
    white-space: pre-wrap;
    overflow-y: auto;
    max-height: 300px;
    margin: 0;
    color: var(--text, #fff);
}

.receipt-loading {
    color: var(--text-muted, #888);
    font-size: 0.9rem;
    padding: 2rem;
}

.receipt-detail-meta {
    display: flex;
    gap: 0.75rem;
    font-size: 0.85rem;
    color: var(--text-muted, #888);
    margin-bottom: 0.5rem;
}

.receipt-items-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.85rem;
}

.receipt-items-table th {
    text-align: left;
    padding: 0.4rem 0.5rem;
    border-bottom: 1px solid var(--border, #333);
    color: var(--text-muted, #888);
    font-weight: 500;
    font-size: 0.75rem;
    text-transform: uppercase;
}

.receipt-items-table td {
    padding: 0.4rem 0.5rem;
    border-bottom: 1px solid var(--border, #222);
}

.receipt-items-table td:last-child {
    text-align: right;
    color: var(--primary, #6c6af6);
}

.receipt-total-row {
    display: flex;
    justify-content: space-between;
    font-weight: 600;
    padding: 0.5rem;
    margin-top: 0.25rem;
    border-top: 1px solid var(--border, #333);
}

.receipt-parse-pending,
.receipt-parse-error {
    padding: 1rem;
    text-align: center;
    color: var(--text-muted, #888);
    font-size: 0.9rem;
}

/* Desktop: drawer on the right */
@media (min-width: 640px) {
    .receipt-viewer-overlay {
        align-items: stretch;
        justify-content: flex-end;
    }

    .receipt-viewer-modal {
        border-radius: 0;
        width: 480px;
        max-height: 100vh;
    }

    .receipt-detail {
        flex-direction: row;
        align-items: flex-start;
    }

    .receipt-detail-file {
        flex: 1;
        min-height: 200px;
    }

    .receipt-detail-items {
        flex: 1;
        overflow-y: auto;
    }
}
```

- [ ] **Step 6.2: Create ReceiptViewerModal.jsx**

Create `frontend/src/components/ReceiptViewerModal.jsx`:

```jsx
import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { API_BASE_URL } from '../config';
import './ReceiptViewerModal.css';

export default function ReceiptViewerModal({ receipts, isOpen, onClose }) {
    const { token } = useAuth();
    const [selectedReceiptId, setSelectedReceiptId] = useState(null);
    const [blobUrl, setBlobUrl] = useState(null);
    const [textContent, setTextContent] = useState(null);
    const [isLoadingFile, setIsLoadingFile] = useState(false);

    const selectedReceipt = receipts?.find(r => r.id === selectedReceiptId) || null;

    // Load file when a receipt is selected
    useEffect(() => {
        if (!selectedReceiptId || !selectedReceipt) return;

        let cancelled = false;
        let createdBlobUrl = null;

        setIsLoadingFile(true);
        setBlobUrl(null);
        setTextContent(null);

        const isTextReceipt = selectedReceipt.image_path?.endsWith('.txt');

        fetch(`${API_BASE_URL}/api/receipts/${selectedReceiptId}/file`, {
            headers: { Authorization: `Bearer ${token}` },
        })
            .then(res => {
                if (!res.ok) throw new Error('Failed to load receipt file');
                return isTextReceipt ? res.text() : res.blob();
            })
            .then(data => {
                if (cancelled) return;
                if (isTextReceipt) {
                    setTextContent(data);
                } else {
                    createdBlobUrl = URL.createObjectURL(data);
                    setBlobUrl(createdBlobUrl);
                }
            })
            .catch(err => console.error('Receipt file load error:', err))
            .finally(() => {
                if (!cancelled) setIsLoadingFile(false);
            });

        return () => {
            cancelled = true;
            if (createdBlobUrl) URL.revokeObjectURL(createdBlobUrl);
        };
    }, [selectedReceiptId]); // eslint-disable-line react-hooks/exhaustive-deps

    // Revoke blob and reset state when modal closes
    useEffect(() => {
        if (!isOpen) {
            if (blobUrl) URL.revokeObjectURL(blobUrl);
            setBlobUrl(null);
            setTextContent(null);
            setSelectedReceiptId(null);
        }
    }, [isOpen]); // eslint-disable-line react-hooks/exhaustive-deps

    const handleBack = () => {
        if (blobUrl) URL.revokeObjectURL(blobUrl);
        setBlobUrl(null);
        setTextContent(null);
        setSelectedReceiptId(null);
    };

    const handleDownload = async () => {
        if (!selectedReceipt) return;
        const ext = selectedReceipt.image_path?.split('.').pop() || 'bin';
        const date = selectedReceipt.date
            ? new Date(selectedReceipt.date).toISOString().split('T')[0]
            : 'unknown';
        const shop = selectedReceipt.shop?.name?.toLowerCase().replace(/\s+/g, '-') || null;
        const filename = shop
            ? `receipt-${date}-${shop}.${ext}`
            : `receipt-${selectedReceipt.id}.${ext}`;

        try {
            const res = await fetch(`${API_BASE_URL}/api/receipts/${selectedReceipt.id}/file`, {
                headers: { Authorization: `Bearer ${token}` },
            });
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            a.click();
            URL.revokeObjectURL(url);
        } catch (err) {
            console.error('Download error:', err);
        }
    };

    const statusLabel = status => {
        if (status === 'parsed') return 'Parsed';
        if (status === 'error') return 'Error';
        return 'Pending';
    };

    if (!isOpen) return null;

    return (
        <div className="receipt-viewer-overlay" data-testid="receipt-viewer-overlay" onClick={onClose}>
            <div className="receipt-viewer-modal" onClick={e => e.stopPropagation()}>
                {selectedReceiptId ? (
                    <>
                        <div className="receipt-viewer-header">
                            <button data-testid="receipt-viewer-back" onClick={handleBack}>
                                ← Back
                            </button>
                            <button
                                data-testid="receipt-viewer-download"
                                className="receipt-download-btn"
                                onClick={handleDownload}
                            >
                                ⬇ Download
                            </button>
                        </div>
                        <div className="receipt-detail">
                            <div className="receipt-detail-file">
                                {isLoadingFile ? (
                                    <div className="receipt-loading">Loading…</div>
                                ) : selectedReceipt?.image_path?.endsWith('.txt') ? (
                                    <pre className="receipt-text">{textContent}</pre>
                                ) : selectedReceipt?.image_path?.endsWith('.pdf') ? (
                                    blobUrl ? (
                                        <img src={blobUrl} alt="Receipt" />
                                    ) : (
                                        <div className="receipt-loading">Download to view PDF</div>
                                    )
                                ) : (
                                    blobUrl && <img src={blobUrl} alt="Receipt" />
                                )}
                            </div>
                            <div className="receipt-detail-items">
                                {selectedReceipt?.status === 'parsed' ? (
                                    <>
                                        <div className="receipt-detail-meta">
                                            {selectedReceipt.shop?.name && (
                                                <span>{selectedReceipt.shop.name}</span>
                                            )}
                                            {selectedReceipt.date && (
                                                <span>
                                                    {new Date(selectedReceipt.date).toLocaleDateString()}
                                                </span>
                                            )}
                                        </div>
                                        <table className="receipt-items-table">
                                            <thead>
                                                <tr>
                                                    <th>Item</th>
                                                    <th>Qty</th>
                                                    <th>Price</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {(selectedReceipt.items ?? []).map(item => (
                                                    <tr key={item.id}>
                                                        <td>{item.name}</td>
                                                        <td>
                                                            {item.quantity} {item.unit}
                                                        </td>
                                                        <td>${item.total_price?.toFixed(2)}</td>
                                                    </tr>
                                                ))}
                                                {(selectedReceipt.items ?? []).length === 0 && (
                                                    <tr>
                                                        <td colSpan={3} style={{ textAlign: 'center', color: 'var(--text-muted)' }}>
                                                            No items available
                                                        </td>
                                                    </tr>
                                                )}
                                            </tbody>
                                        </table>
                                        <div className="receipt-total-row">
                                            <span>Total</span>
                                            <span>${selectedReceipt.total?.toFixed(2)}</span>
                                        </div>
                                    </>
                                ) : selectedReceipt?.status === 'new' ? (
                                    <div className="receipt-parse-pending">⏳ Still processing…</div>
                                ) : (
                                    <div className="receipt-parse-error">
                                        Could not parse this receipt
                                    </div>
                                )}
                            </div>
                        </div>
                    </>
                ) : (
                    <>
                        <div className="receipt-viewer-header">
                            <h3>Receipts ({receipts?.length ?? 0})</h3>
                            <button onClick={onClose}>✕</button>
                        </div>
                        <div className="receipt-list">
                            {(receipts ?? []).map(receipt => (
                                <div
                                    key={receipt.id}
                                    className="receipt-list-item"
                                    onClick={() => setSelectedReceiptId(receipt.id)}
                                    data-testid={`receipt-list-item-${receipt.id}`}
                                >
                                    <div className="receipt-list-icon">
                                        {receipt.image_path?.endsWith('.txt') ? '📝' : '🖼'}
                                    </div>
                                    <div className="receipt-list-info">
                                        <div>
                                            {receipt.date
                                                ? new Date(receipt.date).toLocaleDateString()
                                                : 'Unknown date'}
                                        </div>
                                        {receipt.shop?.name && (
                                            <div className="receipt-shop">{receipt.shop.name}</div>
                                        )}
                                        {receipt.status === 'parsed' && (
                                            <div className="receipt-total">
                                                ${receipt.total?.toFixed(2)}
                                            </div>
                                        )}
                                    </div>
                                    <div
                                        className={`receipt-status-badge receipt-status-${receipt.status}`}
                                    >
                                        {statusLabel(receipt.status)}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
```

- [ ] **Step 6.3: Verify frontend builds**

```bash
cd /Users/ek/work/KinCart/frontend && npm run build
```
Expected: build succeeds with no errors.

---

### Task 5b: Wire ReceiptViewerModal into ListDetail

**Files:**
- Modify: `frontend/src/pages/ListDetail.jsx`

Do this after Task 6 (component exists).

- [ ] **Step 5b.1: Add import**

Near the top of `ListDetail.jsx`, after the existing `ReceiptUploadModal` import (line 7):
```js
import ReceiptViewerModal from '../components/ReceiptViewerModal';
```

- [ ] **Step 5b.2: Render ReceiptViewerModal**

In `ListDetail.jsx`, after the `<ReceiptUploadModal ... />` block (around line 955), add:
```jsx
<ReceiptViewerModal
    receipts={list.receipts || []}
    isOpen={isReceiptViewerOpen}
    onClose={() => setIsReceiptViewerOpen(false)}
/>
```

- [ ] **Step 5b.3: Verify frontend builds**

```bash
cd /Users/ek/work/KinCart/frontend && npm run build
```
Expected: build succeeds with no errors.

---

### Task 7: Frontend tests for ReceiptViewerModal

**Files:**
- Create: `frontend/src/components/ReceiptViewerModal.test.jsx`

- [ ] **Step 7.1: Write tests**

Create `frontend/src/components/ReceiptViewerModal.test.jsx`:

```jsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ReceiptViewerModal from './ReceiptViewerModal';

// Mock useAuth
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({ token: 'test-token' }),
}));

// Mock config
vi.mock('../config', () => ({
    API_BASE_URL: 'http://localhost:8080',
}));

const mockReceipts = [
    {
        id: 1,
        status: 'parsed',
        date: '2026-03-10T00:00:00Z',
        total: 42.5,
        image_path: 'families/1/receipts/2026/03/receipt.jpg',
        shop: { id: 1, name: 'Costco' },
        items: [
            { id: 1, name: 'Milk', quantity: 2, unit: 'L', price: 2.99, total_price: 5.98 },
        ],
    },
    {
        id: 2,
        status: 'new',
        date: '2026-03-12T00:00:00Z',
        total: 0,
        image_path: 'families/1/receipts/2026/03/receipt.txt',
        shop: null,
        items: [],
    },
    {
        id: 3,
        status: 'error',
        date: '2026-03-13T00:00:00Z',
        total: 0,
        image_path: 'families/1/receipts/2026/03/receipt2.jpg',
        shop: null,
        items: [],
    },
];

describe('ReceiptViewerModal', () => {
    let fetchMock;

    beforeEach(() => {
        fetchMock = vi.fn();
        global.fetch = fetchMock;
        global.URL.createObjectURL = vi.fn(() => 'blob:mock-url');
        global.URL.revokeObjectURL = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('renders nothing when closed', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={false} onClose={vi.fn()} />
        );
        expect(screen.queryByText('Receipts (3)')).toBeNull();
    });

    it('renders receipt list when open', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Receipts (3)')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-1')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-2')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-3')).toBeTruthy();
    });

    it('shows correct status badges', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Parsed')).toBeTruthy();
        expect(screen.getByText('Pending')).toBeTruthy();
        expect(screen.getByText('Error')).toBeTruthy();
    });

    it('shows shop name for parsed receipts', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Costco')).toBeTruthy();
    });

    it('navigates to detail view on receipt click', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );

        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(screen.getByTestId('receipt-viewer-back')).toBeTruthy();
            expect(screen.getByTestId('receipt-viewer-download')).toBeTruthy();
        });
    });

    it('shows parsed items in detail view', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(screen.getByText('Milk')).toBeTruthy();
            expect(screen.getByText('$5.98')).toBeTruthy();
        });
    });

    it('shows "still processing" for new receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-2'));

        await waitFor(() => {
            expect(screen.getByText(/Still processing/)).toBeTruthy();
        });
    });

    it('shows error message for failed receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-3'));

        await waitFor(() => {
            expect(screen.getByText(/Could not parse/)).toBeTruthy();
        });
    });

    it('fetches text content for .txt receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            text: () => Promise.resolve('Store: Lidl\nTotal: 10.00'),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-2'));

        await waitFor(() => {
            expect(screen.getByText(/Store: Lidl/)).toBeTruthy();
        });
    });

    it('returns to list view on back button click', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => screen.getByTestId('receipt-viewer-back'));
        fireEvent.click(screen.getByTestId('receipt-viewer-back'));

        expect(screen.getByText('Receipts (3)')).toBeTruthy();
    });

    it('calls onClose when overlay is clicked', () => {
        const onClose = vi.fn();
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={onClose} />
        );
        fireEvent.click(screen.getByTestId('receipt-viewer-overlay'));
        expect(onClose).toHaveBeenCalled();
    });

    it('sends auth header when fetching receipt file', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(fetchMock).toHaveBeenCalledWith(
                'http://localhost:8080/api/receipts/1/file',
                expect.objectContaining({
                    headers: expect.objectContaining({ Authorization: 'Bearer test-token' }),
                })
            );
        });
    });

    it('shows empty items state gracefully', async () => {
        // Receipt with status parsed but no items
        const noItemsReceipts = [
            {
                id: 10,
                status: 'parsed',
                date: '2026-03-10T00:00:00Z',
                total: 5.0,
                image_path: 'families/1/receipts/2026/03/r.jpg',
                shop: null,
                items: [],
            },
        ];

        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={noItemsReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-10'));

        await waitFor(() => {
            expect(screen.getByText('No items available')).toBeTruthy();
        });
    });
});
```

- [ ] **Step 7.2: Run frontend tests**

```bash
cd /Users/ek/work/KinCart/frontend && npm test -- ReceiptViewerModal.test.jsx
```
Expected: all tests pass.

- [ ] **Step 7.3: Run full make**

```bash
cd /Users/ek/work/KinCart && make
```
Expected: all builds, tests, and linters pass.

---

## Final verification checklist

- [ ] `make` passes with zero errors
- [ ] Receipt count badge in ListDetail is now a separate button from the upload button
- [ ] Clicking the badge opens the viewer drawer showing list of receipts
- [ ] Clicking a receipt shows image on left, parsed items on right
- [ ] Text receipts show content in `<pre>` block
- [ ] "Still processing…" shown for status=new
- [ ] "Could not parse" shown for status=error
- [ ] Download button saves the file with a sensible filename
- [ ] Back button returns to the list view
