# Architecture & Data Management

## 1. System Components
- **Backend**: Go service with SQLite database.
- **Frontend**: React (Vite) PWA.
- **Data Store**: `kincart-data/kincart.db`.

## 2. Multi-Family Isolation
- **Strict Isolation**: Personal data (Lists, Items, Categories, Shops) must NOT be accessible or modifiable by members of other families.
- **Scoping**: All queries and mutations must be scoped by `family_id` retrieved from the JWT context.
- **Validation**: Enforce category and shop ownership checks during creation/update of dependent resources (e.g., items, lists).

## 3. Background Processing
- **Queue Pattern**: Resources like Receipts are saved with `Status="new"` and processed asynchronously.
- **Worker**: A background goroutine (running every 10 minutes) handles AI parsing tasks when API keys are available.

## 4. Database Patterns (SQLite)
- **Date Comparison**: ALWAYS use `date(column_name)` when comparing timestamps against date strings in SQL queries (e.g., `date(start_date) <= '2026-02-22'`). SQLite compares timestamps as strings, which can lead to incorrect results if types are mixed.
