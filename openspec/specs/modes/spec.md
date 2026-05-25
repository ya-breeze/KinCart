# Feature: Manager & Shopper Modes

## Requirement: Mode switching
Every user can toggle between Manager mode and Shopper mode at any time.

### Scenario: Mode toggle visible on Dashboard
- **THEN** a mode toggle control (button or switch) is visible in the Dashboard header

### Scenario: Mode persists during session
- **GIVEN** the user switches to Shopper mode
- **WHEN** the user navigates between pages and back to Dashboard
- **THEN** Shopper mode remains active

---

## Requirement: Manager mode capabilities
Manager mode provides full planning and administrative access.

### Scenario: Manager sees all list statuses on Dashboard
- **GIVEN** lists exist with statuses `preparing`, `ready for shopping`, and `completed`
- **THEN** all three groups are visible on the Manager Dashboard

### Scenario: Manager can create, edit, and delete lists
- **GIVEN** the user is in Manager mode
- **THEN** "New List", "Edit", "Duplicate", and "Delete" controls are available

### Scenario: Manager can add, edit, and delete items
- **GIVEN** the user is in Manager mode on a list detail page
- **THEN** the quick-add bar, frequent-item chips, and item edit/delete controls are visible

### Scenario: Manager can upload receipts
- **GIVEN** the user is in Manager mode and the list is `completed`
- **THEN** the "Upload Receipt" button is visible

### Scenario: Manager can access settings
- **GIVEN** the user is in Manager mode
- **THEN** the Settings page (categories, shops) is accessible

---

## Requirement: Shopper mode restrictions
Shopper mode provides a focused, read-only view optimized for in-store use.

### Scenario: Shopper sees only ready-for-shopping lists
- **GIVEN** the user is in Shopper mode
- **THEN** the Dashboard shows ONLY lists with status `ready for shopping`; preparing and completed lists are hidden

### Scenario: Shopper cannot add items
- **GIVEN** the user is in Shopper mode on a list detail page
- **THEN** the quick-add bar and frequent-item chips are NOT rendered

### Scenario: Shopper cannot edit or delete items
- **GIVEN** the user is in Shopper mode
- **THEN** item edit and delete controls are NOT visible

### Scenario: Shopper can see the Upload Receipt button (known gap)
- **GIVEN** the user is in Shopper mode on a list detail page
- **THEN** the "Upload Receipt" button IS visible (no manager-only guard in current code)
- **NOTE** This is a known UI gap — the intent is that receipt upload is a Manager task, but the button is not currently hidden in Shopper mode

### Scenario: Shopper can check off items
- **GIVEN** the user is in Shopper mode on a list detail page
- **THEN** item checkboxes are active and checking them updates `is_bought`

### Scenario: Shopper can complete a list
- **GIVEN** the user is in Shopper mode and a list has status `ready for shopping`
- **WHEN** the shopper clicks "Complete Shopping"
- **THEN** the list status is set to `completed` and the shopper is returned to the Dashboard

### Scenario: Shopper sees items sorted by aisle order when shop is selected
- **GIVEN** the user is in Shopper mode and a shop is selected
- **THEN** items are grouped in the shop's configured aisle order

---

## Requirement: Mode is client-side only
Mode does not affect server-side authorization — the same API endpoints serve both modes.

### Scenario: API does not reject shopper-mode requests from manager endpoints
- **GIVEN** the user is in Shopper mode
- **WHEN** the client makes a PATCH /api/items/:id request (e.g., to mark as bought)
- **THEN** the server processes it normally (no mode header required)
