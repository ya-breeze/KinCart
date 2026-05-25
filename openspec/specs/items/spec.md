# Feature: List Items

## Requirement: Add item via quick-add bar
The manager can add items by typing in the quick-add bar at the top of the list detail page.

### Scenario: Add item by typing name and pressing Enter
- **GIVEN** the manager is viewing a list in `preparing` or `ready for shopping` status
- **WHEN** the manager types an item name and presses Enter
- **THEN** a ConfirmSheet slides up showing the item name, quantity (default 1), unit (default "pcs"), and optional price field

### Scenario: Confirm adds item to list
- **GIVEN** the ConfirmSheet is open
- **WHEN** the manager clicks "Add to List"
- **THEN** the item is added to the list and appears grouped under its category

### Scenario: Price pre-filled from alias history
- **GIVEN** the family has a past alias for the item name with a known price
- **WHEN** the manager types that item name
- **THEN** the ConfirmSheet pre-fills the price field with the alias's last known price

### Scenario: Quick-add bar hidden on completed list
- **GIVEN** the list has status `completed`
- **THEN** the quick-add bar is NOT rendered in the DOM

### Scenario: Quick-add bar reappears when status leaves completed
- **GIVEN** a list was `completed` and its status is changed back to `preparing`
- **THEN** the quick-add bar becomes visible again immediately

---

## Requirement: Add item via frequent-item chips
Frequently purchased items appear as clickable chips for one-click adding.

### Scenario: Frequent-item chips shown below quick-add bar
- **GIVEN** the family has items with frequency ≥ 2
- **THEN** chips for those items appear below the quick-add bar (sorted by frequency desc)

### Scenario: Clicking a chip opens ConfirmSheet
- **WHEN** the manager clicks a frequent-item chip
- **THEN** the ConfirmSheet opens pre-filled with that item's name and last known price

### Scenario: Frequent-item chips hidden on completed list
- **GIVEN** the list has status `completed`
- **THEN** the frequent-item chip grid is NOT rendered in the DOM

---

## Requirement: Item attributes
Each item can carry metadata beyond its name.

### Scenario: Item defaults
- **WHEN** an item is added without specifying quantity or unit
- **THEN** it is created with quantity = 1 and unit = "pcs"

### Scenario: Valid unit values — quick-add (ConfirmSheet)
- **THEN** the ConfirmSheet unit dropdown offers: pcs, g, kg, ml, L, pack

### Scenario: Valid unit values — inline edit (ListDetail)
- **THEN** the inline item-edit unit dropdown offers: pcs, kg, g, 100g, l, pack
- **NOTE** The two dropdowns have different sets: ConfirmSheet has ml/L but not 100g; inline edit has 100g but not ml

### Scenario: Item can be marked urgent
- **WHEN** the manager marks an item as urgent
- **THEN** the item is visually distinguished (e.g., highlighted) in both manager and shopper views

### Scenario: Item can have a photo
- **WHEN** the manager uploads a JPEG/PNG/WebP photo (≤ 10 MB) for an item
- **THEN** the photo is displayed next to the item in the list

### Scenario: Photo upload rejects non-image files
- **WHEN** the manager uploads a file that is not JPEG, PNG, or WebP
- **THEN** an error is shown and the photo is not saved

---

## Requirement: Edit item
The manager can edit an item's name, quantity, unit, price, and category.

### Scenario: Edit item inline
- **GIVEN** the manager is viewing a list in Manager mode
- **WHEN** the manager expands an item row
- **THEN** edit controls for name, qty, unit, price, and category are visible

### Scenario: Flyer-linked item name and price are read-only
- **GIVEN** an item was added from a flyer deal (`flyer_item_id` is set)
- **THEN** the name and price fields are NOT editable
- **AND** an "Unlink from flyer" control is visible

### Scenario: Unlinking flyer re-enables editing
- **WHEN** the manager unlinks the flyer from an item
- **THEN** the item's name and price become editable

---

## Requirement: Delete item

### Scenario: Delete removes item from list
- **WHEN** the manager deletes an item
- **THEN** the item is immediately removed from the list view

---

## Requirement: Check off item (mark as bought)
Both manager and shopper can mark items as bought in-store.

### Scenario: Checking off item updates visual state
- **WHEN** the user taps the checkbox next to an item
- **THEN** the item is visually struck through or grayed out

### Scenario: Progress bar updates on check-off
- **GIVEN** a list with N items
- **WHEN** the user marks one more item as bought
- **THEN** the progress bar increments by 1/N

### Scenario: Unchecking restores item
- **GIVEN** an item is marked as bought
- **WHEN** the user taps its checkbox again
- **THEN** `is_bought` is set to false and the item returns to normal appearance

---

## Requirement: Category grouping
Items are displayed grouped by their assigned category.

### Scenario: Items grouped by category in list detail
- **GIVEN** a list with items assigned to "Dairy" and "Vegetables"
- **THEN** items appear under two category headers in that order (by category sort_order)

### Scenario: Uncategorized items shown at end
- **GIVEN** a list where some items have no category assigned
- **THEN** those items appear under an "Uncategorized" group after all categorized groups

### Scenario: Shop-specific category order applied when shop is selected
- **GIVEN** the manager has configured a custom aisle order for a shop
- **WHEN** the user selects that shop on the list detail page
- **THEN** category groups are reordered to match that shop's aisle layout
