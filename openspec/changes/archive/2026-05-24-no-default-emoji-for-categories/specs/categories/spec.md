## MODIFIED Requirements

### Requirement: Manage categories
The manager can create, edit, reorder, and delete categories from the Settings page.

#### Scenario: Create category
- **WHEN** the manager submits a new category with a name and emoji
- **THEN** the category appears in the category list with the assigned emoji and name

#### Scenario: Create category without emoji shows no emoji
- **GIVEN** the manager creates a category named "Dairy" without selecting an emoji
- **THEN** the frontend displays the category name only, with no emoji

#### Scenario: Edit category name
- **WHEN** the manager changes a category's name
- **THEN** the new name is reflected immediately in item group headers on all lists

#### Scenario: Edit category emoji
- **WHEN** the manager changes a category's emoji
- **THEN** the new emoji is reflected immediately in the category list and item headers

#### Scenario: Delete category
- **WHEN** the manager deletes a category
- **THEN** the category is removed and items that used it become uncategorized (category_id = null)

#### Scenario: Delete does not remove items
- **GIVEN** 5 items are assigned to "Dairy"
- **WHEN** the manager deletes the "Dairy" category
- **THEN** those 5 items remain on their lists but appear under "Uncategorized"

## REMOVED Requirements

### Requirement: Keyword emoji fallback
**Reason**: Keyword guessing produces unreliable results and gives the appearance of configuration when none has been done. Categories without an explicit emoji should display no emoji rather than a guessed one.
**Migration**: No migration needed. Categories that previously displayed a keyword-matched emoji will now display no emoji. Managers who want an emoji must explicitly select one.
