## Context

In the shopper view (`ListDetail.jsx`, from ~line 900), items are grouped by category via `finalSortedCatIds` and rendered per-group; bought items are only dimmed (`opacity: 0.6`, line-through) but stay in place. There is a single check-off control calling `toggleItem` → `PATCH /api/items/:id {is_bought}`. There is no notion of "out of stock". `UpdateItem` applies partial map updates, so a new boolean field is writable through the existing endpoint.

## Goals / Non-Goals

**Goals:**
- Add an `is_absent` state, mutually exclusive with `is_bought` (bought wins).
- In the shopper view, sink bought + absent items into one collapsed "done" section at the bottom.
- Make "not found" visible to the manager as a passive badge on the item.

**Non-Goals:**
- No per-category "done" sub-sections (product decision: one section at the very bottom).
- No history/analytics on absent items in this change.
- No change to how items are added or categorized.
- No proactive notification to the manager when an item is marked absent — the manager sees it
  when they look at the list. A push signal is a separate change with its own delivery model.

## Decisions

- **`IsAbsent bool` on `Item` (default false), mutually exclusive with `IsBought`; bought wins.**
  Two booleans plus an enforced invariant, rather than a single status enum. The enum models
  exclusivity structurally (invalid state unrepresentable) but rewrites every `IsBought` site in
  `receipt_service.go` (`:455`, `:624`, `:642`, `:671`, `:755`), the raw SQL `is_bought = false`
  filter at `:795`, and the check-off contract — a large blast radius across the riskiest code in
  the repo for a two-state flag. Rejected on that basis. "Done" = `is_bought || is_absent`;
  because the two are exclusive, this cannot double-count.

- **The invariant is enforced server-side, in one place.** `UpdateItem`
  (`backend/internal/handlers/items.go`) is today a blind `Updates(updateData)` map write with no
  field-level logic, so the guard is new code regardless of how the state is modelled:
  - If the patch sets `is_bought: true`, force `is_absent: false` into the same map — a single
    write, so no window exists where both are true.
  - If the item is already bought and the patch sets `is_absent: true`, reject with 400.
    Chosen over silently dropping the field, which would return 200 with a body that doesn't
    reflect what the client asked for. The frontend won't offer the control on a bought item, so
    a 400 only fires on a stale tab or a second client — precisely when a real error is wanted.

- **Receipt matching clears absent when it marks an existing item bought.** Two sites mutate an
  existing item: auto-match (`:455`) and manual match (`:642`). Both clear `IsAbsent`. This is the
  path the "shopper marked it missing, but the receipt proves they bought it" case actually
  travels. The other two `IsBought: true` writes (`:671`, `:755`) are struct literals creating new
  items for receipt extras — their `IsAbsent` is already the zero value, so they need no change.
  `:624` sets `IsBought = false` when unmatching and does **not** restore absent — the item
  returns to active. Absent was a shopper observation invalidated by the purchase; resurrecting
  it would assert something now known to be false.

- **Client-side partitioning in the shopper view only.** Split `list.items` into active (`!is_bought && !is_absent`) and done. Active items keep the current category grouping / route order. Done items render in one `<details>`-style collapsed section at the bottom showing "N done", regardless of category. Manager view code path is untouched.

- **Reuse the existing PATCH endpoint.** Add `toggleAbsent(item)` → `PATCH {is_absent: !item.is_absent}`, mirroring `toggleItem`, with `useToast`/`getApiError` handling. No new route or request shape — only the model column and the guard above.

- **Manager visibility is a passive badge.** Absent items render in the manager view with a
  distinct "not found" badge, in place, keeping their category group. The manager view already
  receives the full item payload, so `is_absent` arrives with no API change. The badge clears
  when the item is unmarked or later bought (the exclusivity rule guarantees a bought item never
  still carries it). Considered and deferred: a "missed items" summary at the top of the list —
  worth adding only if the in-place badge proves easy to miss in practice.

- **Absent control placement.** A small secondary control (e.g. a "not available" icon button) next to the check-off on each active item. In the done section, each item shows whether it was bought or absent and offers an undo.

- **Absent rows also offer a direct "bought" action.** The spec requires the shopper to be
  able to mark an absent item bought ("found it after all"), and the backend clears `is_absent`
  on that transition. Undo alone does not satisfy it: the shopper would have to un-mark absent,
  scroll back to find the item in its category group, then check it off — three interactions
  and a screen change for one real-world event. So an absent row in the done section carries
  both **Bought** (→ `toggleItem`, which the server resolves to bought-and-not-absent) and
  **Undo** (→ `toggleAbsent`). Bought rows keep Undo alone, since "un-buy" is the only sensible
  reversal there.

- **Progress bar treats absent as resolved.** The progress bar currently uses `is_bought`. Absent items are "handled" for the trip, so completion counts `is_bought || is_absent` in the shopper view; the estimated total still only sums actual/bought prices (absent items contribute no spend). This keeps the bar from being stuck at <100% when the only remaining items are out of stock.

## Risks / Trade-offs

- [Absent semantics leaking into manager planning] → `is_absent` is a shopper trip state; it drives a badge only, never manager grouping or ordering. If a list is re-shopped, the manager can unmark it, or it returns to plain active once the flag is cleared.
- [Two booleans can still represent "bought AND absent" in the database] → The type does not prevent it; three guards do: the `UpdateItem` guard, the receipt-service clear, and `enforceBoughtAbsentExclusivity` on the two creation paths (`AddItemToList`, `BulkAddItems`). The creation paths matter because they bind a full `models.Item` from JSON, so a client can name both flags in one payload and never touch `UpdateItem`'s guard at all — code review caught this after an earlier draft of this document wrongly claimed every write path was already covered. Any *future* write path must honour the invariant deliberately; that is the standing price of not taking the enum, and the reason these guards live in the handlers rather than the frontend.
- [Stale-client 400s] → A client holding a pre-bought view may PATCH `is_absent: true` and get a 400 it didn't expect. The frontend surfaces it through the standard `useToast`/`getApiError` path and refetches; no special handling.
- [Migration] → `is_absent` defaults false; existing items are unaffected. GORM auto-migrate adds the column. No backfill is needed — no existing row can violate the invariant, since none has `is_absent` set.
