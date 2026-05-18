## Context

`ItemFrequency` tracks how often each item name is used by a family, powering the chip grid in `ListDetail`. Currently, deleting a chip hard-deletes the row. Three separate callsites recreate it: single-item creation (`handlers/items.go`), bulk-item creation (`handlers/items.go`), and receipt match confirmation (`services/receipt_service.go`). Additionally, the two handler callsites use exact-match `WHERE item_name = ?` while the receipt service uses `LOWER(item_name)`, creating duplicate rows for the same item in different cases.

## Goals / Non-Goals

**Goals:**
- Hidden chips never return without explicit user action.
- Case variants ("Milk" / "milk") resolve to the same frequency record.
- Users can discover and restore hidden items from Settings.

**Non-Goals:**
- Auto-restoring items after N purchases (keep it simple — manual only).
- Hiding items from search/autocomplete suggestions (those use `ItemAlias`, not `ItemFrequency`).
- Bulk-hide or bulk-restore operations.

## Decisions

### Soft-delete via `is_hidden` flag (over hard-delete + suppression table)

A separate `SuppressedItem` table would require a JOIN or subquery on every upsert. The `is_hidden` flag keeps all state in one row and one table. At family scale (tens to low hundreds of items), a slightly larger table is not a concern. Restoring an item is a single `UPDATE`.

### Upsert behavior when `is_hidden = true`: skip entirely

When an upsert finds a hidden row, it does nothing — no frequency increment, no un-hiding. The user explicitly hid it; future purchases should not change that. Keeping the counter stale is acceptable because hidden items are not displayed.

### Case normalization: `LOWER()` at lookup time, preserve display case

Store `ItemName` as typed (for display). All lookups use `WHERE LOWER(item_name) = LOWER(?)`. This means the first-added casing wins for display — acceptable tradeoff over a separate `item_name_lower` indexed column, which would require a schema change and complicates the upsert.

### Restore UI: new sub-page under `/settings/frequent-items`

SettingsPage already links to `/settings/aliases` and `/settings/flyer-stats` as sub-pages. The same pattern keeps the main settings page uncluttered. The new page shows only hidden items; active items are not listed (users manage those from the chip grid).

### API: `PATCH /api/family/frequent-items/:id/restore` (over generic PATCH)

An explicit `/restore` action is clearer in intent than `PATCH {is_hidden: false}` and avoids exposing other fields. The hidden-item list endpoint is a separate `GET /api/family/frequent-items/hidden` rather than a query param on the existing endpoint, to keep the chip-grid fetch simple.

## Risks / Trade-offs

- **Stale frequency counter**: A hidden item's `Frequency` and `LastPrice` freeze at hide time. If restored later, the chip's sort position reflects old data. → Acceptable; the user chose to restore it.
- **Case normalization edge case**: If two rows already exist in prod for "Milk" and "milk", the LOWER() lookup will find one arbitrarily (whichever `First()` returns). The other becomes unreachable via normal upsert. → These orphan rows will still show as chips (is_hidden=false) until individually deleted. A one-time dedup migration is out of scope.
- **ListDetail.jsx change not needed**: The DELETE call from the frontend is unchanged — the endpoint URL stays the same. Only the backend behavior changes. This is a clean seam.

## Migration Plan

1. Add `is_hidden BOOLEAN NOT NULL DEFAULT FALSE` to `item_frequencies` table via GORM `AutoMigrate`.
2. No data migration needed — existing rows get `is_hidden = false` by default.
3. Deploy backend first (new column, updated endpoints). Frontend continues to work — delete still works, chips still render (all existing rows have `is_hidden = false`).
4. Deploy frontend (new settings page + link). No rollback risk since it's additive.
5. Rollback: set `is_hidden = false` on all rows and revert to hard-delete handler. No data loss since we're not deleting rows.
