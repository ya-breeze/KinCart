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

A modal (centered on mobile, right-side panel full-height on screens ≥ 640px) that opens when the user clicks the receipt count badge.

Displays all receipts for the current list, each showing:
- Thumbnail (for images) or a document icon (for PDF/text)
- Date extracted from receipt (or upload date if unparsed)
- Shop name (if parsed and shop is preloaded)
- Total (if parsed)
- Status badge: display label `"Parsed"`, `"Pending"`, or `"Error"` mapped from model values `"parsed"`, `"new"`, `"error"` respectively

Clicking a receipt row navigates to the detail view within the same modal.

### ReceiptViewerModal — Detail View

Side-by-side layout within the same modal (stacked on mobile):

**Left panel — original file:**
- Image receipts: full-size scrollable image
- Text receipts (`.txt`): scrollable `<pre>` block with the raw text content
- PDF receipts: rendered as an `<img>` if possible, otherwise a "Download to view PDF" message

**Right panel — parsed data:**
- Shown only when `status === "parsed"`
- Shop name and date at the top
- Itemised table: name | qty | unit | price
- Total at the bottom
- If `status === "new"`: spinner + "Still processing…"
- If `status === "error"`: "Could not parse this receipt"

**Header:**
- "← Back" button (data-testid="receipt-viewer-back") returns to the list view
- **Download button** (data-testid="receipt-viewer-download"): triggers a browser file download of the original receipt file

### Download Behaviour

Clicking Download calls `downloadReceiptFile(receiptId, token)` which fetches `GET /api/receipts/:id/file` with the JWT bearer token, converts the response to a blob, creates a temporary anchor element, and triggers a browser download. The filename is derived from the receipt date and shop name where available (e.g. `receipt-2026-03-10-costco.jpg`), falling back to `receipt-{id}.{ext}`.

---

## Backend

### Preloading Changes — `handlers/lists.go`

Both `GetList` and `GetLists` must be updated to preload the shop name and parsed items on receipts:

```go
// GetList (already preloads Receipts — extend it)
db.Preload("Receipts.Items").Preload("Receipts.Shop")

// GetLists (already preloads Receipts — extend it)
db.Preload("Receipts.Shop")  // items not needed in list summary view
```

### New Endpoint: `GET /api/receipts/:id/file`

**Auth:** Protected — requires valid JWT.
**Authorization:** Receipt must belong to the requesting user's family (verified via `FamilyID`).

**Behaviour:**
1. Look up `Receipt` by `id`, scoped to `family_id` from JWT context (use `db.ScopedFirst`).
2. Resolve the absolute file path using `filepath.Join(dataPath, receipt.ImagePath)` where `dataPath` is `KINCART_DATA_PATH` env var, defaulting to `"./kincart-data"` (matching the pattern in `receipt_service.go`).
3. **Path traversal check:** Verify the resolved absolute path has the absolute `dataPath` as a prefix (`strings.HasPrefix(absPath, absDataPath)`). Return `400` if not.
4. Detect content type from file extension (`.jpg`, `.png`, `.pdf`, `.txt`).
5. Derive a download filename: `receipt-{YYYY-MM-DD}-{shopname}.{ext}` (shop name lowercased, spaces replaced with hyphens), falling back to `receipt-{id}.{ext}`.
6. Serve using `c.FileAttachment(absPath, derivedFilename)` — this sets `Content-Disposition: attachment` correctly without header conflicts.

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
{/* Upload button — existing behaviour, data-testid="upload-receipt-btn" */}
<button data-testid="upload-receipt-btn" onClick={() => setIsReceiptModalOpen(true)}>
  <UploadIcon />
</button>

{/* Viewer badge — new, data-testid="view-receipts-btn" */}
{list.receipts?.length > 0 && (
  <button data-testid="view-receipts-btn" onClick={() => setIsReceiptViewerOpen(true)}>
    <ReceiptIcon />
    <span className="badge">{list.receipts.length}</span>
  </button>
)}
```

Add state: `const [isReceiptViewerOpen, setIsReceiptViewerOpen] = useState(false)`

### New Component: `ReceiptViewerModal.jsx`

**Props:** `{ receipts, listId, isOpen, onClose }`

Each `receipt` in `receipts` has the shape:
```js
{
  id: number,
  status: "new" | "parsed" | "error",
  date: string,         // ISO date string
  total: number,
  imagePath: string,    // relative path, not used directly — fetch via API
  shop: { id, name } | null,
  items: [{ id, name, quantity, unit, price, totalPrice }]  // only on GetList detail
}
```

Internal state:
- `selectedReceiptId: number | null` — `null` = list view, set = detail view
- `blobUrl: string | null` — object URL for the receipt file (image display)

**Image display pattern:** The component must not use `<img src="/api/receipts/:id/file">` directly, as that cannot send an `Authorization` header. Instead:
1. On receipt selection, `fetch("/api/receipts/:id/file", { headers: { Authorization: "Bearer " + token } })`
2. Convert response to blob: `const blob = await res.blob()`
3. Create object URL: `const url = URL.createObjectURL(blob)`
4. Set as `blobUrl` state, use as `<img src={blobUrl}>`
5. On component unmount or receipt change: `URL.revokeObjectURL(blobUrl)`

**API utilities** (add to existing API helper file):
- `fetchReceiptBlob(receiptId, token)` → returns a blob URL string for display
- `downloadReceiptFile(receiptId, token)` → fetches blob, creates a temporary `<a>` element, triggers download, revokes the object URL

### Responsive layout

- Below `640px`: modal is full-width centered overlay; detail view stacks image on top, items below
- At `640px` and above: modal is a right-side drawer (fixed position, full height, `min-width: 480px`); detail view is side-by-side (image left, items right)

---

## Data Flow

```
ListDetail mounts
  → fetchList() returns list with receipts[].shop and receipts[].items preloaded
  → User clicks receipt count badge (data-testid="view-receipts-btn")
  → ReceiptViewerModal opens with receipts[]
  → User clicks a receipt row
  → selectedReceiptId set → detail view renders
  → fetchReceiptBlob(id, token) → fetch with auth → blob → object URL → <img src>
  → Text receipts: same fetch → read as text → display in <pre>
  → User clicks Download (data-testid="receipt-viewer-download")
    → downloadReceiptFile(id, token) → fetch → blob → <a download> click → revoke
  → User clicks Back (data-testid="receipt-viewer-back")
    → selectedReceiptId = null → list view
    → blobUrl revoked
```

---

## Out of Scope

- Deleting receipts
- Re-parsing failed receipts
- Viewing receipts outside of the list context (no global receipts page)
- Editing parsed receipt data

---

## Files Changed

**Backend:**
- `backend/internal/handlers/receipts.go` — add `GetReceiptFile` handler
- `backend/internal/handlers/lists.go` — add `Preload("Receipts.Items")` and `Preload("Receipts.Shop")`
- `backend/cmd/server/main.go` — register new route
- `backend/internal/handlers/receipts_test.go` — tests for new endpoint

**Frontend:**
- `frontend/src/pages/ListDetail.jsx` — split receipt button
- `frontend/src/components/ReceiptViewerModal.jsx` — new component (list + detail views)
- `frontend/src/components/ReceiptViewerModal.test.jsx` — tests
