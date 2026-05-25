# Feature: Categories

## Requirement: Manage categories
The manager can create, edit, reorder, and delete categories from the Settings page.

### Scenario: Create category
- **WHEN** the manager submits a new category with a name and emoji
- **THEN** the category appears in the category list with the assigned emoji and name

### Scenario: Create category without emoji shows no emoji
- **GIVEN** the manager creates a category named "Dairy" without selecting an emoji
- **THEN** the frontend displays the category name only, with no emoji

### Scenario: Edit category name
- **WHEN** the manager changes a category's name
- **THEN** the new name is reflected immediately in item group headers on all lists

### Scenario: Edit category emoji
- **WHEN** the manager changes a category's emoji
- **THEN** the new emoji is reflected immediately in the category list and item headers

### Scenario: Delete category
- **WHEN** the manager deletes a category
- **THEN** the category is removed and items that used it become uncategorized (category_id = null)

### Scenario: Delete does not remove items
- **GIVEN** 5 items are assigned to "Dairy"
- **WHEN** the manager deletes the "Dairy" category
- **THEN** those 5 items remain on their lists but appear under "Uncategorized"

---

## Requirement: Category sort order
Categories have a defined display order that controls grouping in list detail.

### Scenario: Items grouped by category sort order
- **GIVEN** "Vegetables" has sort_order=1 and "Dairy" has sort_order=2
- **THEN** items in "Vegetables" appear above items in "Dairy" in the list detail view

### Scenario: Reorder categories
- **WHEN** the manager drags "Dairy" above "Vegetables"
- **THEN** list detail now shows Dairy items before Vegetables items

### Scenario: New category gets next sort order
- **GIVEN** existing categories have sort orders 1, 2, 3
- **WHEN** the manager creates a new category
- **THEN** the new category gets sort_order=4

---

## Requirement: Category assignment on items

### Scenario: Assign category when adding item
- **GIVEN** categories exist
- **WHEN** the manager adds a new item and selects a category in the ConfirmSheet
- **THEN** the item appears under that category group in the list

### Scenario: Change item category
- **WHEN** the manager changes the category of an existing item
- **THEN** the item immediately moves to the new category group in the list view
