# Feature: Shops & Aisle Routing

## Requirement: Manage shops
The manager can create, rename, and delete shops from the Settings page.

### Scenario: Create shop
- **WHEN** the manager submits a shop name
- **THEN** the shop appears in the shops list and is available for selection on list detail and receipt upload

### Scenario: Edit shop name
- **WHEN** the manager renames a shop
- **THEN** the new name is reflected everywhere the shop is referenced (aliases, category orders, receipt matches)

### Scenario: Delete shop
- **WHEN** the manager deletes a shop
- **THEN** the shop is removed along with all its ShopCategoryOrder records

### Scenario: Delete shop does not affect aliases (known gap)
- **GIVEN** aliases reference the deleted shop via shop_id
- **WHEN** the shop is deleted
- **THEN** those aliases remain in the DB but their shop_id now points to a deleted record
- **NOTE** The delete handler only removes ShopCategoryOrder rows; it does NOT null-out alias shop_id references. Aliases effectively become shop-agnostic in practice (shop lookup will return nothing) but the FK is left dangling.

---

## Requirement: Aisle routing (shop category order)
Each shop can have a custom category order that defines the aisle layout in-store.

### Scenario: Configure aisle order for a shop
- **GIVEN** a shop "Lidl" exists with categories "Dairy", "Meat", "Vegetables"
- **WHEN** the manager sets the order to Vegetables → Dairy → Meat
- **THEN** that order is saved for "Lidl"

### Scenario: Default category order used when no shop is selected
- **GIVEN** the user has not selected a shop on the list detail page
- **THEN** items are grouped by the global category sort_order

### Scenario: Shop-specific order applied when shop is selected
- **GIVEN** the "Lidl" shop has a custom aisle order
- **WHEN** the user selects "Lidl" on the list detail page
- **THEN** category groups are displayed in Lidl's configured order

### Scenario: Categories not in shop order appear at end
- **GIVEN** "Lidl" has an order for "Dairy" and "Meat" but the list also has "Bakery" items
- **WHEN** "Lidl" is selected
- **THEN** "Dairy" and "Meat" groups appear first in configured order, "Bakery" appears at end

### Scenario: Removing all categories from shop order reverts to global order
- **GIVEN** "Lidl" had a custom order
- **WHEN** the manager clears all category entries from Lidl's order
- **THEN** the list reverts to the global category sort_order

---

## Requirement: Shop selection on list detail

### Scenario: Shop selector visible to manager and shopper
- **THEN** the shop dropdown is visible on the list detail page for both Manager and Shopper

### Scenario: Shop selection persists during the session
- **GIVEN** the user selects "Tesco" on a list
- **WHEN** the user scrolls or checks off items
- **THEN** "Tesco" remains selected and the aisle order stays applied
