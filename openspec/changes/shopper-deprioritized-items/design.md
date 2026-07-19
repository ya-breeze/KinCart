## Context

In the shopper view (`ListDetail.jsx`, from ~line 900), items are grouped by category via `finalSortedCatIds` and rendered per-group; bought items are only dimmed (`opacity: 0.6`, line-through) but stay in place. There is a single check-off control calling `toggleItem` → `PATCH /api/items/:id {is_bought}`. There is no notion of "out of stock". `UpdateItem` applies partial map updates, so a new boolean field is writable through the existing endpoint.

## Goals / Non-Goals

**Goals:**
- Add an `is_absent` state distinct from `is_bought`.
- In the shopper view, sink bought + absent items into one collapsed "done" section at the bottom.
- Keep the manager view unchanged.

**Non-Goals:**
- No per-category "done" sub-sections (product decision: one section at the very bottom).
- No history/analytics on absent items in this change.
- No change to how items are added or categorized.

## Decisions

- **`IsAbsent bool` on `Item` (default false), separate from `IsBought`.** Two independent booleans rather than a single status enum, because the existing check-off toggles `is_bought` and receipt matching sets `is_bought`; overloading a status field would ripple into those paths. "Done" = `is_bought || is_absent`. Alternative considered: `status` enum (`active|bought|absent`) — rejected to avoid touching receipt/check-off logic.

- **Client-side partitioning in the shopper view only.** Split `list.items` into active (`!is_bought && !is_absent`) and done. Active items keep the current category grouping / route order. Done items render in one `<details>`-style collapsed section at the bottom showing "N done", regardless of category. Manager view code path is untouched.

- **Reuse the existing PATCH endpoint.** Add `toggleAbsent(item)` → `PATCH {is_absent: !item.is_absent}`, mirroring `toggleItem`, with `useToast`/`getApiError` handling. No backend handler change needed beyond the model column.

- **Absent control placement.** A small secondary control (e.g. a "not available" icon button) next to the check-off on each active item. In the done section, each item shows whether it was bought or absent and offers an undo.

- **Progress bar treats absent as resolved.** The progress bar currently uses `is_bought`. Absent items are "handled" for the trip, so completion counts `is_bought || is_absent` in the shopper view; the estimated total still only sums actual/bought prices (absent items contribute no spend). This keeps the bar from being stuck at <100% when the only remaining items are out of stock.

## Risks / Trade-offs

- [Absent semantics leaking into manager planning] → `is_absent` is a shopper trip state; the manager view ignores it for grouping. If a list is re-shopped, the manager can unmark or the item simply shows as active again when neither flag is set.
- [Two booleans allow the "bought AND absent" combination] → Handled by treating any true as "done" and showing bought precedence in styling; the UI won't offer both simultaneously in the normal flow.
- [Migration] → `is_absent` defaults false; existing items are unaffected. GORM auto-migrate adds the column.

## Open Questions

- Should marking an item absent be manager-visible as a signal ("shopper couldn't find X")? Out of scope here; the field is stored and could drive a future notification.
