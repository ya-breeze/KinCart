## 1. Backend

- [ ] 1.1 Add `IsAbsent bool` (`gorm:"default:false"`, `json:"is_absent"`) to `Item` in `internal/models/models.go`
- [ ] 1.2 Confirm `PATCH /api/items/:id` persists `is_absent` via the existing map-based `Updates` (add a handler test)
- [ ] 1.3 Ensure `is_absent` is returned in list/item responses (serialization check)

## 2. Frontend — absent action

- [ ] 2.1 Add `toggleAbsent(item)` in `ListDetail.jsx` → `PATCH {is_absent: !item.is_absent}` with `useToast`/`getApiError`
- [ ] 2.2 Add a "not available" control next to the check-off on active shopper items
- [ ] 2.3 In the done section, show whether each item was bought or absent, with an undo control

## 3. Frontend — done section

- [ ] 3.1 Partition shopper items into active (`!is_bought && !is_absent`) and done (`is_bought || is_absent`)
- [ ] 3.2 Keep active items grouped by category/route order; render done items in one collapsed section at the bottom with a "N done" count
- [ ] 3.3 Hide the done section entirely when there are no done items
- [ ] 3.4 Update the progress bar to count `is_bought || is_absent`; keep the estimated total summing only actual spend
- [ ] 3.5 Leave the manager view grouping unchanged

## 4. Verification

- [ ] 4.1 `make lint`, `make test-backend`, `make test-frontend` pass
- [ ] 4.2 E2E/manual: shopper marks items bought and absent → both sink to the collapsed done section; unmarking returns an item to its category; manager view is unchanged
