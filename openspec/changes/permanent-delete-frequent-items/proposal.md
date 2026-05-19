## Why

When a user deletes a frequent item chip, the chip reappears the next time they add an item with the same name — because deletion is a hard DELETE with no memory of the user's intent. Users who remove phantom chips (bag fees, receipt noise, mistyped names) get them back uninvited, eroding trust in the feature.

## What Changes

- Frequent item deletion becomes a **soft-delete**: sets `is_hidden = true` on the `ItemFrequency` row instead of deleting it.
- All three upsert callsites are guarded: a hidden row is never revived by future item additions or receipt processing.
- Case-sensitivity inconsistency is fixed: all upsert lookups use `LOWER()` so "Milk" and "milk" resolve to the same record.
- A new **Frequent Items settings page** (`/settings/frequent-items`) lists hidden items and lets users restore any of them.
- SettingsPage gains a link to the new page alongside the existing Aliases and Flyer Stats links.

## Capabilities

### New Capabilities
- `frequent-item-hidden-state`: Frequent items can be hidden (soft-deleted) by the user. Hidden items do not appear as chips and are not revived by future purchases. Users can restore hidden items from a settings page.

### Modified Capabilities
<!-- No existing spec-level requirements are changing — this adds new behavior to the existing frequent-items chip feature without altering its current requirements. -->

## Impact

- **Backend model**: `ItemFrequency` gains `is_hidden bool` field + DB migration.
- **Backend handlers**: `DELETE /api/family/frequent-items/:id` (soft-delete), `GET /api/family/frequent-items` (filter hidden), new `GET /api/family/frequent-items/hidden`, new `PATCH /api/family/frequent-items/:id/restore`.
- **Backend upserts**: `handlers/items.go` (×2) and `services/receipt_service.go` (×1) — add hidden-check and `LOWER()` normalization.
- **Frontend**: new `FrequentItemsPage.jsx`, new route `/settings/frequent-items`, link added to `SettingsPage.jsx`.
- **No API breaking changes**: the existing DELETE endpoint keeps the same URL; its behavior changes internally.
