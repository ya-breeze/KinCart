## 1. Backend — model & history

- [x] 1.1 Add `Unit string` and `CategoryID *uuid.UUID` to `ItemAlias` in `internal/models/models.go`
- [x] 1.2 Extend `upsertItemAlias` to accept and persist unit + category (latest wins; keep existing non-empty value when the new one is empty)
- [x] 1.3 Update all `upsertItemAlias` call sites in `receipt_service.go` to pass the item's unit and category

## 2. Backend — resolver

- [x] 2.1 Implement `resolveItemDefaults(ctx, tx, familyID, name, shopID)` returning `(unit, categoryID)`: alias history first (unit shop-preferred → most common; category most-recent), using the Go-lowercased name index. Add a batch form `resolveItemDefaultsBatch(..., names []string, ...)` that derives defaults from already-loaded aliases, for the paste and bulk paths
- [x] 2.2 Add the Gemini fallback **for the paste-preview and receipt paths only**: give the prompt the family's existing category names and constrain the returned `category` to one of them (or none — never invented); extend the shopping-list parse schema with that optional per-item `category`; add a minimal single-item categorize helper for the receipt path; guard on `gemini != nil`
- [x] 2.3 Resolve the AI-returned category name to a `Category` row using Go-side lowercasing (never SQL `LOWER()`, which misses Cyrillic/Czech names); unmatched/none → leave uncategorized
- [x] 2.4 Unit tests for the resolver: shop-preferred unit, cross-shop fallback, category-from-history (most-recent, tie-broken by frequency), AI category constrained to family list, non-ASCII category name matched Go-side, empty when unknown, explicit value not overridden

## 3. Backend — wire into add paths

- [x] 3.1 Fix the alias lookup in `internal/handlers/items.go:597`: query the indexed `PlannedNameLower` column with Go-lowercased names instead of SQL `LOWER(planned_name)`, which only folds ASCII (CLAUDE.md note 7). Pre-existing bug — the paste preview's price hint is already broken for Cyrillic/Czech names today
- [x] 3.2 Regression test for 3.1: a Cyrillic-named item with alias history gets its price hint (and now unit/category) in the paste preview. Note `receipt_service_test.go` deliberately uses ASCII names because of this same trap — the new test must use a non-ASCII name on purpose
- [x] 3.3 `ParseListText`: enrich each result with resolved unit/category from the already-loaded `byName` aliases (use `req.ShopID`, fall back to the list's `shop_id`). Do not call the per-item resolver inside the loop — it would reintroduce the N+1 that `items.go:585` avoids
- [ ] 3.4 `AddItemToList` and `BulkAddItems`: fill empty unit/category from alias history only, using the list's shop. **No Gemini call on these paths** — an unseen item keeps `pcs`/uncategorized
- [ ] 3.5 `receipt_service.go` (both new-item creation sites, ~665 and ~740): replace the first-category default with `resolveItemDefaults`
- [ ] 3.6 Tests: receipt-created item lands in the remembered category, not the first-by-sort-order one; manual add of an unseen item issues no AI call

## 4. Frontend

- [ ] 4.1 Surface the suggested unit and category in the paste preview (`PasteItemsPanel.jsx`) so the user can see/adjust before confirming
- [ ] 4.2 Ensure confirmed bulk-add carries unit + category_id through to creation
- [ ] 4.3 Error handling via `useToast`/`getApiError` for any new fetches

## 5. Verification

- [ ] 5.1 `make lint`, `make test-backend`, `make test-frontend` pass
- [ ] 5.2 E2E/manual: paste a known item → correct unit/category prefilled; upload a receipt with a known item → lands in the right category; unseen item with AI → sensible guess; with AI off → plain defaults, no error
