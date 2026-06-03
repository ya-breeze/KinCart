## ADDED Requirements

### Requirement: Expired flyer images are deleted automatically
The system SHALL delete local image files for flyers whose effective expiry date is more than 30 days in the past, while preserving all DB records.

Effective expiry date = `Flyer.EndDate` if set and non-zero; otherwise `Flyer.CreatedAt`.

#### Scenario: Source page files deleted after expiry window
- **WHEN** a flyer's effective expiry date is more than 30 days ago
- **THEN** all `FlyerPage.LocalPath` files for that flyer are deleted from disk
- **AND** `FlyerPage.LocalPath` is cleared (set to empty string) in the DB

#### Scenario: Item crop files deleted after expiry window
- **WHEN** a flyer's effective expiry date is more than 30 days ago
- **THEN** all `FlyerItem.LocalPhotoPath` files for that flyer are deleted from disk
- **AND** `FlyerItem.LocalPhotoPath` is cleared (set to empty string) in the DB

#### Scenario: Remote PhotoURL is preserved
- **WHEN** a flyer item's local image file is deleted
- **THEN** `FlyerItem.PhotoURL` retains its original value unchanged

#### Scenario: All DB records are preserved
- **WHEN** a flyer's images are cleaned up
- **THEN** the `Flyer`, `FlyerPage`, and `FlyerItem` records remain in the DB with all fields intact (name, price, dates, categories, keywords, etc.)

#### Scenario: Cleanup is idempotent
- **WHEN** the cleanup runs and a local file is already missing from disk
- **THEN** the DB path is still cleared and no error is raised

#### Scenario: Cleanup runs before daily backup
- **WHEN** the daily backup task starts
- **THEN** the image cleanup step runs first, so the resulting archive reflects the freed space

#### Scenario: Non-expired flyers are untouched
- **WHEN** a flyer's effective expiry date is within the last 30 days
- **THEN** no files are deleted and no DB paths are cleared for that flyer
