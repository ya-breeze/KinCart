## ADDED Requirements

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
