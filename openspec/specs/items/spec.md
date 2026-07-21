# List Items

## Purpose
Manage the items within a shopping list — adding, editing, checking off, and grouping them.
## Requirements
### Requirement: Add item via quick-add bar

The manager SHALL be able to add items by typing in the quick-add bar at the top of the list detail page.

#### Scenario: Add item by typing name and pressing Enter
- **GIVEN** the manager is viewing a list in `preparing` or `ready for shopping` status
- **WHEN** the manager types an item name and presses Enter
- **THEN** a ConfirmSheet slides up showing the item name, quantity (default 1), unit (default "pcs"), and optional price field

#### Scenario: Confirm adds item to list
- **GIVEN** the ConfirmSheet is open
- **WHEN** the manager clicks "Add to List"
- **THEN** the item is added to the list and appears grouped under its category

#### Scenario: Price pre-filled from alias history
- **GIVEN** the family has a past alias for the item name with a known price
- **WHEN** the manager types that item name
- **THEN** the ConfirmSheet pre-fills the price field with the alias's last known price

#### Scenario: Quick-add bar hidden on completed list
- **GIVEN** the list has status `completed`
- **THEN** the quick-add bar is NOT rendered in the DOM

#### Scenario: Quick-add bar reappears when status leaves completed
- **GIVEN** a list was `completed` and its status is changed back to `preparing`
- **THEN** the quick-add bar becomes visible again immediately

---

### Requirement: Add item via frequent-item chips

Frequently purchased items SHALL appear as clickable chips for one-click adding.

#### Scenario: Frequent-item chips shown below quick-add bar
- **GIVEN** the family has items with frequency ≥ 2
- **THEN** chips for those items appear below the quick-add bar (sorted by frequency desc)

#### Scenario: Clicking a chip opens ConfirmSheet
- **WHEN** the manager clicks a frequent-item chip
- **THEN** the ConfirmSheet opens pre-filled with that item's name and last known price

#### Scenario: Frequent-item chips hidden on completed list
- **GIVEN** the list has status `completed`
- **THEN** the frequent-item chip grid is NOT rendered in the DOM

---

### Requirement: Item attributes

Each item MAY carry metadata beyond its name, and the system SHALL persist these attributes.

#### Scenario: Item defaults
- **WHEN** an item is added without specifying quantity or unit
- **THEN** it is created with quantity = 1 and unit = "pcs"

#### Scenario: Valid unit values — quick-add (ConfirmSheet)
- **THEN** the ConfirmSheet unit dropdown offers: pcs, g, kg, ml, L, pack

#### Scenario: Valid unit values — inline edit (ListDetail)
- **THEN** the inline item-edit unit dropdown offers: pcs, kg, g, 100g, l, pack
- **NOTE** The two dropdowns have different sets: ConfirmSheet has ml/L but not 100g; inline edit has 100g but not ml

#### Scenario: Item can be marked urgent
- **WHEN** the manager marks an item as urgent
- **THEN** the item is visually distinguished (e.g., highlighted) in both manager and shopper views

#### Scenario: Item can have a photo
- **WHEN** the manager uploads a JPEG/PNG/WebP photo (≤ 10 MB) for an item
- **THEN** the photo is displayed next to the item in the list

#### Scenario: Photo upload rejects non-image files
- **WHEN** the manager uploads a file that is not JPEG, PNG, or WebP
- **THEN** an error is shown and the photo is not saved

---

### Requirement: Edit item

The manager SHALL be able to edit an item's name, quantity, unit, price, and category.

#### Scenario: Edit item inline
- **GIVEN** the manager is viewing a list in Manager mode
- **WHEN** the manager expands an item row
- **THEN** edit controls for name, qty, unit, price, and category are visible

#### Scenario: Flyer-linked item name and price are read-only
- **GIVEN** an item was added from a flyer deal (`flyer_item_id` is set)
- **THEN** the name and price fields are NOT editable
- **AND** an "Unlink from flyer" control is visible

#### Scenario: Unlinking flyer re-enables editing
- **WHEN** the manager unlinks the flyer from an item
- **THEN** the item's name and price become editable

---

### Requirement: Delete item

The manager SHALL be able to delete an item from a list.

#### Scenario: Delete removes item from list
- **WHEN** the manager deletes an item
- **THEN** the item is immediately removed from the list view

---

### Requirement: Check off item (mark as bought)

Both manager and shopper SHALL be able to mark items as bought in-store.

#### Scenario: Checking off item updates visual state
- **WHEN** the user taps the checkbox next to an item
- **THEN** the item is visually struck through or grayed out

#### Scenario: Progress bar updates on check-off
- **GIVEN** a list with N items
- **WHEN** the user marks one more item as bought
- **THEN** the progress bar increments by 1/N

#### Scenario: Unchecking restores item
- **GIVEN** an item is marked as bought
- **WHEN** the user taps its checkbox again
- **THEN** `is_bought` is set to false and the item returns to normal appearance

---

### Requirement: Mark item as absent (out of stock)

An item SHALL have an `is_absent` state that the shopper can set and clear. Absent means the item was not available in the store on this trip. `is_absent` and `is_bought` SHALL be mutually exclusive: an item SHALL NOT be both at once, and bought takes precedence.

#### Scenario: Shopper marks an item absent
- **WHEN** the shopper taps the "mark absent" control on an active item
- **THEN** the item's `is_absent` is set to true and it is treated as de-prioritized

#### Scenario: Shopper unmarks an absent item
- **GIVEN** an item marked absent
- **WHEN** the shopper taps the control again
- **THEN** `is_absent` is set to false and the item returns to the active list

#### Scenario: Marking an absent item bought clears absent
- **GIVEN** an item marked absent
- **WHEN** the shopper marks it bought (e.g. found it after all)
- **THEN** `is_bought` is true and `is_absent` is cleared to false
- **AND** the item is shown as bought, not as absent

#### Scenario: A bought item cannot be marked absent
- **GIVEN** an item already marked bought
- **WHEN** a request attempts to set `is_absent` to true
- **THEN** the request is rejected with 400 and the item is left unchanged

#### Scenario: Clearing bought while setting absent is allowed
- **GIVEN** an item already marked bought
- **WHEN** a single request sets `is_bought` to false and `is_absent` to true
- **THEN** the request succeeds and the item ends up absent and not bought
- **AND** exclusivity is judged on the state the request produces, not on the fields it mentions

#### Scenario: Receipt matching clears absent
- **GIVEN** an item the shopper marked absent
- **WHEN** receipt matching marks that item bought
- **THEN** `is_absent` is cleared to false

#### Scenario: Unmatching a receipt item does not restore absent
- **GIVEN** an item that was marked absent and then marked bought by receipt matching
- **WHEN** the receipt match is undone and `is_bought` returns to false
- **THEN** `is_absent` remains false and the item returns to the active list

#### Scenario: Absent and bought are visually distinguishable
- **GIVEN** the done section contains one bought and one absent item
- **THEN** the two are rendered with distinct styling so the shopper can tell them apart

---

### Requirement: Shopper done section

In the shopper view, items that are bought or absent SHALL be collected into a single "done" section at the very bottom of the list, out of the active category groups. The section SHALL be collapsed by default and expandable.

#### Scenario: Bought and absent items leave the active groups
- **GIVEN** a shopper list with active, bought, and absent items
- **THEN** active items remain grouped by category in route order at the top
- **AND** bought and absent items appear only in the done section at the bottom

#### Scenario: Done section collapsed by default
- **WHEN** the shopper opens a list that has done items
- **THEN** the done section is collapsed and shows a count of done items
- **AND** the shopper can expand it to see the items

#### Scenario: No done section when nothing is done
- **GIVEN** a list where no item is bought or absent
- **THEN** no done section is shown and all items remain in their category groups

#### Scenario: Restoring an item returns it to its category group
- **GIVEN** an item in the done section
- **WHEN** the shopper unmarks it bought/absent so it is neither
- **THEN** it returns to its category group in the active area

#### Scenario: Manager view is unaffected
- **WHEN** a manager views the list
- **THEN** the done-section grouping does not apply; the manager view keeps its existing grouping and ordering

---

### Requirement: Manager sees which items were not found

In the manager view, items marked absent SHALL be shown with a distinct "not found" badge, in place within their category group, so the manager can see what the shopper missed.

#### Scenario: Absent item is badged for the manager
- **GIVEN** a list containing an item marked absent
- **WHEN** a manager views the list
- **THEN** that item is shown with a "not found" badge
- **AND** it stays in its category group rather than moving or being hidden

#### Scenario: Bought items are not badged
- **GIVEN** a list containing a bought item and an absent item
- **WHEN** a manager views the list
- **THEN** only the absent item carries the "not found" badge

#### Scenario: Badge clears when the item is resolved
- **GIVEN** an item shown to the manager with a "not found" badge
- **WHEN** the item is later marked bought, or the absent mark is removed
- **THEN** the badge is no longer shown

---

### Requirement: Category grouping

Items SHALL be displayed grouped by their assigned category. When the list has an associated shop with a configured aisle order, the groups SHALL follow that shop's order automatically; otherwise they follow the default category sort order.

#### Scenario: Items grouped by category in list detail
- **GIVEN** a list with items assigned to "Dairy" and "Vegetables"
- **THEN** items appear under two category headers in that order (by category sort_order)

#### Scenario: Uncategorized items shown at end
- **GIVEN** a list where some items have no category assigned
- **THEN** those items appear under an "Uncategorized" group after all categorized groups

#### Scenario: Shop-specific category order applied when shop is selected
- **GIVEN** the manager has configured a custom aisle order for a shop
- **WHEN** the shopper selects that shop on the list detail page
- **THEN** category groups are reordered to match that shop's aisle layout
- **AND** the selection is persisted on the list, so it still applies on the next visit

#### Scenario: Category order follows the list's shop automatically
- **GIVEN** the list has an associated shop for which the manager configured a custom aisle order
- **WHEN** the list is viewed
- **THEN** category groups are ordered to match that shop's aisle layout without the user selecting the shop each visit

#### Scenario: Default order when the list has no shop
- **GIVEN** the list has no associated shop, or its shop has no configured aisle order
- **WHEN** the list is viewed
- **THEN** category groups follow the default category sort order

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
- **WHEN** the item is added **via the paste preview or receipt processing**
- **THEN** the system requests a common-sense unit and a category **chosen from the family's own category names** (the AI is given the family's categories and must return one of them or none — it never invents a name)
- **AND** the returned category name is matched to its category row using Cyrillic-safe (Go-lowercased) comparison, not SQL `LOWER()`

#### Scenario: Manual add never waits on AI
- **GIVEN** no history exists for the item name
- **AND** the AI service is available
- **WHEN** the item is added directly to a list (single or bulk add, not via the paste preview)
- **THEN** no AI request is made and the item keeps unit `pcs` and remains uncategorized
- **AND** the add completes with no added latency versus today

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

