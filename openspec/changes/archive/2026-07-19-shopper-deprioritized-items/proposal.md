## Why

While shopping, bought items stay in place among the active ones and clutter the list, and there is no way to record that an item was out of stock. The shopper has to visually skip over items that are already handled, which slows down the in-store flow.

## What Changes

- Add an "absent" (out of stock) state to an item, mutually exclusive with "bought" — bought wins, so marking an absent item bought clears absent, and a bought item cannot be marked absent.
- In the shopper view, move both bought and absent items out of the active category groups into a single collapsed "done" section at the very bottom of the list.
- The shopper can mark an item absent (and undo it) with a control next to the existing check-off.
- Active items remain grouped by category/route order at the top; the "done" section is collapsed by default and expandable.
- Bought vs absent items remain visually distinguishable within the done section.
- In the manager view, absent items carry a "not found" badge so the manager can see what the shopper missed.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `items`: Items gain an `is_absent` state, mutually exclusive with `is_bought`, and a shopper action to set/clear it; the shopper-view grouping moves bought and absent items into a bottom "done" section; the manager view badges absent items.

## Impact

- **Backend model:** `Item` gains `IsAbsent bool` (default false).
- **Backend handler:** `UpdateItem` gains a guard enforcing exclusivity — setting `is_bought: true` also clears `is_absent`; setting `is_absent: true` on an already-bought item is rejected with 400. This is the handler's first piece of field-level logic; it is currently a blind map `Updates`.
- **Receipt service:** the sites that set `IsBought = true` also clear `IsAbsent`, so a receipt match resolves an item the shopper had marked missing.
- **Frontend (shopper):** `ListDetail.jsx` partitions items into active vs done (`is_bought || is_absent`); active items keep category grouping, done items render in one collapsed section at the bottom. Add a "mark absent"/"unmark" control (`toggleAbsent`) beside the check-off, with the `useToast`/`getApiError` error pattern, hidden on bought items.
- **Frontend (manager):** absent items render a "not found" badge in place; grouping and ordering unchanged.
- **Progress bar:** counts `is_bought || is_absent`; exclusivity means no double-counting.
- **No API surface change** beyond the new field on item read/update.
