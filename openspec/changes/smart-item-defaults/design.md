## Context

Items are created with `Unit` defaulting to `pcs` and `CategoryID` unset. Receipt-created items grab the first family category by `sort_order` (`receipt_service.go:665-669`, `740-743`) — effectively "Uncategorized". The app already maintains per-`(family, planned name, shop)` history in `ItemAlias` (with `LastPrice`, `PurchaseCount`, `ShopID`), upserted by `upsertItemAlias` on every receipt match/create (**three** call sites: `receipt_service.go:655`, `688`, `770`), and `ParseListText` already batch-loads aliases to enrich the paste preview with prices. But `ItemAlias` records no unit and no category, so unit/category can't be learned.

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

- **Category is shop-independent; unit is per-shop.** Category resolution ignores shop and takes the **most recent** category recorded for the name (tie-broken by frequency). Unit resolution prefers the alias whose `ShopID` matches the list's shop, else the most common unit across that name's aliases. This matches the real-world pattern ("yogurt = a pack at Makro").

- **One shared resolver, in two shapes.** Add `resolveItemDefaults(ctx, tx, familyID, name string, shopID *uuid.UUID) (unit string, categoryID *uuid.UUID)` for single-item callers (receipt path, `AddItemToList`), plus a batch form `resolveItemDefaultsBatch(ctx, tx, familyID, names []string, shopID *uuid.UUID)` for `ParseListText` and `BulkAddItems`. `ParseListText` already batch-loads every parsed name's aliases into its `byName` map with an explicit `// avoids N+1 queries` comment (`handlers/items.go:585`); the per-item resolver must **not** be called in that loop or it undoes that. The batch form derives defaults from the already-loaded aliases rather than re-querying.
  1. Load aliases for the name; if found, derive unit (shop-preferred → most common) and category (most recent non-null).
  2. Else, if a Gemini client is available, request a common-sense unit and a category **chosen from the family's own category list** (the prompt is given the family's existing category names and must return one of them or none — it never invents a name). The returned name is matched back to the `Category` row using **Go-side lowercasing** (see the `LOWER()` note below), not a free-text English guess.
  3. Else return empty → caller keeps `pcs`/uncategorized.
  Callers only fill fields the request left empty, so an explicit user choice always wins.

- **Wire points, and where AI is allowed.** (a) `ParseListText` — enrich each `parsedItemResult` with resolved unit/category (using `req.ShopID`, and the target list's `shop_id` as a fallback) so the preview and the confirmed bulk-add carry them. (b) `receipt_service.go` new-item creation — replace the first-category default with `resolveItemDefaults`. (c) `AddItemToList`/`BulkAddItems` fill empty unit/category from **alias history only — no Gemini call**.

  The AI fallback is confined to (a) and (b), where a call is already in flight or already asynchronous: paste batches the category into the existing `ParseShoppingText` call, and receipt processing runs in a background ticker. `AddItemToList` is a synchronous single-item add and `BulkAddItems` would need one call per unseen name, so neither may block on Gemini — an unseen item there simply keeps `pcs`/uncategorized. This keeps manual add latency unchanged.

- **Gemini category path — constrained choice, localized.** Families create their own categories and none are seeded, so a family's category names are frequently **non-English** (e.g. Czech/Russian). A free-text English guess would almost never match. Therefore the prompt is given the family's **actual category names** and must pick one of them (or return none). Extend the shopping-list parse schema with an optional `category` per item constrained to that list (cheap — same call already parses the text), and add a minimal categorize helper for the single-item/receipt path. The returned name is resolved to its `Category` via Go-lowercased comparison; unmatched/none → left uncategorized. The feature never creates categories.

## Risks / Trade-offs

- [AI latency/cost] → Only call Gemini when history misses, and only on the two paths that can absorb it: batched into the existing parse call for paste, and on the receipt path (guarded by `s.gemini != nil`). The synchronous add paths (`AddItemToList`/`BulkAddItems`) never call Gemini, so a manual add is no slower than today.
- [**Correction found in implementation** — receipt item creation is not in the background ticker] → The design first assumed new receipt items are created inside the background `ProcessReceipt` ticker. They are not: creation happens in `ConfirmMatch` / `ConfirmAllMatches`, which are user-triggered, and both sit inside a `s.db.Transaction`. An AI (or even history) call inside that transaction would hold the SQLite write lock across a network round-trip. So the receipt resolver (`resolveNewReceiptItemDefaults`) is called **before** the transaction opens, and its result is passed in — the transaction does no network I/O. For `ConfirmAllMatches` this means pre-resolving every unmatched item up front (one map, keyed by receipt-item ID). Known trade-off: confirming a receipt with many never-seen items makes one AI call per item before the write; acceptable because the receipt-review flow tolerates a brief pause and the calls are outside the lock, but a candidate for batching later if it bites.
- [Pre-existing SQL `LOWER()` bug in the very query this change rides on] → `handlers/items.go:597` looks aliases up with `LOWER(planned_name)` instead of the indexed `PlannedNameLower` column, so the paste preview's price hint is **already** broken for Cyrillic/Czech names (same ASCII-folding trap documented in CLAUDE.md note 7). Fixed here rather than deferred, because the new unit/category resolution reads through that same query and would otherwise ship silently broken in exactly the languages this feature's localized-category design exists to serve.
- [AI-sourced categories enter history] → Accepted. A receipt confirm writes the item's category to its alias with "latest wins" regardless of whether that category originated from history, the user, or an AI guess. A wrong guess therefore persists until a later purchase overwrites it; the user can always correct the item inline before confirming. Rejected alternative: tracking a `source` on the alias and ranking user-confirmed memories above AI ones — not worth the extra column and resolver ranking rules at this stage.
- [Gemini returns a category outside the family's list] → The prompt constrains it to the family's own names; any unmatched/none result leaves the item uncategorized rather than inventing a category.
- [Wrong remembered unit annoys the user] → Always overridable inline; latest purchase updates the memory, so it self-corrects.
- [**Found in review** — auto-matched purchases were not recording history] → `applyItemMatches` (the high-confidence auto-match path in `ProcessReceipt`) originally called only `updateItemFrequency`, never `upsertItemAlias`. So once an item began auto-matching, its remembered unit/category froze and the "latest purchase self-corrects" property above did not actually hold for the common steady state. Fixed: the auto-match branch now upserts the alias with the planned item's unit/category too, mirroring the manual match path. This also starts incrementing the alias `PurchaseCount` on auto-match (it previously did not), which is strictly more faithful to real purchase frequency for variant ordering and the unit tie-break. `shopID` is threaded into `applyItemMatches` to record the per-shop alias.
- [Gemini hanging the user's receipt-confirm request] → The single-item categorize call on the receipt path is bounded by `geminiCategorizeTimeout` (10s) via `context.WithTimeout`. On deadline it hits the same error branch as any AI failure and degrades to uncategorized, so a slow Gemini cannot hang the confirm.
- [**Found in review** — a remembered category can be deleted out from under the alias] → `ItemAlias.CategoryID` outlived its `Category`: `DeleteCategory` cleared the id from `items` but not from `item_aliases`, so the resolver would prefill a dangling id — making a valid add fail `validateItemsFamily` with a 400, or saving a receipt item pointing at a non-existent category. Defence in depth: (1) `DeleteCategory` now nulls `item_aliases.category_id` for the family (source); (2) `ResolveItemDefaultsBatch` drops any resolved category id with no live `Category` row (one extra query, only when a category resolved) — covering the add and receipt paths and any rows already dangling; (3) `enrichParsedItem` surfaces a suggested category only when it maps to a live category, covering the pure paste path that bypasses the batch resolver.
- [ASCII-only `LOWER()` pitfall for Cyrillic names] → Applies to **both** the alias name lookup **and** matching the AI-returned category name to a `Category` row. Use Go-side lowercasing (as the existing `PlannedNameLower` index does) — never SQL `LOWER()`, which only folds ASCII and would silently miss Cyrillic/Czech category names.

## Migration Plan

- GORM auto-migrate adds `unit` and `category_id` to `item_aliases` (nullable/empty defaults). Existing aliases have empty unit/null category and simply provide no hint until next purchase refreshes them. Rollback-safe: new columns are ignored by prior code.

## Open Questions

- _(Resolved in review)_ Category memory uses **most recent**, tie-broken by frequency, when a name was filed under different categories over time.
