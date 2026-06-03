## MODIFIED Requirements

### Requirement: Browse discounted items
The manager can browse current and upcoming store deals on the Flyers page.

#### Scenario: Flyer items page shows a card grid
- **GIVEN** flyer data has been parsed and stored
- **WHEN** the manager opens the Flyers page
- **THEN** items are displayed as cards with: photo (or placeholder if image expired), name, price, original price (if discounted), shop name, and valid date range

#### Scenario: Item card shows placeholder when local image is gone
- **GIVEN** a flyer item's `LocalPhotoPath` has been cleared by the expiry cleanup
- **WHEN** the manager views that item's card
- **THEN** a placeholder image is displayed (no broken image indicator, no fallback to `PhotoURL`)

#### Scenario: Filter by shop
- **WHEN** the manager selects a shop from the shop dropdown filter
- **THEN** only items from that shop are shown

#### Scenario: Filter by search text
- **WHEN** the manager types in the search field
- **THEN** only items whose name, categories, or keywords contain the search text are shown (case-insensitive)

#### Scenario: Filter by activity — "Now"
- **WHEN** the manager selects "Now" in the activity filter
- **THEN** only items whose valid date range includes today are shown

#### Scenario: Filter by activity — "Future"
- **WHEN** the manager selects "Future" in the activity filter
- **THEN** only items whose start date is in the future are shown

#### Scenario: Infinite scroll loads more items
- **GIVEN** more than 24 flyer items match the current filters
- **WHEN** the manager scrolls to the bottom of the page
- **THEN** the next page of items is loaded and appended to the grid
