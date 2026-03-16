# Plan: Two-Phase AI Receipt Matching with Manual Confirmation

## Problem
When a receipt is parsed, items like "selský jogurt 2%" don't match planned "jogurt" because matching uses `strings.EqualFold` (exact case-insensitive). This creates duplicate items.

## Design Overview

Receipt processing becomes a **two-phase flow**:
1. **Phase 1 (automatic):** Parse receipt → AI suggests matches with confidence % → auto-accept high-confidence matches
2. **Phase 2 (manual):** User reviews uncertain/unmatched items → confirms, re-maps, or dismisses

Both the planned name and the receipt name are stored as **item aliases** for future automatic matching.

---

## Data Model Changes

### New: `ItemAlias` model
```go
type ItemAlias struct {
    ID        uint   `gorm:"primaryKey" json:"id"`
    FamilyID  uint   `gorm:"not null;index" json:"family_id"`
    ItemName  string `gorm:"not null" json:"item_name"`  // canonical name (planned)
    AliasName string `gorm:"not null" json:"alias_name"` // receipt/alternate name
    CreatedAt time.Time `json:"created_at"`
}
// Unique constraint on (family_id, LOWER(alias_name))
```

Purpose: When user confirms "jogurt" = "selský jogurt 2%", store this mapping. Next time the same receipt name appears, skip AI matching entirely.

### New fields on `ReceiptItem`
```go
type ReceiptItem struct {
    // ... existing fields ...
    MatchedItemID  *uint   `json:"matched_item_id"`   // planned item it matched to (nil = unmatched)
    MatchStatus    string  `json:"match_status"`       // "auto", "confirmed", "manual", "unmatched", "dismissed"
    Confidence     float64 `json:"confidence"`         // AI confidence 0-100
    SuggestedItems string  `json:"suggested_items"`    // JSON array of {item_id, item_name, confidence}
}
```

### New field on `Receipt`
```go
type Receipt struct {
    // ... existing fields ...
    Status string // add new value: "pending_review" (between "parsed" and fully resolved)
}
```

---

## AI Changes (`backend/internal/ai/gemini.go`)

### New method: `MatchReceiptItems`

After parsing the receipt, make a second AI call that receives:
- List of parsed receipt item names
- List of planned item names

Returns match suggestions:
```go
type MatchSuggestion struct {
    ReceiptItemName string        `json:"receipt_item_name"`
    Matches         []MatchCandidate `json:"matches"` // 0 or more
}

type MatchCandidate struct {
    PlannedItemName string  `json:"planned_item_name"`
    Confidence      float64 `json:"confidence"` // 0-100
}
```

**Prompt design:**
```
You are a shopping item matcher. Given receipt items and a planned shopping list,
determine which receipt items correspond to which planned items.

Rules:
- A receipt item may match 0 or 1 planned items
- A planned item may match 0 or 1 receipt items
- Return confidence as percentage (0-100)
- Consider that planned items are often short/generic ("jogurt") while receipt
  items are specific ("selský jogurt 2%")
- Items in different languages or with brand names can still match
- If no good match exists, return empty matches array

Receipt items: [...]
Planned items: [...]
```

### Optimization: Check `ItemAlias` table first

Before calling AI matching, look up each receipt item name in `ItemAlias`. If a known alias exists, use that mapping directly (confidence=100, status="auto") and skip the AI call for that item.

---

## Backend Flow Changes (`backend/internal/services/receipt_service.go`)

### Modified `ProcessReceipt` flow:

```
1. Parse receipt with Gemini (existing)
2. Create ReceiptItem records (existing)
3. NEW: Check ItemAlias table for known mappings
4. NEW: For unresolved items, call MatchReceiptItems AI
5. NEW: Apply matching results:
   - confidence >= 90%: auto-match (mark MatchStatus="auto", link item, mark bought)
   - confidence < 90%: store suggestions, set MatchStatus="unmatched"
6. Set receipt status:
   - All items auto-matched → "parsed" (fully resolved)
   - Some items need review → "pending_review"
```

### New method: `ConfirmMatch`
Called when user confirms/changes a match from the UI.

```go
func (s *ReceiptService) ConfirmMatch(receiptItemID uint, plannedItemID *uint, familyID uint) error {
    // 1. Update ReceiptItem.MatchedItemID and MatchStatus
    // 2. If plannedItemID != nil:
    //    a. Link planned item to receipt item, mark bought, update price
    //    b. Create ItemAlias (receipt_name → planned_name) for future matching
    // 3. If plannedItemID == nil (dismissed):
    //    a. Create new item in list (current behavior for unmatched)
    //    b. Store receipt name as alias of itself for frequency tracking
    // 4. Recalculate list total
}
```

### New method: `ManualMatch`
Called when user drags/assigns a receipt item to a planned item manually.

```go
func (s *ReceiptService) ManualMatch(receiptItemID uint, plannedItemID uint, familyID uint) error {
    // Same as ConfirmMatch but MatchStatus="manual"
}
```

### New method: `DismissReceiptItem`
Called when user says "this receipt item is not in my list" (e.g., bag fee).

```go
func (s *ReceiptService) DismissReceiptItem(receiptItemID uint, familyID uint) error {
    // Set MatchStatus="dismissed", don't create item in list
}
```

---

## New API Endpoints

```
GET    /api/receipts/:id/matches      → Get receipt with match suggestions
PATCH  /api/receipts/:id/matches/:receipt_item_id  → Confirm/change a match
POST   /api/receipts/:id/matches/:receipt_item_id/dismiss  → Dismiss an item
POST   /api/receipts/:id/matches/confirm-all  → Accept all current matches
```

### `GET /api/receipts/:id/matches` response:
```json
{
  "receipt_id": 1,
  "status": "pending_review",
  "shop_name": "Lidl",
  "date": "2026-03-15",
  "total": 450.50,
  "items": [
    {
      "receipt_item_id": 10,
      "receipt_name": "selský jogurt 2%",
      "quantity": 2,
      "price": 29.90,
      "total_price": 59.80,
      "match_status": "auto",
      "confidence": 95,
      "matched_item": {"id": 5, "name": "jogurt"},
      "suggestions": [
        {"item_id": 5, "item_name": "jogurt", "confidence": 95}
      ]
    },
    {
      "receipt_item_id": 11,
      "receipt_name": "Président Brie 200g",
      "quantity": 1,
      "price": 89.90,
      "total_price": 89.90,
      "match_status": "unmatched",
      "confidence": 0,
      "matched_item": null,
      "suggestions": [
        {"item_id": 7, "item_name": "sýr", "confidence": 65},
        {"item_id": 12, "item_name": "brie", "confidence": 55}
      ]
    },
    {
      "receipt_item_id": 12,
      "receipt_name": "Taška",
      "quantity": 1,
      "price": 5.00,
      "total_price": 5.00,
      "match_status": "unmatched",
      "confidence": 0,
      "matched_item": null,
      "suggestions": []
    }
  ],
  "unmatched_planned_items": [
    {"id": 7, "name": "sýr"},
    {"id": 12, "name": "brie"},
    {"id": 15, "name": "mléko"}
  ]
}
```

### `PATCH /api/receipts/:id/matches/:receipt_item_id` body:
```json
{"planned_item_id": 7}       // Confirm or change match
// or
{"planned_item_id": null}     // Create as new item
```

---

## Frontend Changes

### New: `ReceiptMatchModal.jsx`

Shown after receipt upload when status is `"pending_review"`. Three sections:

**Section 1: Auto-matched items (green)**
- Shows pairs: "jogurt" ← "selský jogurt 2%" (95%)
- Each has a checkmark, user can click to change/unmatch

**Section 2: Items needing review (yellow)**
- Receipt item name on left
- Dropdown/list of suggested planned items with confidence % on right
- "Create as new" option
- "Dismiss" button (for bag fees, etc.)

**Section 3: Unmatched planned items (grey)**
- Planned items that weren't matched to any receipt item
- User can drag a receipt item here (or the item shows as "not bought")

**Actions:**
- "Confirm All" button — accepts all auto-matches and creates new items for unmatched
- Individual confirm/change per item
- "Match Manually" — select a receipt item, then select a planned item

### Modified: `ReceiptUploadModal.jsx`

After upload, if response status is `"pending_review"`:
- Instead of auto-closing, transition to `ReceiptMatchModal`
- If `"parsed"` (all auto-matched), close as before

### Modified: `ListDetail.jsx`

- Add "Review Matches" button on receipt badge if receipt status is `"pending_review"`
- Show both names on item tooltip: "Receipt: selský jogurt 2% → Planned: jogurt"

---

## ItemAlias Usage for Fast List Creation

When creating a new shopping list and the user types an item name:
- Query `ItemAlias` + `ItemFrequency` to suggest items
- Show both the canonical name and known aliases
- Example: User types "sel" → suggests "selský jogurt 2%" which maps to canonical "jogurt"
- The `GET /api/family/frequent-items` endpoint should join with `ItemAlias` to return alias info

---

## Auto-Match Confidence Threshold

- **>= 90%**: Auto-match, no user action needed
- **50-89%**: Show as suggestion, user confirms
- **< 50%**: Show as unmatched, user can manually assign
- **0 suggestions**: "Create as new" or "Dismiss"

The 90% threshold should be configurable per family (stored in `Family.Config` or as env var) to let users tune aggressiveness.

---

## Implementation Order

1. **DB models**: Add `ItemAlias`, extend `ReceiptItem`, migrate
2. **AI matching**: New `MatchReceiptItems` method in gemini.go
3. **Service layer**: Modify `ProcessReceipt`, add `ConfirmMatch`/`ManualMatch`/`DismissReceiptItem`
4. **API endpoints**: New match review/confirm endpoints in receipts handler
5. **Frontend**: `ReceiptMatchModal` component
6. **Frontend**: Modify upload flow to show match modal
7. **Alias integration**: Wire `ItemAlias` into frequent items / autocomplete
8. **Tests**: Backend service tests, handler tests, frontend component tests
