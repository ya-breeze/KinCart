## 1. Backend — cleanup logic

- [ ] 1.1 Add `cleanupExpiredFlyerImages(db, uploadsPath, flyerItemsPath)` function to `backup` package that queries flyers with effective expiry > 30 days ago
- [ ] 1.2 For each eligible flyer: delete `FlyerPage.LocalPath` files and bulk-clear the column in DB
- [ ] 1.3 For each eligible flyer: delete `FlyerItem.LocalPhotoPath` files and bulk-clear the column in DB
- [ ] 1.4 Handle missing files gracefully (log warning, continue — idempotent)
- [ ] 1.5 Call `cleanupExpiredFlyerImages` as the first step in `backup.Task.run()`, before archive creation

## 2. Backend — tests

- [ ] 2.1 Unit test: eligible flyer (EndDate set, > 30 days ago) — files deleted, DB paths cleared, PhotoURL unchanged
- [ ] 2.2 Unit test: eligible flyer (EndDate zero, CreatedAt > 30 days ago) — falls back to CreatedAt correctly
- [ ] 2.3 Unit test: non-eligible flyer (EndDate within 30 days) — no files touched
- [ ] 2.4 Unit test: idempotency — file already missing, no error raised, DB path still cleared

## 3. Frontend — graceful degradation

- [ ] 3.1 In the flyer item card component, pass `PhotoURL` as a fallback src to `LazyImage` when `LocalPhotoPath` is empty
- [ ] 3.2 Verify `LazyImage` already handles load errors with placeholder (no additional change needed if so)
- [ ] 3.3 Visually confirm: card with expired image shows placeholder without broken image indicator

## 4. Verification

- [ ] 4.1 Deploy to WIP stack and confirm cleanup runs on backup trigger (check logs)
- [ ] 4.2 Confirm DB records (flyer, pages, items) remain after cleanup
- [ ] 4.3 Confirm backup archive size reflects freed space
- [ ] 4.4 Confirm flyer cards render correctly for items with cleared `LocalPhotoPath`
