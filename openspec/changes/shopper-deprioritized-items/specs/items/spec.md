## ADDED Requirements

### Requirement: Mark item as absent (out of stock)

An item SHALL have an `is_absent` state, distinct from `is_bought`, that the shopper can set and clear. Absent means the item was not available in the store on this trip.

#### Scenario: Shopper marks an item absent
- **WHEN** the shopper taps the "mark absent" control on an active item
- **THEN** the item's `is_absent` is set to true and it is treated as de-prioritized

#### Scenario: Shopper unmarks an absent item
- **GIVEN** an item marked absent
- **WHEN** the shopper taps the control again
- **THEN** `is_absent` is set to false and the item returns to the active list

#### Scenario: Absent is independent of bought
- **GIVEN** an item marked absent
- **WHEN** the shopper marks it bought (e.g. found it after all)
- **THEN** both states are tracked independently and the item is shown as bought

#### Scenario: Absent and bought are visually distinguishable
- **GIVEN** the done section contains one bought and one absent item
- **THEN** the two are rendered with distinct styling so the shopper can tell them apart

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
- **THEN** the done-section grouping does not apply; the manager view is unchanged
