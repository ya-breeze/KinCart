# Testing Strategy

## 1. Frameworks
- **Backend**: Standard Go `testing` package with table-driven tests.
- **Frontend**: Vitest for unit tests, Playwright for E2E tests.

## 2. Playwright E2E Patterns
- **Isolation**: Unregister all Service Workers and clear `localStorage` in `beforeEach` to prevent state leak.
- **Wait Policy**: Prefer explicit visibility checks (`toBeVisible`) with custom timeouts (`{ timeout: 10000 }`) for API-dependent elements.
- **Auth Persistence**: Confirmation of `localStorage` token presence before navigating to protected routes prevents race conditions.

## 3. UI Selectors
- **Rules**: Favor data-testids or direct text selection over brittle CSS classes.
- **Common traps**:
    - Placeholder text vs Labels (Inputs often use specific placeholders like `"e.g. Organic Bananas"`).
    - Toggle states (Login defaults to Shopper Mode).

## 4. Test Environment
- **Docker**: Run via `make docker-up-e2e`.
- **Seeding**: Uses environment variables (`KINCART_SEED_USERS`, `KINCART_SEED_FLYERS`) in `docker-compose.e2e.yml`.
- **Pre-check**: Always verify the frontend is responsive before starting E2E suites.

## 5. Final Validation Check
CRITICAL: Before completing any task, run:
1. `make test` (Unit tests)
2. `make test-e2e` (E2E tests)
3. `make lint` (Code quality)
