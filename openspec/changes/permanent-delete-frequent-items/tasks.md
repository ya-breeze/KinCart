## 1. Backend — Model & Migration

- [x] 1.1 Add `IsHidden bool` field (`gorm:"default:false"`) to `ItemFrequency` in `backend/internal/models/models.go`
- [x] 1.2 Verify GORM `AutoMigrate` in `database` package includes `ItemFrequency` so the column is added on next startup

## 2. Backend — Update Upsert Callsites

- [x] 2.1 In `handlers/items.go` single-item creation: change lookup to `WHERE family_id = ? AND LOWER(item_name) = LOWER(?)` and skip upsert if found row has `is_hidden = true`
- [x] 2.2 In `handlers/items.go` bulk-item creation: same `LOWER()` normalization and hidden-skip
- [x] 2.3 In `services/receipt_service.go` `updateItemFrequency`: add hidden-skip (lookup already uses `LOWER()`)

## 3. Backend — Update Delete Endpoint

- [x] 3.1 In `handlers/items.go` `DeleteFrequentItem`: replace `database.DB.Delete(&freq)` with `database.DB.Model(&freq).Update("is_hidden", true)`

## 4. Backend — New Endpoints

- [x] 4.1 In `handlers/items.go`: add `GetHiddenFrequentItems` handler — queries `WHERE family_id = ? AND is_hidden = true`, returns same shape as frequent items list
- [x] 4.2 In `handlers/items.go`: add `RestoreFrequentItem` handler — `PATCH /api/family/frequent-items/:id/restore` sets `is_hidden = false`
- [x] 4.3 Register both new routes in the router (same auth middleware group as existing frequent-items routes)

## 5. Backend — Update Display Query

- [x] 5.1 In `handlers/items.go` `GetFrequentItems`: add `.Where("is_hidden = ?", false)` to the fetch query

## 6. Frontend — New FrequentItemsPage

- [x] 6.1 Create `frontend/src/pages/FrequentItemsPage.jsx` — fetches `GET /api/family/frequent-items/hidden`, lists hidden items, each with a "Restore" button that calls `PATCH .../restore`
- [x] 6.2 Add empty-state message when no hidden items exist
- [x] 6.3 Add route `/settings/frequent-items` in `App.jsx` pointing to `FrequentItemsPage`
- [x] 6.4 Add a "Frequent Items" navigation link in `SettingsPage.jsx` alongside the existing Aliases and Flyer Stats links

## 7. Testing

- [x] 7.1 Run existing backend tests: `make test` in `backend/` — confirm no regressions
- [x] 7.2 Run existing E2E tests against WIP stack: `BASE_URL=<wip-url> npx playwright test --reporter=line` in `e2e/`
- [ ] 7.3 Manual smoke test: delete a chip → add same item → confirm chip does not reappear
- [ ] 7.4 Manual smoke test: navigate to Settings → Frequent Items → restore a hidden item → confirm chip reappears
