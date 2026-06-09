# Flyer Browsing

## Purpose
Let users browse discounted flyer items, filter them, and add them to shopping lists.

## Requirements

### Requirement: Browse discounted items

The manager SHALL be able to browse current and upcoming store deals on the Flyers page.

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

---

### Requirement: Add flyer item to a list

The manager SHALL be able to add a flyer item to a new or existing shopping list.

#### Scenario: Add to existing list
- **GIVEN** the manager is on the Flyers page
- **WHEN** the manager clicks "Add to List" on a flyer card and selects an existing list
- **THEN** the item is added to that list with name, price, and photo pre-filled from the flyer deal

#### Scenario: Add to new list
- **WHEN** the manager clicks "Add to List" and selects "Create New List"
- **THEN** a new list is created and the flyer item is added to it

#### Scenario: Flyer-linked item name and price are protected
- **GIVEN** an item was added from a flyer
- **THEN** the item's name and price fields are read-only in list detail (to preserve the deal record)

#### Scenario: Photo pre-filled from flyer
- **GIVEN** the flyer item has an extracted photo
- **WHEN** the item is added to a list
- **THEN** the item's `local_photo_path` is set to the flyer image

---

### Requirement: Graceful degradation without Gemini

The Flyers feature SHALL degrade gracefully when no Gemini API key is configured.

#### Scenario: Flyers page empty without Gemini key
- **GIVEN** `GEMINI_API_KEY` is not configured
- **THEN** the flyer scheduler does not run and no items are parsed (page shows empty state)

#### Scenario: Existing flyer data visible without Gemini
- **GIVEN** flyer data was parsed previously and is stored in the DB
- **WHEN** `GEMINI_API_KEY` is removed
- **THEN** previously parsed items remain visible and browsable

---

### Requirement: Flyer stats page

The system SHALL provide a stats page that shows aggregated flyer data.

#### Scenario: Stats page shows aggregated flyer data
- **GIVEN** the manager opens `/flyers/stats`
- **THEN** a dashboard is shown with: top shops by item count, price trends, most popular items, and activity timeline
