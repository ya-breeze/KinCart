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
