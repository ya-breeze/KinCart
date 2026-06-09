# Shopping Lists

## Purpose
Create and manage shopping lists through their status lifecycle and dashboard presentation.

## Requirements

### Requirement: Create list

A user SHALL be able to create a new shopping list with a title.

#### Scenario: Create list
- **WHEN** the user clicks "New List" and submits a title
- **THEN** a new list is created with status `preparing` and appears on the Dashboard

#### Scenario: Create list without title
- **WHEN** the user submits the create-list form with an empty title
- **THEN** the list is not created and an error is shown

---

### Requirement: List status lifecycle

A list SHALL move through three statuses: `preparing` → `ready for shopping` → `completed`.

#### Scenario: Status badge shows current status
- **GIVEN** a list exists with status `preparing`
- **THEN** the list card shows a badge labeled "PREPARING"

#### Scenario: Manager advances status by clicking badge
- **GIVEN** a list with status `preparing`
- **WHEN** the manager clicks the status badge
- **THEN** the status advances to `ready for shopping`

#### Scenario: Status cycles: preparing → ready → completed → preparing
- **GIVEN** a list with status `completed`
- **WHEN** the manager clicks the status badge
- **THEN** the status returns to `preparing`

#### Scenario: CompletedAt is set automatically
- **WHEN** a list's status is changed to `completed`
- **THEN** the server sets `completed_at` to the current timestamp automatically

#### Scenario: CompletedAt is reset on re-completion
- **GIVEN** a list with status `completed` and a `completed_at` timestamp
- **WHEN** the status is changed to `preparing` and then back to `completed`
- **THEN** `completed_at` is set to the new completion time (the old value is cleared when the status leaves `completed`)

---

### Requirement: Dashboard list grouping

Lists SHALL be grouped and ordered by status on the Dashboard.

#### Scenario: Lists grouped by status
- **GIVEN** lists exist with statuses `preparing`, `ready for shopping`, and `completed`
- **THEN** the Dashboard shows three sections: PREPARING, READY FOR SHOPPING, COMPLETED

#### Scenario: Completed lists sorted newest first
- **GIVEN** multiple completed lists
- **THEN** they are ordered by `completed_at` descending (most recently completed first)

#### Scenario: Shopper sees only ready-for-shopping lists
- **GIVEN** the user is in Shopper mode
- **THEN** the Dashboard shows only lists with status `ready for shopping`

---

### Requirement: List card summary

Each list card on the Dashboard SHALL show a summary of the list's state.

#### Scenario: Card shows item count and progress
- **GIVEN** a list with 5 items, 2 of which are marked as bought
- **THEN** the list card shows "2/5" or a 40% progress bar

#### Scenario: Card shows estimated amount
- **GIVEN** a list with an estimated amount set
- **THEN** the list card shows the estimated amount

---

### Requirement: Duplicate list

A manager SHALL be able to duplicate an existing list to reuse it.

#### Scenario: Duplicate creates a copy
- **WHEN** the manager clicks Duplicate on a list
- **THEN** a new list is created with the same title (suffixed " (Copy)"), all the same items, and status `preparing`

#### Scenario: Duplicated items are not bought
- **GIVEN** the source list has items marked as bought
- **WHEN** the list is duplicated
- **THEN** all items in the copy have `is_bought = false`

---

### Requirement: Delete list

A manager SHALL be able to delete a list at any status.

#### Scenario: Delete list removes it from Dashboard
- **WHEN** the manager deletes a list
- **THEN** the list and all its items no longer appear anywhere in the app

#### Scenario: Delete requires confirmation
- **WHEN** the manager clicks Delete on a list
- **THEN** a confirmation prompt is shown before deletion proceeds
