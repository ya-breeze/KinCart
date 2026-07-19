## 1. Backend — model & history

- [ ] 1.1 Add `Unit string` and `CategoryID *uuid.UUID` to `ItemAlias` in `internal/models/models.go`
- [ ] 1.2 Extend `upsertItemAlias` to accept and persist unit + category (latest wins; keep existing non-empty value when the new one is empty)
- [ ] 1.3 Update all `upsertItemAlias` call sites in `receipt_service.go` to pass the item's unit and category

## 2. Backend — resolver

- [ ] 2.1 Implement `resolveItemDefaults(ctx, tx, familyID, name, shopID)` returning `(unit, categoryID)`: alias history first (unit shop-preferred → most common; category most-recent), using the Go-lowercased name index
- [ ] 2.2 Add the Gemini fallback: extend the shopping-list parse schema with an optional per-item `category`; add a minimal single-item categorize helper; map category names to existing family categories case-insensitively (never create categories); guard on `gemini != nil`
- [ ] 2.3 Unit tests for the resolver: shop-preferred unit, cross-shop fallback, category-from-history, empty when unknown, explicit value not overridden

## 3. Backend — wire into add paths

- [ ] 3.1 `ParseListText`: enrich each result with resolved unit/category (use `req.ShopID`, fall back to the list's `shop_id`)
- [ ] 3.2 `AddItemToList` and `BulkAddItems`: fill empty unit/category via the resolver using the list's shop
- [ ] 3.3 `receipt_service.go` (both new-item creation sites): replace the first-category default with `resolveItemDefaults`
- [ ] 3.4 Tests: receipt-created item lands in the remembered category, not the first-by-sort-order one

## 4. Frontend

- [ ] 4.1 Surface the suggested unit and category in the paste preview (`PasteItemsPanel.jsx`) so the user can see/adjust before confirming
- [ ] 4.2 Ensure confirmed bulk-add carries unit + category_id through to creation
- [ ] 4.3 Error handling via `useToast`/`getApiError` for any new fetches

## 5. Verification

- [ ] 5.1 `make lint`, `make test-backend`, `make test-frontend` pass
- [ ] 5.2 E2E/manual: paste a known item → correct unit/category prefilled; upload a receipt with a known item → lands in the right category; unseen item with AI → sensible guess; with AI off → plain defaults, no error
