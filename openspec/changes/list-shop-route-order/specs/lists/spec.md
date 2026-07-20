## ADDED Requirements

### Requirement: List shop association

A shopping list SHALL have an optional associated shop (`shop_id`), which determines the category ordering used when viewing the list. The association is nullable; a list may have no shop. The manager sets the shop when creating the list; the shopper may change it from the list detail view while shopping.

#### Scenario: Create a list with a shop
- **WHEN** the user creates a list and selects a shop in the create-list dialog
- **THEN** the list is created with `shop_id` set to that shop
- **AND** the shop belongs to the user's family

#### Scenario: Create a list without a shop
- **WHEN** the user creates a list and leaves the shop selector on "No shop / default order"
- **THEN** the list is created with a null `shop_id`

#### Scenario: Shop must belong to the family
- **WHEN** a create or update request supplies a `shop_id` that does not belong to the requester's family
- **THEN** the request is rejected with a validation error and the list's shop is not changed

#### Scenario: Shopper changes a list's shop
- **WHEN** the shopper selects a different shop for a list that is ready for shopping
- **THEN** the list's `shop_id` is persisted via the list update endpoint
- **AND** subsequent loads of the list reflect the new shop

#### Scenario: Clear a list's shop
- **WHEN** the shopper sets a list's shop selector back to "Default Order"
- **THEN** the list's `shop_id` is persisted as null

#### Scenario: Duplicating a list keeps its shop
- **WHEN** a list with an associated shop is duplicated
- **THEN** the copy carries the same `shop_id`, so its route order still applies

#### Scenario: List responses include the shop
- **WHEN** a client fetches a list or the list collection
- **THEN** each list's `shop_id` is present in the response (null when unset)

---

### Requirement: List category order follows the list's shop

When a list has an associated shop with a configured category route order, the list view SHALL group items by category in that shop's route order. Otherwise it SHALL use the default category sort order.

#### Scenario: Shopper opens a list with a shop configured
- **GIVEN** a list whose `shop_id` refers to a shop with a saved category route order
- **WHEN** the shopper opens the list
- **THEN** the items are grouped by category in that shop's route order without the shopper selecting a shop manually

#### Scenario: List has no shop
- **GIVEN** a list with a null `shop_id`
- **WHEN** the list is viewed
- **THEN** categories are ordered by the default `Category.SortOrder`

#### Scenario: Shop has no configured route order
- **GIVEN** a list whose shop has no rows in the per-shop category order
- **WHEN** the list is viewed
- **THEN** categories are ordered by the default `Category.SortOrder`

#### Scenario: Categories with no shop-specific order fall to the end
- **GIVEN** a list whose shop orders only some categories
- **WHEN** the list is viewed
- **THEN** ordered categories appear first in the shop's order and any remaining categories follow in default order
