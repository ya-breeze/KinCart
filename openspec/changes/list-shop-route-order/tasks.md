## 1. Backend — model & persistence

- [x] 1.1 Add `ShopID *uuid.UUID` (`gorm:"type:uuid"`, `json:"shop_id"`) to `ShoppingList` in `internal/models/models.go`
- [x] 1.2 In `CreateList`, validate a non-null `shop_id` belongs to the family (`Where("id = ? AND family_id = ?")`), return 400 on mismatch, then persist
- [x] 1.3 In `UpdateList`, validate a non-null `shop_id` belongs to the family; ensure `shop_id: null` clears the association (tenant id/family_id preservation stays intact)
- [x] 1.4 Confirm `GetList`/`GetLists` return `shop_id` (no code change expected — verify serialization)

## 2. Backend — tests

- [x] 2.1 Add handler test: create list with a valid `shop_id` persists it; response includes it
- [x] 2.2 Add handler test: create/update with a `shop_id` from another family is rejected (400) and not persisted
- [x] 2.3 Add handler test: `PATCH {shop_id: null}` clears the association **and** leaves `title`/`status` unchanged (guards the full-`Save` path from clobbering unsent fields)

## 3. Frontend — create dialog

- [x] 3.1 In `Dashboard.jsx`, fetch shops and add a shop `<select>` to the create-list dialog with a default "No shop / default order" option
- [x] 3.2 Include `shop_id` (or omit/null when unset) in the create-list POST body
- [x] 3.3 Use the `useToast`/`getApiError` error pattern for the shops fetch and create call

## 4. Frontend — list detail

- [x] 4.1 In `ListDetail.jsx`, initialize `selectedShopId` from `list.shop_id` after the list loads and fetch that shop's order
- [x] 4.2 In `handleShopChange`, PATCH the list with the new `shop_id` (for both manager and shopper) in addition to reordering the local view; handle errors via toast
- [x] 4.3 Verify `getSortedCategoryIds()` produces the shop's route order automatically on load and falls back to default when no shop / no order

## 5. Verification

- [x] 5.1 `make lint` and `make test-backend` pass
- [ ] 5.2 Manual/E2E check: manager creates a list with a shop; shopper opens it and sees shop-ordered categories without picking a shop; a list with no shop is unchanged
