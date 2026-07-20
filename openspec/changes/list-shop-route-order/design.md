## Context

`ShoppingList` (backend/internal/models/models.go) has no shop field. The per-shop category route order already exists as `ShopCategoryOrder` and is served by `GET/PATCH /api/shops/:id/order`. In `ListDetail.jsx` the shop selector (`selectedShopId`) is transient local component state that defaults to `''`; `getSortedCategoryIds()` only applies a shop's order when a shop is actively selected, otherwise it falls back to `Category.SortOrder`. Because the selection is never persisted, the shopper must re-pick the shop on every visit, and the manager cannot pre-set the store.

Multi-tenancy is via kin-core `TenantModel`: handlers must scope every query by `family_id` (`familyID := c.MustGet("family_id").(uuid.UUID)`) and validate that any referenced foreign entity belongs to the family.

## Goals / Non-Goals

**Goals:**
- Persist an optional shop on a list so the shopper's list auto-sorts by that shop's route.
- Let the manager set the shop at creation and the shopper change it on the list detail.
- Keep the default behavior identical when no shop is set.

**Non-Goals:**
- No changes to how per-shop category order is configured (Settings / `ShopCategoryOrder`).
- No new sort logic beyond reusing the existing shop-order mapping in `getSortedCategoryIds()`.
- Not adding multiple shops per list.

## Decisions

- **Nullable `ShopID *uuid.UUID` on `ShoppingList`.** A pointer (not `uuid.UUID`) so "no shop" is a real null rather than the zero UUID, matching the existing `Receipt.ShopID *uuid.UUID` pattern. GORM auto-migrate adds the column; existing rows get null and behave as today. Alternative considered: a join table for future multi-shop — rejected as YAGNI.

- **Validate shop ownership on write.** `CreateList` and `UpdateList` validate that a non-null `shop_id` belongs to the family (same `Where("id = ? AND family_id = ?")` check used elsewhere) before persisting. An invalid shop is a `400`, consistent with `validateItemsFamily`.

- **`UpdateList` must accept clearing the shop.** Because `UpdateList` uses `ShouldBindJSON` into the loaded struct then `Save`, sending `shop_id: null` naturally clears it. We keep the existing tenant-field preservation logic (id/family_id are restored after bind). No partial-PATCH semantics needed — the frontend already sends the full list object on update.

- **Frontend: derive from `list.shop_id`, persist on change.** `ListDetail.jsx` derives the selected shop from `list.shop_id` and fetches that shop's order when it changes. `handleShopChange` PATCHes the list with the new `shop_id` and reverts the view if the request fails. Note `ListDetail` has two render trees (`if (isManager)` returns its own); the selector lives in the shopper tree only, so a manager-side control would need separate UI. `Dashboard.jsx` create dialog adds a shop `<select>` (options fetched from `/api/shops`, plus a "No shop / default order" empty option) and includes `shop_id` in the create body.

- **Reuse `getSortedCategoryIds()` unchanged in logic.** It already produces shop-ordered ids when `selectedShopId` + `shopOrder` are set, with unordered categories falling to the end (`|| 999`). Initializing `selectedShopId` from the persisted value is the only change needed to make it apply automatically.

## Risks / Trade-offs

- [A deleted shop leaves a dangling `shop_id` on lists] → The shop-order fetch returns empty, so the list falls back to default order (no crash). Optionally, `DeleteShop` could null out referencing lists, but current fallback already degrades gracefully; leave lists untouched to avoid extra write scope.
- [Shopper overwriting the manager's shop] → Explicitly accepted per product decision: any shop change persists. Simple and predictable.
- [Race: two users change the shop] → Last write wins, consistent with the rest of the app's list updates. No locking added.

## Migration Plan

- GORM auto-migrate adds the nullable column on backend startup (no manual `ALTER TABLE` needed since the column is nullable). Existing lists read back null and are unaffected. Rollback is safe: the column is ignored by the prior code.
