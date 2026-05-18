## ADDED Requirements

### Requirement: Add-item UI hidden on completed list
When a shopping list has status `completed`, the manager view SHALL hide all item-addition UI elements so that the list cannot be accidentally extended after the shopping trip is done.

#### Scenario: Frequent-item chips hidden when completed
- **WHEN** the manager opens a list with status `completed`
- **THEN** the frequent-item chip grid SHALL NOT be rendered in the DOM

#### Scenario: Search/quick-add input hidden when completed
- **WHEN** the manager opens a list with status `completed`
- **THEN** the search/quick-add input bar SHALL NOT be rendered in the DOM

#### Scenario: Add-item UI visible on non-completed list
- **WHEN** the manager opens a list with status `preparing` or `ready for shopping`
- **THEN** the frequent-item chips and search/quick-add input bar SHALL be visible

#### Scenario: Add-item UI reappears after status change away from completed
- **WHEN** a list status is changed from `completed` back to `preparing`
- **THEN** the frequent-item chips and search/quick-add input SHALL become visible again immediately
