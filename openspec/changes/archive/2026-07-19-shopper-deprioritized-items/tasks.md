## 1. Backend

- [x] 1.1 Add `IsAbsent bool` (`gorm:"default:false"`, `json:"is_absent"`) to `Item` in `internal/models/models.go`
- [x] 1.2 Confirm `PATCH /api/items/:id` persists `is_absent` via the existing map-based `Updates` (add a handler test)
- [x] 1.3 Ensure `is_absent` is returned in list/item responses (serialization check)
- [x] 1.4 Add the exclusivity guard to `UpdateItem` in `internal/handlers/items.go`, before the `Updates` call: if the patch sets `is_bought: true`, force `is_absent: false` into the same map; if the item is already bought and the patch sets `is_absent: true`, return 400
- [x] 1.5 Handler tests for both transitions: marking an absent item bought clears `is_absent`; marking a bought item absent returns 400 and leaves the row unchanged
- [x] 1.6 In `internal/services/receipt_service.go`, clear `IsAbsent` on the two sites that mark an **existing** item bought — auto-match (`:455`) and manual match (`:642`). The `IsBought: true` struct literals at `:671`/`:755` create new items (receipt extras) whose `IsAbsent` is already false; leave them alone. Leave the unmatch path (`:624`) setting only `IsBought = false`
- [x] 1.7 Receipt-service test: an absent item matched by a receipt ends up bought and not absent (use ASCII item names — see CLAUDE.md note 7 on the `LOWER(receipt_name)` alias lookup)

## 2. Frontend — absent action

- [x] 2.1 Add `toggleAbsent(item)` in `ListDetail.jsx` → `PATCH {is_absent: !item.is_absent}` with `useToast`/`getApiError`
- [x] 2.2 Add a "not available" control next to the check-off on active shopper items; do not render it on bought items (the backend rejects that transition)
- [x] 2.3 In the done section, show whether each item was bought or absent, with an undo control
- [x] 2.4 In the done section, absent rows also offer a direct "Bought" control ("found it after all"); bought rows keep Undo alone

## 3. Frontend — done section

- [x] 3.1 Partition shopper items into active (`!is_bought && !is_absent`) and done (`is_bought || is_absent`)
- [x] 3.2 Keep active items grouped by category/route order; render done items in one collapsed section at the bottom with a "N done" count
- [x] 3.3 Hide the done section entirely when there are no done items
- [x] 3.4 Update the progress bar to count `is_bought || is_absent`; keep the estimated total summing only actual spend
- [x] 3.5 Leave the manager view grouping and ordering unchanged

## 4. Frontend — manager badge

- [x] 4.1 Render a "not found" badge on absent items in the manager view, in place within their category group (vanilla CSS per `.cursorrules`)
- [x] 4.2 Frontend test: manager view badges an absent item, does not badge a bought one, and drops the badge once the item is bought

## 5. Verification

- [x] 5.1 `make lint`, `make test-backend`, `make test-frontend` pass
- [x] 5.2 E2E/manual: shopper marks items bought and absent → both sink to the collapsed done section; unmarking returns an item to its category; manager view grouping is unchanged but absent items are badged
- [x] 5.3 E2E/manual: mark an item absent, then mark it bought → it shows as bought and the manager badge is gone
