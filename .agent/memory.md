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

## Session: 2026-01-29 - E2E Testing with Playwright

### 1. E2E Testing Infrastructure
- **Framework**: Playwright is used for End-to-End testing, configured in `e2e/playwright.config.ts`.
- **Test Location**: E2E tests are in `e2e/tests/` directory with specs: `smoke.spec.ts`, `lists.spec.ts`, `flyers.spec.ts`.
- **Execution**: Run via `make test-e2e` which now includes a Docker health check.
- **Configuration**:
    - Test timeout: 60 seconds (increased for slower environments)
    - Base URL: `http://localhost:80`
    - Tests run in parallel using 4 workers

### 2. Backend Test Data Seeding
- **Fact**: Backend seeds test data via environment variables in `docker-compose.yml`.
- **Implementation**: 
    - `KINCART_SEED_USERS`: Format `family:username:password` (e.g., `Smith:dad:pass1,Smith:mom:pass2`)
    - `KINCART_SEED_FLYERS`: Format `ShopName:Item1|Price1,Item2|Price2;ShopName2:...` (e.g., `Lidl:Milk|1.50,Bread|2.00`)
    - Backend function: `seedFlyersFromEnv()` in `internal/database/db.go`
- **Purpose**: Provides consistent test data without manual database setup.

### 3. UI Selector Best Practices
- **Lesson Learned**: Always verify UI selectors using browser inspection before writing tests.
- **Common Issues**:
    - Placeholder text differs from label text (e.g., item name input uses `placeholder="e.g. Organic Bananas"`, not "Item Name")
    - Button text vs. title attributes (e.g., "Back to Dashboard" button uses `title` attribute, not visible text in some contexts)
    - Mode labels default to "Shopper Mode" not "Manager Mode" on login
- **Solution**: Used `browser_subagent` to inspect actual DOM structure and confirm selectors.

### 4. Test Reliability Patterns
- **Explicit Waits**: Always add `{ timeout: 10000 }` or higher for asynchronous elements (e.g., API-loaded cards, dialog boxes).
- **State Transitions**: Wait for explicit state changes (e.g., URL changes, mode labels) before proceeding to next test step.
- **Dialog Handling**: Set up dialog handlers BEFORE triggering actions that create prompts (e.g., list creation).
- **Avoid Race Conditions**: Don't immediately switch modes after state-changing actions; allow UI to stabilize.

### 5. Makefile Enhancement
- **Preference**: E2E tests should fail fast with clear error messages if prerequisites aren't met.
- **Implementation**: Added Docker health check to `test-e2e` target that verifies `nginx` service is running before executing tests.
- **Error Message**: Provides actionable guidance: "Please run 'make docker-up' first to start the application."

### 6. Test Coverage Philosophy
- **Preference**: E2E tests should cover complete user flows, not isolated actions.
- **Implementation Examples**:
    - **Lists Flow**: Manager creates list → adds items → marks ready → Shopper shops → toggles items → completes shopping
    - **Flyers Flow**: Filter by shop → add item to new list → verify in dashboard
- **Reasoning**: Integration tests validate the entire workflow, catching issues in state transitions and cross-component interactions.

### 7. Final Validation Workflow
- **CRITICAL RULE**: ALWAYS run the following commands as final validation before completing work:
    1. `make test` - Run all backend and frontend unit tests
    2. `make test-e2e` - Run Playwright E2E tests
    3. `make lint` - Run linting for code quality
- **Action Required**: If any of these fail, fix the issues before considering the work complete.
- **Purpose**: Ensures code quality, prevents regressions, and validates that all changes integrate properly with the existing codebase.
