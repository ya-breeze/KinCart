## Why

Every item is added with `Unit: "pcs"` and no category, so items constantly land in "Uncategorized" and single-vs-pack has to be fixed by hand. This is worst for receipt-matched/created items, which get dumped into the first category by sort order. The app already records per-`(family, planned name, shop)` purchase history in `ItemAlias` but stores neither the unit nor the category it was last bought as, so nothing is learned.

## What Changes

- Remember the **unit** and **category** an item was last used as, keyed by item name (and shop, for unit), on `ItemAlias`.
- When an item is added, prefill its **unit** and **category** from history:
  - Unit is resolved per-shop first (using the list's shop from the `list-shop-route-order` change), falling back to the most common unit for that name across shops.
  - Category is resolved by item name (category is shop-independent).
- When there is no history for an item — on the paste-preview and receipt paths only — ask Gemini for a common-sense unit and a category **chosen from the family's own category list** (the prompt is given the family's category names and must return one of them or none — never a free-text/English guess, since families name categories in their own language). Leave defaults (`pcs` / uncategorized) if Gemini is unavailable or returns nothing usable.
- Apply the inference in the paste-to-list preview and to receipt-created items so they no longer default into the first/"Uncategorized" category.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `items`: Items gain history-driven unit and category defaults on add, with an AI common-sense fallback.
- `paste-to-list`: The parse preview enriches each parsed item with a remembered/inferred unit and category, not just a price hint.
- `receipts`: Receipt-created and receipt-matched items derive their category from history/AI instead of the first-by-sort-order default.

## Impact

- **Backend model:** `ItemAlias` gains `Unit string` and `CategoryID *uuid.UUID`. `upsertItemAlias` records them from the matched/receipt item. GORM auto-migrate (nullable/defaulted columns).
- **Backend logic:** A shared `resolveItemDefaults(familyID, name, shopID)` helper returns `(unit, categoryID)` using alias history then Gemini fallback, plus a batch form for the paste path. Wired into `ParseListText` (preview enrichment) and `receipt_service.go` new-item creation (replacing the first-category default at lines ~665 and ~740). `AddItemToList`/`BulkAddItems` use history only, with no AI call on the synchronous add path.
- **Bug fix carried here:** the alias lookup in `ParseListText` currently uses SQL `LOWER(planned_name)` instead of the indexed `PlannedNameLower` column, breaking it for non-ASCII names; corrected as part of this change since the new resolution reads through it.
- **AI:** Extend the Gemini shopping-list schema (and add a lightweight categorize path) to return a suggested unit and category name; map the category name to an existing family `Category` case-insensitively.
- **Frontend:** Paste preview shows the suggested unit/category (already renders variants/price); no structural change beyond surfacing the new fields. Receipt-created items simply appear in the right category.
- **Dependency:** Best combined after `list-shop-route-order` so per-shop unit resolution can use the list's shop; degrades gracefully if that shop is absent.
