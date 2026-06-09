# Frequent Items & Item Aliases

## Purpose
Track item purchase frequency and aliases to power quick-add chips and autocomplete.

## Requirements

### Requirement: Frequency tracking

Every time an item is added to a list its frequency count SHALL be incremented.

#### Scenario: Adding an item increments its frequency
- **WHEN** the manager adds an item named "Milk" to a list
- **THEN** `ItemFrequency.frequency` for "Milk" is incremented by 1

#### Scenario: Items with frequency < 2 do not appear as chips
- **GIVEN** "Bread" has been added to a list exactly once
- **THEN** "Bread" does NOT appear in the frequent-item chip grid

#### Scenario: Items with frequency ≥ 2 appear as chips
- **GIVEN** "Milk" has been added to lists at least twice
- **THEN** "Milk" appears as a chip in the quick-add bar

#### Scenario: Chips sorted by frequency descending
- **GIVEN** "Milk" has frequency 10 and "Eggs" has frequency 3
- **THEN** "Milk" chip appears before "Eggs" chip

#### Scenario: Chip list is capped at 10 items
- **GIVEN** more than 10 items have frequency ≥ 2
- **THEN** only the top 10 by frequency are returned by the API and shown as chips

---

### Requirement: Hide and restore frequent items

The manager SHALL be able to remove an item from the suggestion chips without deleting its history, and restore it later.

#### Scenario: Hiding removes chip from quick-add bar
- **GIVEN** "Milk" appears as a frequent-item chip
- **WHEN** the manager hides "Milk"
- **THEN** "Milk" no longer appears in the chip grid

#### Scenario: Hidden items can be restored
- **GIVEN** "Milk" is hidden
- **WHEN** the manager restores "Milk" from the Frequent Items page
- **THEN** "Milk" reappears as a chip (if frequency ≥ 2)

#### Scenario: Hiding suppresses frequency increment
- **GIVEN** an item is marked hidden
- **WHEN** the same item name is added to a list via quick-add
- **THEN** the frequency is NOT incremented (hiding skips both chip display and frequency tracking)

---

### Requirement: Item aliases

An alias SHALL map a generic planned name (e.g., "Milk") to a specific receipt name (e.g., "Parmalat UHT 1L") at a given shop.

#### Scenario: Alias created on receipt match confirmation
- **GIVEN** a receipt item "Parmalat UHT 1L" is matched to planned item "Milk"
- **WHEN** the user confirms the match
- **THEN** an alias is created: planned_name="Milk", receipt_name="Parmalat UHT 1L"

#### Scenario: Alias upsert increments purchase count
- **GIVEN** an alias (planned="Milk", receipt="Parmalat UHT 1L") already exists with count=2
- **WHEN** the same mapping is confirmed again
- **THEN** purchase_count becomes 3 and last_used_at is updated

#### Scenario: Same planned name can have multiple aliases
- **GIVEN** "Milk" has been matched to "Parmalat UHT 1L" at Lidl and "Tesco Whole Milk" at Tesco
- **THEN** both appear as variants under "Milk" on the Frequent Items page

#### Scenario: Price pre-fill uses most-used alias variant
- **GIVEN** "Milk" has two variants: "Parmalat" (count=5, price=29.90) and "Tesco Milk" (count=1, price=31.00)
- **WHEN** the manager types "Milk" in the quick-add bar
- **THEN** the ConfirmSheet pre-fills price = 29.90 (from the highest-count variant)

---

### Requirement: Alias management page

The manager SHALL be able to view and edit all aliases from the Settings or Aliases page.

#### Scenario: Aliases listed grouped by planned name
- **GIVEN** aliases exist for "Milk" and "Eggs"
- **THEN** the aliases page shows two groups: "Milk" (with its variants) and "Eggs" (with its variants)

#### Scenario: Rename alias group updates all variants
- **WHEN** the manager renames the "Milk" group to "Mléko"
- **THEN** all aliases with planned_name="Milk" now have planned_name="Mléko"

#### Scenario: Delete alias group removes all variants
- **WHEN** the manager deletes the "Milk" group
- **THEN** all aliases with planned_name="Milk" are removed

#### Scenario: Edit individual alias receipt name
- **WHEN** the manager edits the receipt_name of a single alias
- **THEN** only that alias is updated; other aliases in the group are unchanged

#### Scenario: Delete individual alias
- **WHEN** the manager deletes one alias variant
- **THEN** only that variant is removed; others in the group remain

---

### Requirement: Item suggestions autocomplete

Typing in the quick-add bar SHALL trigger autocomplete from alias history.

#### Scenario: Autocomplete requires at least 2 characters
- **WHEN** the manager types 1 character
- **THEN** no autocomplete suggestions are shown

#### Scenario: Autocomplete returns matching planned names
- **GIVEN** aliases exist for "Milk" and "Mléko"
- **WHEN** the manager types "ml"
- **THEN** "Mléko" appears in autocomplete suggestions (case-insensitive prefix match: returns items whose planned_name starts with the query, not contains)
