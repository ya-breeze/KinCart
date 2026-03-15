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

A modal (centered on mobile, drawer-style on wider screens) that opens when the user clicks the receipt count badge.

Displays all receipts for the current list, each showing:
- Thumbnail (for images) or a document icon (for PDF/text)
- Date extracted from receipt (or upload date if unparsed)
- Shop name (if parsed)
- Total (if parsed)
- Status badge: `parsed` | `pending` | `error`

Clicking a receipt row navigates to the detail view within the same modal.

### ReceiptViewerModal — Detail View

Side-by-side layout within the same modal:

**Left panel — original file:**
- Image receipts: full-size scrollable image
- Text receipts (`.txt`): scrollable `<pre>` block with the raw text content
- PDF receipts: rendered as an `<img>` if possible, otherwise a "Download to view PDF" message

**Right panel — parsed data:**
- Shown only when `status === "parsed"`
- Shop name and date at the top
- Itemised table: name | qty | unit | price
- Total at the bottom
- If `status === "pending"`: spinner + "Still processing…"
- If `status === "error"`: "Could not parse this receipt"

**Header:**
- "← Back" button returns to the list view
- **Download button**: triggers a browser file download of the original receipt file

### Download Behaviour

Clicking Download fetches `GET /api/receipts/:id/file` with the JWT token and triggers a browser file download using the response blob. The filename is derived from the receipt date and shop name where available (e.g. `receipt-2026-03-10-costco.jpg`).

---

## Backend

### New Endpoint: `GET /api/receipts/:id/file`

**Auth:** Protected — requires valid JWT.
**Authorization:** Receipt must belong to the requesting user's family (verified via `FamilyID`).

**Behaviour:**
1. Look up `Receipt` by `id`, scoped to `family_id` from JWT context.
2. Resolve the absolute file path: `KINCART_DATA_PATH + "/" + receipt.ImagePath`.
3. Detect content type from file extension (`.jpg`, `.png`, `.pdf`, `.txt`).
4. Set `Content-Disposition: attachment; filename="<derived-name>"` for download.
5. Stream the file with `c.File()`.

**Error responses:**
- `404` if receipt not found or not in family
- `404` if file missing on disk (log the inconsistency)

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
<button onClick={() => setIsReceiptModalOpen(true)}>
  <UploadIcon />
</button>

{/* Viewer badge — new */}
{list.receipts?.length > 0 && (
  <button onClick={() => setIsReceiptViewerOpen(true)}>
    <ReceiptIcon />
    <span className="badge">{list.receipts.length}</span>
  </button>
)}
```

Add state: `const [isReceiptViewerOpen, setIsReceiptViewerOpen] = useState(false)`

### New Component: `ReceiptViewerModal.jsx`

**Props:** `{ receipts, listId, isOpen, onClose }`

Internal state:
- `selectedReceiptId: number | null` — `null` = list view, set = detail view
- `fileContent: string | null` — cached text content for text receipts

**Subcomponent: `ReceiptDetail`** (can be a local component within the same file)

Props: `{ receipt, onBack }`

Renders the side-by-side layout described above. Fetches the file URL as `/api/receipts/${receipt.id}/file` for display (image `src` or text fetch).

### API utility

Add `getReceiptFileUrl(receiptId)` to the existing API helpers — returns the authenticated URL or triggers a fetch+blob download depending on usage.

---

## Data Flow

```
ListDetail mounts
  → fetchList() returns list with receipts[] preloaded
  → User clicks receipt count badge
  → ReceiptViewerModal opens with receipts[]
  → User clicks a receipt row
  → selectedReceiptId set → detail view renders
  → Image: <img src="/api/receipts/:id/file" /> with auth header via fetch+blob URL
  → Text: fetch /api/receipts/:id/file → display as <pre>
  → User clicks Download → fetch /api/receipts/:id/file → trigger blob download
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
- `backend/cmd/server/main.go` — register new route
- `backend/internal/handlers/receipts_test.go` — tests for new endpoint

**Frontend:**
- `frontend/src/pages/ListDetail.jsx` — split receipt button
- `frontend/src/components/ReceiptViewerModal.jsx` — new component (list + detail views)
- `frontend/src/components/ReceiptViewerModal.test.jsx` — tests
