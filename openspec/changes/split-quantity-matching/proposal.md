## Why

A manager often plans one line with a quantity, e.g. "2 fruit kefirs", but the shopper buys two *different* variants (strawberry + peach). On the receipt these are two distinct line items. Today matching is effectively one-receipt-item-to-one-planned-item: the first variant consumes the planned item (link-alias even deletes it), so the second variant is forced to become an "extra"/duplicate. The planned quantity is ignored and the review breaks.

## What Changes

- Allow a single planned item with quantity N to be matched by **multiple** receipt items (up to N), each recording a variant, without creating duplicates or forcing extras.
- Stop deleting the planned item on link when quantity remains to be consumed; the planned item is retained and marked bought once at least one receipt item is matched to it.
- Record each distinct receipt variant as its own alias of the planned name (so future receipts recognize every variant).
- Reflect remaining capacity in the match modal so the manager/shopper can see a planned item can still absorb more receipt lines.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `receipts`: The match model allows many receipt items to map to one planned item up to its quantity; link/confirm no longer removes a planned item that still has unconsumed quantity; aliases are created per variant.

## Impact

- **Backend match model:** Relax the one-to-one assumption. `ReceiptItem.MatchedItemID` already points to a planned item, so many receipt items can reference the same planned item; the planned item's single `Item.ReceiptItemID` back-link becomes insufficient and is de-emphasized (kept as the first/primary link or ignored in favor of the `ReceiptItem → MatchedItemID` direction).
- **`LinkAlias` (`items.go`):** Do not delete the planned item while unconsumed quantity remains; only collapse/remove when fully consumed (see design for the exact rule). Still upsert the variant alias.
- **Confirm/auto-match (`receipt_service.go`):** Permit assigning another receipt item to an already-matched planned item until its quantity is filled; mark bought once ≥1 match; aggregate price across matched variants.
- **Frontend (`ReceiptMatchModal.jsx`):** A planned item stays selectable as a link target while it has remaining capacity; show remaining count; do not treat the extra variant as an unavoidable "extra".
- **No new endpoints**; behavior of existing match/link/confirm endpoints changes.
