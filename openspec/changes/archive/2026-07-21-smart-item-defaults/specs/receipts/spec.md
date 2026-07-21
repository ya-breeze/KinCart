## ADDED Requirements

### Requirement: Receipt-created items derive a sensible category

When a receipt item has no planned match and a new list item is created for it, the system SHALL derive that item's category from history/AI resolution instead of assigning the first category by sort order.

#### Scenario: New receipt item uses remembered category
- **GIVEN** a receipt item whose name exists in purchase history with a category
- **WHEN** a new list item is created from it
- **THEN** the new item is assigned the remembered category, not the first-by-sort-order category

#### Scenario: New receipt item with no history
- **GIVEN** a receipt item whose name has no history
- **WHEN** a new list item is created from it and AI is available
- **THEN** the item's category is set from the AI common-sense guess, or left uncategorized if none is usable

#### Scenario: An uncategorized item stays visible on the list
- **GIVEN** a new list item was left uncategorized
- **WHEN** the list is viewed
- **THEN** the item carries the zero UUID as its category and is grouped under the existing "Uncategorized" heading, not hidden from the list

#### Scenario: Category history is updated from the receipt
- **WHEN** a receipt item is matched to a planned item or creates a new item
- **THEN** the item's category is recorded to history so later additions of the same name reuse it
