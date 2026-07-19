## Context

Items are created with `Unit` defaulting to `pcs` and `CategoryID` unset. Receipt-created items grab the first family category by `sort_order` (`receipt_service.go:660-664`, `734-737`) — effectively "Uncategorized". The app already maintains per-`(family, planned name, shop)` history in `ItemAlias` (with `LastPrice`, `PurchaseCount`, `ShopID`), upserted by `upsertItemAlias` on every receipt match/create, and `ParseListText` already batch-loads aliases to enrich the paste preview with prices. But `ItemAlias` records no unit and no category, so unit/category can't be learned.

## Goals / Non-Goals

**Goals:**
- Learn the unit and category an item was last used as, and prefill them on add.
- Resolve unit per-shop (using the list's shop) with a cross-shop fallback; resolve category by name.
- Fall back to a common-sense AI guess for never-seen items; degrade cleanly without AI.

**Non-Goals:**
- No new user-facing settings for defaults; inference is automatic and always overridable by an explicit value.
- No re-categorization of existing items already on lists.
- No change to how categories themselves are created/ordered.

## Decisions

- **Store history on `ItemAlias`, not a new table.** Add `Unit string` and `CategoryID *uuid.UUID`. Aliases already key per family/name/shop and are already loaded in the hot paths, so this is the lowest-friction home. `upsertItemAlias` gains unit/category params and writes them (latest wins; keep an existing non-empty value if the new one is empty). Alternative considered: a dedicated `item_default` table — rejected as duplicative of alias history.

- **Category is shop-independent; unit is per-shop.** Category resolution ignores shop (take the most recent/most frequent category recorded for the name). Unit resolution prefers the alias whose `ShopID` matches the list's shop, else the most common unit across that name's aliases. This matches the real-world pattern ("yogurt = a pack at Makro").

- **One shared resolver.** Add `resolveItemDefaults(ctx, tx, familyID, name string, shopID *uuid.UUID) (unit string, categoryID *uuid.UUID)`:
  1. Load aliases for the name; if found, derive unit (shop-preferred → most common) and category (most recent non-null).
  2. Else, if a Gemini client is available, request a common-sense `{unit, category}` and map the category string to an existing family `Category` by case-insensitive name; unmatched category names are dropped.
  3. Else return empty → caller keeps `pcs`/uncategorized.
  Callers only fill fields the request left empty, so an explicit user choice always wins.

- **Wire points.** (a) `ParseListText` — enrich each `parsedItemResult` with resolved unit/category (using `req.ShopID`, and the target list's `shop_id` as a fallback) so the preview and the confirmed bulk-add carry them. (b) `receipt_service.go` new-item creation — replace the first-category default with `resolveItemDefaults`. `AddItemToList`/`BulkAddItems` fill empty unit/category via the resolver using the list's shop.

- **Gemini category path.** Extend the shopping-list parse schema with an optional `category` per item (cheap — same call already parses the text), and add a minimal categorize helper for the single-item/receipt path. Category names are matched to existing family categories only; the feature never creates categories.

## Risks / Trade-offs

- [AI latency/cost on the receipt path] → Only call Gemini when history misses; batch within the existing parse call for paste. Receipt processing already runs in a background ticker, so an extra call there is acceptable and guarded by `s.gemini != nil`.
- [Gemini returns a category that doesn't exist for the family] → Drop it (leave uncategorized) rather than inventing categories.
- [Wrong remembered unit annoys the user] → Always overridable inline; latest purchase updates the memory, so it self-corrects.
- [ASCII-only `LOWER()` pitfall for Cyrillic names] → Reuse the existing Go-lowercased `PlannedNameLower` index used elsewhere for alias lookups rather than SQL `LOWER()`.

## Migration Plan

- GORM auto-migrate adds `unit` and `category_id` to `item_aliases` (nullable/empty defaults). Existing aliases have empty unit/null category and simply provide no hint until next purchase refreshes them. Rollback-safe: new columns are ignored by prior code.

## Open Questions

- Should category memory prefer "most recent" or "most frequent" when a name was filed under different categories over time? Default: most recent, tie-broken by frequency. (Confirm during review.)
