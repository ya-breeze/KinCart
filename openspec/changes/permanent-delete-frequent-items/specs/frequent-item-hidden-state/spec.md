## ADDED Requirements

### Requirement: Deleting a frequent item chip hides it permanently
When a user deletes a frequent item chip, the system SHALL mark it as hidden rather than deleting it. A hidden item SHALL NOT appear in the chip grid. Future item additions or receipt processing with the same name SHALL NOT restore the chip.

#### Scenario: Chip disappears after deletion
- **WHEN** a user clicks the delete (X) button on a frequent item chip
- **THEN** the chip is removed from the grid immediately and does not reappear on subsequent page loads

#### Scenario: Adding the same item again does not restore the chip
- **WHEN** a user has deleted a frequent item chip named "Milk"
- **AND** the user later adds an item named "Milk" (or any case variant) to a shopping list
- **THEN** the "Milk" chip SHALL NOT reappear in the chip grid

#### Scenario: Receipt processing does not restore a hidden chip
- **WHEN** a user has deleted a frequent item chip named "Milk"
- **AND** a receipt is processed that matches or creates an item named "Milk"
- **THEN** the "Milk" chip SHALL NOT reappear in the chip grid

#### Scenario: Case variants are treated as the same item
- **WHEN** a frequent item chip named "Milk" exists
- **AND** a user adds an item named "milk" (lowercase) to a list
- **THEN** the system SHALL increment the existing "Milk" frequency record rather than creating a new "milk" record

### Requirement: Users can restore hidden frequent items
The system SHALL provide a settings page listing all hidden frequent items, where users can restore individual items.

#### Scenario: Hidden items appear in the settings page
- **WHEN** a user navigates to the Frequent Items settings page
- **THEN** all items previously deleted from the chip grid SHALL be listed

#### Scenario: Restoring a hidden item makes the chip reappear
- **WHEN** a user clicks "Restore" on a hidden frequent item
- **THEN** the item is removed from the hidden list
- **AND** the chip SHALL reappear in the chip grid on the shopping list page

#### Scenario: No hidden items shows an empty state
- **WHEN** a user navigates to the Frequent Items settings page
- **AND** no items have been hidden
- **THEN** the page SHALL display an empty state message indicating there are no hidden items
