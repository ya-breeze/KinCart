## ADDED Requirements

### Requirement: Remember unit and category from purchase history

The system SHALL record, per family, the unit and category an item name was last used as, so future additions of the same item can be prefilled. The unit SHALL be recorded per shop (via `ItemAlias`); the category SHALL be recorded per item name.

#### Scenario: Unit and category are recorded on purchase
- **WHEN** an item is matched to a receipt or a receipt-created item is saved
- **THEN** the item's alias record stores that item's unit and category

#### Scenario: Per-shop unit is retained
- **GIVEN** the same item bought at two different shops with different units
- **THEN** each shop's alias retains its own recorded unit

### Requirement: Prefill item defaults on add

When an item is added, the system SHALL prefill its unit and category from history first, then from an AI common-sense guess, then fall back to the plain defaults.

#### Scenario: Unit resolved from the list's shop
- **GIVEN** the item's list has a shop and history records a unit for that item at that shop
- **WHEN** the item is added
- **THEN** the item's unit defaults to the shop-specific remembered unit

#### Scenario: Unit resolved across shops when no shop match
- **GIVEN** the item's list has no shop, or no shop-specific history exists
- **AND** history records a unit for that item at other shops
- **WHEN** the item is added
- **THEN** the item's unit defaults to the most common remembered unit for that name

#### Scenario: Category resolved from history
- **GIVEN** history records a category for that item name
- **WHEN** the item is added
- **THEN** the item's category defaults to the remembered category

#### Scenario: AI fallback for an unseen item
- **GIVEN** no history exists for the item name
- **AND** the AI service is available
- **WHEN** the item is added
- **THEN** the system requests a common-sense unit and a category **chosen from the family's own category names** (the AI is given the family's categories and must return one of them or none — it never invents a name)
- **AND** the returned category name is matched to its category row using Cyrillic-safe (Go-lowercased) comparison, not SQL `LOWER()`

#### Scenario: AI returns a category the family does not have
- **GIVEN** the AI returns no category, or a name that does not match any of the family's categories
- **WHEN** the item is added
- **THEN** the item is left uncategorized (no category is invented)

#### Scenario: Plain defaults when nothing is known
- **GIVEN** no history exists and the AI service is unavailable or returns no usable category
- **WHEN** the item is added
- **THEN** the item keeps unit `pcs` and remains uncategorized

#### Scenario: An explicit user choice is never overridden
- **WHEN** the request already specifies a unit or category
- **THEN** the provided value is kept and no inference replaces it
