## Context

Matching data model: `ReceiptItem.MatchedItemID *uuid.UUID` points to a planned `Item`; the planned `Item.ReceiptItemID *uint` points back to a single receipt item; `Item.IsBought` and price/quantity are updated on match (`receipt_service.go`). The `LinkAlias` handler (`items.go:786-792`) **deletes** the planned item after linking when `planned_item_id` is supplied, to avoid duplicates. The match modal (`ReceiptMatchModal.jsx`) stages decisions locally and blocks confirm while any receipt item is undecided; an unmatched item with no suggestions is an "extra".

The one-to-one back-link and the delete-on-link rule are what break split purchases: the first variant consumes (and deletes) the planned item, leaving the second variant with nothing to match.

## Goals / Non-Goals

**Goals:**
- Let one planned item (quantity N) absorb up to N receipt variants without duplicates.
- Preserve the planned item until its quantity is consumed.
- Record every variant as an alias.

**Non-Goals:**
- No change to AI parsing of receipts.
- No automatic splitting of a planned line into per-variant rows on the list.
- No hard enforcement that stops the manager from over-linking beyond N (kept permissive).

## Decisions

- **Treat `ReceiptItem.MatchedItemID` as the source of truth (many-to-one).** Multiple receipt items may reference the same planned item. The planned item's single `Item.ReceiptItemID` is demoted to "primary/first match" (or ignored); membership is derived by querying receipt items whose `MatchedItemID` = planned id. This avoids a schema migration to a join table.

- **Remaining capacity = `plannedItem.Quantity − count(matched receipt items)`.** A planned item stays a valid link target while capacity > 0. The modal surfaces the remaining count. Over-linking beyond capacity is allowed (permissive) but not encouraged.

- **`LinkAlias` no longer deletes the planned item when capacity remains.** New rule: upsert the variant alias (unchanged), then:
  - If linking would leave remaining capacity ≥ 1 → keep the planned item, mark it bought, attach the match.
  - If this link consumes the last unit → keep the planned item as the bought record (do **not** delete). The original delete-on-link behavior existed to avoid a duplicate generic row; with split matching the planned item *is* the aggregated bought record, so retention is correct.
  (Decision to confirm in review: retain vs. delete-when-fully-consumed. Retention is simpler and keeps the aggregated spend visible.)

- **Price aggregation.** When multiple receipt items match one planned item, the planned item's recorded spend is the sum of matched line prices; quantity stays as planned. `recalculateListTotal` already sums item prices, so we set the planned item's price to the running sum of its matched lines.

- **Alias per variant.** `upsertItemAlias` is already keyed by receipt name, so each variant naturally becomes its own alias of the planned name — no change beyond calling it once per matched variant.

## Risks / Trade-offs

- [Deriving membership by query changes several call sites that assumed one receipt item per planned item] → Audit `receipt_service.go` match/unmatch/confirm and the modal's decision staging; add tests for the two-variant path.
- [Price aggregation double-counting on re-confirm/unmatch] → Recompute the planned item's spend from its currently-matched receipt items rather than incrementally adding, to stay idempotent.
- [Unmatch semantics] → Unmatching one of several variants should decrement capacity/spend and only clear `IsBought` when the last match is removed.
- [Modal "extra" logic] → An additional variant of a planned item must not be classified as an unavoidable extra while the planned item has capacity.

## Open Questions

- When a planned item's quantity is fully consumed by variants, do we keep the single aggregated planned row (recommended) or split it into one row per variant for clearer price history? Default: keep aggregated; per-variant history already lives in aliases.
- Should capacity be a hard cap (block over-linking) or a soft hint (allow, warn)? Default: soft hint.
- If the manager never set a quantity (defaults to 1), do we still allow a second variant to link? Default: yes, permissively (soft cap), so the common "forgot to set quantity" case still works.
