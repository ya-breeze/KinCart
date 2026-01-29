# Project Memory: KinCart

This file tracks key project updates, logic decisions, and user preferences discovered during development sessions.

## Session: 2026-01-28 - Flyer Linking & Logic Refinement

### 1. Flyer Item Linking Logic
- **Fact**: Items in a shopping list can be linked to parsed flyer deals (`flyer_item_id`).
- **Logic**: Linked items have protected fields (`name`, `price`, `local_photo_path`) to prevent accidental overrides of deal data.
- **Implementation**: 
    - Backend: `UpdateItem` and `AddItemPhoto` handlers enforce these restrictions.
    - Frontend: `ListDetail.jsx` disables these fields in the UI and provides an "Unlink" button to remove the `flyer_item_id`.

### 2. Database & Search Logic (SQLite)
- **Fact**: The database is located at `kincart-data/kincart.db`.
- **Lesson Learned**: SQLite compares timestamps as strings. A comparison like `'2026-01-28 00:00:00' <= '2026-01-28'` is FALSE.
- **Rule**: ALWAYS use `date(column_name)` when comparing timestamps against date strings in SQL queries for flyer items (e.g., in `GetFlyerItems`).

### 3. User Preferences & UI Rules
- **Preference**: Categories are **optional**. Users should be able to add items without selecting a category. These items appear as "Uncategorized".
- **Preference**: Maintain high visual quality using `lucide-react` icons and "Sale Deal" badges for flyer-linked items.

### 4. Technical Configuration
- **Retailer Crawling**: Supports Albert, Billa, Tesco, Kaufland, Globus, and Lidl via `akcniceny.cz`.
- **Parser**: Uses Gemini 3.0 Flash for multi-modal parsing of flyer images/PDFs.

## Session: 2026-01-29 - Flyer UX & Performance Optimization

### 1. Performance: On-Demand Image Loading
- **Fact**: The `FlyerItemsPage` can contain hundreds of items depending on active flyers.
- **Preference**: Images should be loaded on demand rather than all at once to maintain fast initial page loads.
- **Implementation**: 
    - Created `LazyImage.jsx` component using the `IntersectionObserver` API.
    - Set a `200px` root margin to trigger loading slightly before the image becomes visible.
    - Added a fade-in animation and loading indicators for a premium feel.

### 2. UI/UX: Filter Reset
- **Feature**: Added a clear (X) button to the search input in `FlyerItemsPage.jsx`.
- **Logic**: The button is conditionally rendered only when the filter query is not empty.
- **UX Rule**: Providing quick "reset" actions for search/filters is preferred for high-interaction pages.


## Session: 2026-01-29 - Data Isolation & Security

### 1. Multi-Family Data Isolation
- **Fact**: KinCart is designed with strict isolation between different families. Personal data (Lists, Items, Categories, Shops) must NOT be accessible or modifiable by members of other families.
- **Logic**: All personal data queries and mutations must be scoped by `family_id` retrieved from the JWT context.
- **Implementation**: 
    - Created `validateItemsFamily` helper in `internal/handlers/utils.go` to verify category ownership.
    - Added nested item validation in `CreateList` and `UpdateList` to prevent cross-family category injection.
    - Scoped `DeleteCategory` to ensure item resets only affect the current family.
    - Enforced shop ownership checks for category ordering.

### 2. Testing Framework
- **Fact**: Handlers are validated using `internal/handlers/isolation_test.go` which uses a shared in-memory SQLite database (`file::memory:?cache=shared`).
- **Preference**: Use specific test cases to verify security boundaries (e.g., cross-family category linking attempts).
