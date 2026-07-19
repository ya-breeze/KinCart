## Why

A shopping list has no associated shop, so the shopper must pick a shop from a dropdown every time they open a list to get the aisle-optimized category order. The manager's intended store is never captured, and the choice is lost on reload — the list falls back to the default category order and the "route optimization" feature goes unused in practice.

## What Changes

- Persist an optional shop on each shopping list (`ShoppingList.shop_id`, nullable).
- The create-list dialog gains an optional shop selector ("No shop / default order" is the default).
- The list detail lets the manager change the list's shop; the change is saved to the list.
- The shopper view initializes its shop selector from `list.shop_id` and auto-sorts the list by that shop's saved category route order on open — no manual selection needed.
- Any shop change (manager or shopper) persists back onto the list.
- Fallback is unchanged: a list with no shop, or a shop with no configured route order, uses the default `Category.SortOrder`.
- Reuses the existing per-shop route order (`ShopCategoryOrder` + `GET/PATCH /api/shops/:id/order`); no changes to that endpoint.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `lists`: A list gains an optional shop association that is persisted and returned; category ordering in the list view is driven by the list's shop.

## Impact

- **Backend:** `ShoppingList` model gains `ShopID *uuid.UUID` (nullable, GORM auto-migrate). `CreateList` and `UpdateList` accept, validate (shop must belong to the family), and persist `shop_id`. `GetList`/`GetLists` already return the full model, so `shop_id` is included automatically.
- **Frontend:** `Dashboard.jsx` create-list dialog gains a shop `<select>` and sends `shop_id`. `ListDetail.jsx` initializes `selectedShopId` from `list.shop_id`, and `handleShopChange` PATCHes the list (both manager and shopper) in addition to reordering the view.
- **Data:** Existing lists get a null shop and behave exactly as today. No migration of existing data required.
- **APIs:** `POST /api/lists`, `PATCH /api/lists/:id` accept `shop_id`; list responses include `shop_id`.
