## 1. Backend — matching model

- [ ] 1.1 Add a helper to count receipt items matched to a planned item (`MatchedItemID = plannedID`) and compute remaining capacity (`Quantity − matchedCount`)
- [ ] 1.2 In the confirm/auto-match path (`receipt_service.go`), allow assigning a receipt item to an already-matched planned item while capacity remains; mark bought on first match
- [ ] 1.3 Recompute the planned item's spend idempotently from all its currently-matched receipt lines (avoid incremental double-count)
- [ ] 1.4 Ensure unmatching one variant decrements capacity/spend and only clears `is_bought` when the last match is removed

## 2. Backend — link-alias

- [ ] 2.1 In `LinkAlias`, stop deleting the planned item when it still has (or, per review decision, when it has any) quantity; keep it as the aggregated bought record
- [ ] 2.2 Upsert an alias per matched variant (verify existing per-receipt-name keying already covers this)

## 3. Backend — tests

- [ ] 3.1 Two receipt lines match one planned item (qty 2): both linked, no duplicate, planned item bought, spend = sum
- [ ] 3.2 Planned item retained after first link so the second link succeeds
- [ ] 3.3 Unmatch one of two variants: capacity/spend adjust; still bought while one remains
- [ ] 3.4 Over-link beyond quantity is permitted and does not break the review

## 4. Frontend — match modal

- [ ] 4.1 Keep a planned item selectable as a link target while it has remaining capacity; show remaining count
- [ ] 4.2 Do not classify an additional variant as an unavoidable "extra" while the planned item has capacity
- [ ] 4.3 Ensure staged-decision confirm handles multiple receipt items pointing at one planned item

## 5. Verification

- [ ] 5.1 `make lint`, `make test-backend`, `make test-frontend` pass
- [ ] 5.2 E2E/manual: plan "2 fruit kefirs", upload a receipt with two kefir variants, match both to the one planned item; verify no duplicate, correct bought state and total, and both variants become aliases
