## MODIFIED Requirements

### Requirement: Category grouping

Items SHALL be displayed grouped by their assigned category. When the list has an associated shop with a configured aisle order, the groups SHALL follow that shop's order automatically; otherwise they follow the default category sort order.

#### Scenario: Items grouped by category in list detail
- **GIVEN** a list with items assigned to "Dairy" and "Vegetables"
- **THEN** items appear under two category headers in that order (by category sort_order)

#### Scenario: Uncategorized items shown at end
- **GIVEN** a list where some items have no category assigned
- **THEN** those items appear under an "Uncategorized" group after all categorized groups

#### Scenario: Category order follows the list's shop automatically
- **GIVEN** the list has an associated shop for which the manager configured a custom aisle order
- **WHEN** the list is viewed
- **THEN** category groups are ordered to match that shop's aisle layout without the user selecting the shop each visit

#### Scenario: Default order when the list has no shop
- **GIVEN** the list has no associated shop, or its shop has no configured aisle order
- **WHEN** the list is viewed
- **THEN** category groups follow the default category sort order
