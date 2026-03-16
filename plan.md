# Plan: Two-Phase AI Receipt Matching with Manual Confirmation

## Problem
When a receipt is parsed, items like "selský jogurt 2%" don't match planned "jogurt" because matching uses `strings.EqualFold` (exact case-insensitive). This creates duplicate items.

## Design Overview

Receipt processing becomes a **two-phase flow**:
1. **Phase 1 (automatic):** Parse receipt → AI suggests matches with confidence % → auto-accept high-confidence matches
2. **Phase 2 (manual):** User reviews uncertain/unmatched items → confirms, re-maps, or dismisses

Every confirmed match stores **both names** (planned + receipt) along with **shop, price, and usage count** in an `ItemAlias` table. This builds a per-family purchase history where one generic planned name (e.g., "jogurt") can map to many specific receipt names (e.g., "selský jogurt 2%", "Activia jahoda 150g") — enabling both smarter receipt matching and richer item suggestions during list creation.

---

## Data Model Changes

### New: `ItemAlias` model
```go
type ItemAlias struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    FamilyID      uint      `gorm:"not null;index" json:"family_id"`
    PlannedName   string    `gorm:"not null" json:"planned_name"`   // generic name user writes on list ("jogurt")
    ReceiptName   string    `gorm:"not null" json:"receipt_name"`   // specific name from receipt ("selský jogurt 2%")
    ShopID        *uint     `json:"shop_id"`                        // which shop this was bought at (nullable)
    Shop          *Shop     `gorm:"foreignKey:ShopID" json:"shop"`
    LastPrice     float64   `json:"last_price"`                     // last known price
    PurchaseCount int       `gorm:"default:1" json:"purchase_count"` // how many times this exact mapping was used
    LastUsedAt    time.Time `json:"last_used_at"`                   // when this mapping was last confirmed
    CreatedAt     time.Time `json:"created_at"`
}
// Unique constraint on (family_id, LOWER(planned_name), LOWER(receipt_name), shop_id)
```

**Key insight:** One planned name maps to **many** receipt names. "jogurt" can resolve to:
- "selský jogurt 2%" at Lidl (bought 5x, last price 29.90 Kč)
- "Activia jahoda 150g" at Albert (bought 2x, last price 34.90 Kč)
- "BIO jogurt bílý" at Lidl (bought 1x, last price 42.00 Kč)

This enables two use cases:
1. **Receipt matching:** receipt says "selský jogurt 2%" → look up alias → auto-match to planned "jogurt"
2. **List creation suggestions:** user types "jogurt" → show all known products they've bought as "jogurt" before, with shop name, price, and frequency, so they can pick or just leave the generic name

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

**All AI calls MUST use Gemini's structured output** via `ResponseMIMEType: "application/json"` + `ResponseSchema` (a `*genai.Schema`). This enforces strict typing at the API level — Gemini is constrained to return only valid JSON matching the schema. No free-form text parsing or markdown fence stripping needed.

The existing `ParseReceipt` and `ParseReceiptText` already do this correctly via `buildReceiptSchema()`. The new `MatchReceiptItems` method must follow the same pattern.

### New method: `MatchReceiptItems`

After parsing the receipt, make a second AI call that receives:
- List of parsed receipt item names
- List of planned item names

**Go types:**
```go
type MatchResult struct {
    Suggestions []MatchSuggestion `json:"suggestions"`
}

type MatchSuggestion struct {
    ReceiptItemName string           `json:"receipt_item_name"`
    Matches         []MatchCandidate `json:"matches"` // 0 or more
}

type MatchCandidate struct {
    PlannedItemName string  `json:"planned_item_name"`
    Confidence      int     `json:"confidence"` // 0-100 integer
}
```

**Gemini response schema (strict):**
```go
func buildMatchSchema() *genai.Schema {
    return &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "suggestions": {
                Type: genai.TypeArray,
                Items: &genai.Schema{
                    Type: genai.TypeObject,
                    Properties: map[string]*genai.Schema{
                        "receipt_item_name": {Type: genai.TypeString},
                        "matches": {
                            Type: genai.TypeArray,
                            Items: &genai.Schema{
                                Type: genai.TypeObject,
                                Properties: map[string]*genai.Schema{
                                    "planned_item_name": {Type: genai.TypeString},
                                    "confidence":        {Type: genai.TypeInteger},
                                },
                                Required: []string{"planned_item_name", "confidence"},
                            },
                        },
                    },
                    Required: []string{"receipt_item_name", "matches"},
                },
            },
        },
        Required: []string{"suggestions"},
    }
}
```

**API call:**
```go
func (c *GeminiClient) MatchReceiptItems(ctx context.Context, receiptItems []string, plannedItems []string) (*MatchResult, error) {
    // Build prompt with both lists
    // Call c.client.Models.GenerateContent with:
    //   ResponseMIMEType: "application/json",
    //   ResponseSchema:   buildMatchSchema(),
    // Unmarshal into MatchResult
}
```

**Prompt design:**
```
You are a shopping item matcher. Given receipt items and a planned shopping list,
determine which receipt items correspond to which planned items.

Rules:
- A receipt item may match 0 or 1 planned items
- A planned item may match 0 or 1 receipt items
- Return confidence as integer percentage (0-100)
- Consider that planned items are often short/generic ("jogurt") while receipt
  items are specific ("selský jogurt 2%")
- Items in different languages or with brand names can still match
- If no good match exists, return empty matches array
- You MUST return one suggestion entry per receipt item, even if matches is empty

Receipt items: [...]
Planned items: [...]
```

### Optimization: Check `ItemAlias` table first

Before calling AI matching, look up each receipt item name in `ItemAlias` (filtered by family + optionally by shop). If a known alias exists, use that mapping directly (confidence=100, status="auto") and skip the AI call for that item. If the same receipt name maps to different planned names (rare edge case from different shops), include all as suggestions.

---

## Backend Flow Changes (`backend/internal/services/receipt_service.go`)

### Modified `ProcessReceipt` flow:

```
1. Parse receipt with Gemini (existing) — returns store name, items, prices
2. Create ReceiptItem records (existing)
3. NEW: Find/create shop from receipt store name (needed for alias lookup)
4. NEW: Check ItemAlias table for known mappings
   - For each receipt item, query: WHERE family_id=? AND LOWER(receipt_name)=LOWER(?)
   - If alias found AND the planned_name exists in current list → auto-match (confidence=100)
   - If alias found but planned_name NOT in list → still useful context for AI
5. NEW: For unresolved items, call MatchReceiptItems AI
   - Pass both the planned items AND any alias hints from step 4
6. NEW: Apply matching results:
   - confidence >= 90%: auto-match (mark MatchStatus="auto", link item, mark bought)
   - confidence < 90%: store suggestions, set MatchStatus="unmatched"
7. NEW: Identify edge cases:
   a. **Extra items** (receipt items with 0 match suggestions):
      - These are items bought but NOT on the planned list (impulse buys, bag fees, etc.)
      - Set MatchStatus="unmatched", suggestions=[]
      - User can later: "Add to list" (creates new item), "Dismiss" (ignore), or
        "Match to planned" (manually link to an existing planned item)
   b. **Unbought planned items** (planned items not matched by any receipt item):
      - Collect all planned items whose IDs do NOT appear in any ReceiptItem.MatchedItemID
      - Return these as `unmatched_planned_items` in the response
      - These remain in their current state (not marked bought)
      - User can later: leave as-is (still need to buy), mark bought manually,
        or delete from list
8. Set receipt status:
   - All items auto-matched AND no unmatched planned items → "parsed" (fully resolved)
   - Any items need review OR unmatched planned items exist → "pending_review"
```

### New method: `ConfirmMatch`
Called when user confirms/changes a match from the UI.

```go
func (s *ReceiptService) ConfirmMatch(receiptItemID uint, plannedItemID *uint, familyID uint) error {
    // 1. Update ReceiptItem.MatchedItemID and MatchStatus
    // 2. If plannedItemID != nil:
    //    a. Link planned item to receipt item, mark bought, update price
    //    b. Upsert ItemAlias: (planned_name, receipt_name, shop_id)
    //       - If alias exists: increment PurchaseCount, update LastPrice, LastUsedAt
    //       - If new: create with PurchaseCount=1
    // 3. If plannedItemID == nil (create as new — for EXTRA/impulse items):
    //    a. Create new item in list with name from receipt, mark as bought
    //    b. Set price/quantity from receipt item
    //    c. Upsert ItemAlias with planned_name = receipt_name (self-alias for frequency tracking)
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
POST   /api/receipts/:id/matches/confirm-all  → Accept all current matches, add extras as new items, leave unbought planned items unchanged
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
      "suggestions": [],
      "is_extra": true
    },
    {
      "receipt_item_id": 13,
      "receipt_name": "Chipsy Bohemia 150g",
      "quantity": 1,
      "price": 44.90,
      "total_price": 44.90,
      "match_status": "unmatched",
      "confidence": 0,
      "matched_item": null,
      "suggestions": [],
      "is_extra": true
    }
  ],
  "unmatched_planned_items": [
    {"id": 15, "name": "mléko"},
    {"id": 18, "name": "chleba"}
  ]
}
```

**Notes on the response:**
- `is_extra: true` — receipt item has NO planned counterpart (impulse buy, bag fee, etc.). Derived from: has 0 suggestions AND was not matched by alias lookup.
- `unmatched_planned_items` — planned items NOT matched by ANY receipt item. These remain on the list; user may still need to buy them on another trip or at another store.
- A receipt item with `suggestions` but no auto-match (`match_status: "unmatched"`) is NOT extra — it has potential matches but needs user confirmation.
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

**Section 3: Extra items — not on your list (blue)**
- Receipt items with zero match suggestions (impulse buys, bag fees, etc.)
- Each has: "Add to list" (creates new planned item, marks bought) or "Dismiss" (ignore)
- User can also manually match to an unmatched planned item from Section 4

**Section 4: Unbought planned items (grey)**
- Planned items that weren't matched by any receipt item
- Shows as "not bought yet" — these remain on the list as-is
- User can: leave as-is, manually match to a receipt item from Section 3, or mark bought manually

**Actions:**
- "Confirm All" button — accepts all auto-matches, adds unmatched receipt items as new list items (marked bought), leaves unbought planned items unchanged
- Individual confirm/change per item
- "Match Manually" — select a receipt item, then select a planned item (works across sections 2/3/4)
- "Dismiss" — mark a receipt item as not relevant (bag fees, etc.)

### Modified: `ReceiptUploadModal.jsx`

After upload, if response status is `"pending_review"`:
- Instead of auto-closing, transition to `ReceiptMatchModal`
- If `"parsed"` (all auto-matched), close as before

### Modified: `ListDetail.jsx`

- Add "Review Matches" button on receipt badge if receipt status is `"pending_review"`
- Show both names on item tooltip: "Receipt: selský jogurt 2% → Planned: jogurt"

---

## ItemAlias Usage for Fast List Creation

The alias table enables rich autocomplete when users build shopping lists.

### Scenario: User types "jogurt"

Query `ItemAlias WHERE LOWER(planned_name) LIKE '%jogurt%' AND family_id = ?` returns:

| Planned | Receipt Name | Shop | Last Price | Count | Last Used |
|---------|-------------|------|-----------|-------|-----------|
| jogurt | selský jogurt 2% | Lidl | 29.90 Kč | 5 | 2026-03-10 |
| jogurt | Activia jahoda 150g | Albert | 34.90 Kč | 2 | 2026-02-20 |
| jogurt | BIO jogurt bílý | Lidl | 42.00 Kč | 1 | 2026-01-15 |

**UI shows:**
```
jogurt
  ├─ selský jogurt 2%    Lidl   29.90 Kč  (5x, last: Mar 10)
  ├─ Activia jahoda 150g Albert 34.90 Kč  (2x, last: Feb 20)
  └─ BIO jogurt bílý     Lidl   42.00 Kč  (1x, last: Jan 15)
```

User can:
- **Add "jogurt" (generic)** — keeps it vague, matched at receipt time
- **Pick a specific variant** — adds that exact product name with pre-filled price

### Scenario: User types "sel"

Query also matches on `receipt_name`, so typing "sel" surfaces:
```
selský jogurt 2%  →  (planned as "jogurt")  Lidl  29.90 Kč
```

### API changes

`GET /api/family/frequent-items` should return alias data alongside frequency:
```json
{
  "items": [
    {
      "item_name": "jogurt",
      "frequency": 8,
      "last_price": 29.90,
      "variants": [
        {"receipt_name": "selský jogurt 2%", "shop_name": "Lidl", "last_price": 29.90, "count": 5, "last_used": "2026-03-10"},
        {"receipt_name": "Activia jahoda 150g", "shop_name": "Albert", "last_price": 34.90, "count": 2, "last_used": "2026-02-20"},
        {"receipt_name": "BIO jogurt bílý", "shop_name": "Lidl", "last_price": 42.00, "count": 1, "last_used": "2026-01-15"}
      ]
    }
  ]
}
```

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
