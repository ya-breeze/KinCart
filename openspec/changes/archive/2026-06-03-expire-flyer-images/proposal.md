## Why

Flyer image files (cropped item crops in `flyer_items/` and raw page downloads in `uploads/flyer_pages/`) accumulate indefinitely and are the dominant cost in daily backups (~7.9 GB/backup). Flyers are time-limited promotions — once a flyer expires, its images have no functional value but the price history in DB records remains useful.

## What Changes

- A cleanup step runs at the **start of each daily backup task**, before the archive is created
- Any flyer whose effective expiry date is more than 30 days in the past is considered eligible for image cleanup
- Effective expiry = `flyer.EndDate` if set; otherwise `flyer.CreatedAt + 30 days`
- For each eligible flyer:
  - All `FlyerPage.LocalPath` files on disk are deleted; `LocalPath` is cleared in DB
  - All `FlyerItem.LocalPhotoPath` files on disk are deleted; `LocalPhotoPath` is cleared in DB
- `FlyerItem.PhotoURL` (remote shop URL) is **preserved** — no change (not used as fallback)
- All DB records (flyers, pages, items, prices, dates) are fully preserved

## Capabilities

### New Capabilities
- `flyer-image-expiry`: Background cleanup that deletes local image files for expired flyers while preserving all DB records and remote URLs

### Modified Capabilities
- `flyers`: Items with expired local images must degrade gracefully in the UI (no broken image indicators)

## Impact

- `backend/internal/backup/backup.go` — add cleanup step before archive creation
- `backend/internal/models/models.go` — no structural changes; `LocalPhotoPath` and `LocalPath` fields already nullable-equivalent (empty string = no file)
- `frontend/src/` — flyer card must show a placeholder when `LocalPhotoPath` is empty (no fallback to `PhotoURL`)
- Disk: immediate reduction of ~7 GB in live data and ~79 GB in backup storage once old backups cycle out
