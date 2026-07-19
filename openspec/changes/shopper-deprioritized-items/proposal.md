## Why

While shopping, bought items stay in place among the active ones and clutter the list, and there is no way to record that an item was out of stock. The shopper has to visually skip over items that are already handled, which slows down the in-store flow.

## What Changes

- Add an "absent" (out of stock) state to an item, distinct from "bought".
- In the shopper view, move both bought and absent items out of the active category groups into a single collapsed "done" section at the very bottom of the list.
- The shopper can mark an item absent (and undo it) with a control next to the existing check-off.
- Active items remain grouped by category/route order at the top; the "done" section is collapsed by default and expandable.
- Bought vs absent items remain visually distinguishable within the done section.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `items`: Items gain an `is_absent` state and a shopper action to set/clear it; the shopper-view grouping moves bought and absent items into a bottom "done" section.

## Impact

- **Backend model:** `Item` gains `IsAbsent bool` (default false). `UpdateItem` already applies partial map updates, so `PATCH {is_absent}` needs no handler change beyond the model column.
- **Frontend:** `ListDetail.jsx` shopper view partitions items into active vs done (`is_bought || is_absent`); active items keep category grouping, done items render in one collapsed section at the bottom. Add a "mark absent"/"unmark" control (`toggleAbsent`) beside the check-off, with the `useToast`/`getApiError` error pattern.
- **Progress bar:** Decide how absent items count toward completion (see design).
- **No API surface change** beyond the new field on item read/update.
