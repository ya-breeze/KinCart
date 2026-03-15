# Receipt Viewer — Design Spec
**Date:** 2026-03-15

## Overview

Add the ability to view and download uploaded receipts from within a list. Currently receipts can be uploaded but there is no way to view the original file or its parsed contents after upload.

## User-Facing Behaviour

### Entry Points

The receipt badge in the `ListDetail` header is split into two distinct controls:

- **Upload button** (existing): opens the existing `ReceiptUploadModal` — no change to this flow
- **Receipt count badge** (new trigger): clicking the count badge opens the new `ReceiptViewerModal`

If there are no receipts yet, the badge is hidden; only the upload button is shown.

### ReceiptViewerModal — List View

A modal (centered on mobile, right-side drawer full-height on screens ≥ 640px) that opens when the user clicks the receipt count badge.

Displays all receipts for the current list, each showing:
- Thumbnail (for images) or a document icon (for PDF/text)
- Date extracted from receipt (or upload date if unparsed)
- Shop name (if parsed and shop is available)
- Total (if parsed)
- Status badge: display label `"Parsed"`, `"Pending"`, or `"Error"` mapped from model values `"parsed"`, `"new"`, `"error"` respectively

Clicking a receipt row navigates to the detail view within the same modal.

### ReceiptViewerModal — Detail View

Side-by-side layout within the same modal (stacked on mobile):

**Left panel — original file:**
- While loading (blob fetch in progress): show a spinner
- Image receipts: `<img>` sourced from a blob URL created via authenticated fetch
- Text receipts (`.txt`): scrollable `<pre>` block with the raw text content
- PDF receipts: rendered as an `<img>` from blob URL if possible, otherwise a "Download to view PDF" message

Whether a receipt is a text receipt is determined by checking whether `receipt.image_path` ends with `.txt`.

**Right panel — parsed data:**
- Shown only when `status === "parsed"`
- Shop name and date at the top
- Itemised table: name | qty | unit | price
- Total at the bottom
- If `status === "new"`: spinner + "Still processing…"
- If `status === "error"`: "Could not parse this receipt"
- `items` may be an empty array if the data source does not preload them — render the table gracefully with an empty state ("No items available")

**Header:**
- "← Back" button (`data-testid="receipt-viewer-back"`) returns to the list view
- **Download button** (`data-testid="receipt-viewer-download"`): triggers a browser file download of the original receipt file

### Download Behaviour

Clicking Download calls `downloadReceiptFile(receiptId, token)` which fetches `GET ${API_BASE_URL}/api/receipts/:id/file` with the JWT bearer token, converts the response to a blob, creates a temporary `<a>` element, and triggers a browser download. The filename is derived from the receipt date and shop name where available (e.g. `receipt-2026-03-10-costco.jpg`), falling back to `receipt-{id}.{ext}`.

---

## Backend

### Model Change — `models/models.go`

Add a `Shop` association field to the `Receipt` struct so GORM can preload it:

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

### Preloading Changes — `handlers/lists.go`

Both `GetList` and `GetLists` must be updated to preload the shop and (for `GetList`) items:

```go
// GetList — detail view needs shop name and parsed items
db.Preload("Items").Preload("Receipts").Preload("Receipts.Items").Preload("Receipts.Shop")

// GetLists — summary view, receipt count only; no shop preload needed
db.Preload("Receipts")
```

### New Endpoint: `GET /api/receipts/:id/file`

**Auth:** Protected — requires valid JWT.
**Authorization:** Receipt must belong to the requesting user's family (verified via `family_id`).

**Behaviour:**
1. Look up `Receipt` by `id` scoped to `family_id` from JWT context using `Where("id = ? AND family_id = ?", id, familyID)` — consistent with the existing pattern in `GetList`.
2. Get `dataPath` from `KINCART_DATA_PATH` env var, defaulting to `"./kincart-data"`.
3. Resolve to absolute paths before any comparison:
   ```go
   absDataPath, _ := filepath.Abs(dataPath)
   absFilePath, _ := filepath.Abs(filepath.Join(dataPath, receipt.ImagePath))
   ```
4. **Path traversal check (Linux/Docker only):** `strings.HasPrefix(absFilePath, absDataPath+"/")`. Return `400` if the check fails. Note: using `"/"` directly rather than `os.PathSeparator` is intentional — the project runs on Linux in Docker.
5. Detect content type from file extension.
6. Derive a download filename: `receipt-{YYYY-MM-DD}-{shopname}.{ext}` (shop name lowercased, spaces → hyphens). If `receipt.Shop` is nil (no shop matched), fall back to `receipt-{id}.{ext}` — do not dereference a nil pointer. Requires preloading `Shop` on the receipt lookup — add `Preload("Shop")` to the query.
7. Serve using `c.FileAttachment(absFilePath, derivedFilename)`.

**Error responses:**
- `404` if receipt not found or not in family
- `404` if file missing on disk (log the inconsistency server-side)
- `400` if resolved path fails traversal check

**Route registration:** Add to the protected group in `cmd/server/main.go`:
```
GET /api/receipts/:id/file  →  handlers.GetReceiptFile
```

---

## Frontend

### `ListDetail.jsx` — Header Changes

Split the current single receipt button into two controls:

```jsx
{/* Upload button — existing behaviour */}
<button data-testid="upload-receipt-btn" onClick={() => setIsReceiptModalOpen(true)}>
  <UploadIcon />
</button>

{/* Viewer badge — new */}
{list.receipts?.length > 0 && (
  <button data-testid="view-receipts-btn" onClick={() => setIsReceiptViewerOpen(true)}>
    <ReceiptIcon />
    <span className="badge">{list.receipts.length}</span>
  </button>
)}
```

Add state: `const [isReceiptViewerOpen, setIsReceiptViewerOpen] = useState(false)`

### New Component: `ReceiptViewerModal.jsx`

**Props:** `{ receipts, isOpen, onClose }`

Token is obtained internally via `const { token } = useAuth()` — consistent with the pattern already used in the component file for other API calls.

Each `receipt` in `receipts` has the shape:
```js
{
  id: number,
  status: "new" | "parsed" | "error",
  date: string,           // ISO date string
  total: number,
  image_path: string,     // relative path (snake_case from JSON); used only to detect file type (.txt check)
  shop: { id, name } | null,
  items: [{ id, name, quantity, unit, price, total_price }]  // may be empty array
}
```

**Internal state:**
- `selectedReceiptId: number | null` — `null` = list view, set = detail view
- `blobUrl: string | null` — object URL for image/PDF display (null while loading or for text receipts)
- `textContent: string | null` — raw text for `.txt` receipts (null while loading or for non-text receipts)
- `isLoadingFile: boolean` — true while fetching the receipt file

**File loading logic:**

When `selectedReceiptId` changes (and is non-null), trigger a fetch:

```js
const isTextReceipt = receipt.image_path.endsWith('.txt')

const res = await fetch(`${API_BASE_URL}/api/receipts/${receipt.id}/file`, {
  headers: { Authorization: `Bearer ${token}` }
})

if (isTextReceipt) {
  const text = await res.text()
  setTextContent(text)
} else {
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  setBlobUrl(url)
}
setIsLoadingFile(false)
```

On unmount or when `selectedReceiptId` changes away, revoke the blob URL:
```js
return () => { if (blobUrl) URL.revokeObjectURL(blobUrl) }
```

**File loading and download utilities** — defined inline in `ReceiptViewerModal.jsx` (the project does not have a shared `api.js`):

- `fetchReceiptBlob(receiptId, token)` — fetch + blob + `URL.createObjectURL`
- `downloadReceiptFile(receiptId, filename, token)` — fetch + blob + temporary `<a download>` click + `URL.revokeObjectURL`

### Styles

Add `frontend/src/components/ReceiptViewerModal.css` with:
- `.receipt-viewer-modal` — base modal styles
- `.receipt-viewer-drawer` — right-side panel at `min-width: 640px` (fixed position, full height, min-width 480px)
- `.receipt-detail` — side-by-side layout using CSS grid or flexbox
- Mobile: stacked (image then items), desktop: two columns

---

## Data Flow

```
ListDetail mounts
  → fetchList() returns list with receipts[].shop and receipts[].items preloaded
  → User clicks receipt count badge (data-testid="view-receipts-btn")
  → ReceiptViewerModal opens with receipts[]
  → List view renders: thumbnail/icon, date, shop name, total, status badge

  → User clicks a receipt row
  → selectedReceiptId set, isLoadingFile = true → detail view renders with spinner
  → For image/PDF: fetch with auth → blob → URL.createObjectURL → <img src={blobUrl}>
  → For .txt: fetch with auth → res.text() → <pre>{textContent}</pre>
  → isLoadingFile = false, panel shows content

  → User clicks Download (data-testid="receipt-viewer-download")
    → downloadReceiptFile() → fetch → blob → <a download> click → URL.revokeObjectURL

  → User clicks Back (data-testid="receipt-viewer-back")
    → selectedReceiptId = null → list view
    → blobUrl revoked, textContent cleared
```

---

## Implementation Notes

- Use `receipt.items ?? []` (not `||`) when rendering the items table — `items` will be `null` when the data comes from `GetLists` (summary), which does not preload receipt items.
- Extract the file extension for the download filename from `receipt.image_path` client-side, not from the `Content-Disposition` response header.
- Revoke blob URLs on modal close (`isOpen` → false) in addition to on `selectedReceiptId` change, to prevent leaks across open/close cycles.

---

## Out of Scope

- Deleting receipts
- Re-parsing failed receipts
- Viewing receipts outside of the list context (no global receipts page)
- Editing parsed receipt data

---

## Files Changed

**Backend:**
- `backend/internal/models/models.go` — add `Shop *Shop` association field to `Receipt`
- `backend/internal/handlers/receipts.go` — add `GetReceiptFile` handler
- `backend/internal/handlers/lists.go` — add `Preload("Receipts.Items")` and `Preload("Receipts.Shop")` to `GetList` only
- `backend/cmd/server/main.go` — register new route
- `backend/internal/handlers/receipts_test.go` — tests for new endpoint

**Frontend:**
- `frontend/src/pages/ListDetail.jsx` — split receipt button
- `frontend/src/components/ReceiptViewerModal.jsx` — new component (list + detail views, inline fetch utilities)
- `frontend/src/components/ReceiptViewerModal.css` — styles for modal, drawer, and detail layout
- `frontend/src/components/ReceiptViewerModal.test.jsx` — tests
