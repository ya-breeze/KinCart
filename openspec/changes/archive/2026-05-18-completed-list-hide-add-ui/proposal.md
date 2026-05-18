## Why

When a shopping list is marked as **completed**, adding new items is meaningless — the trip is done. Currently the frequent-item chips and the search/quick-add bar are visible and interactive on completed lists, cluttering the UI and inviting accidental edits.

## What Changes

- Hide the entire "bottom quick-add bar" (frequent-item chips + search input + autocomplete) when the list status is `completed`.
- The change is purely in the frontend (`ListDetail.jsx`) — no backend or API changes required.

## Capabilities

### New Capabilities
- `completed-list-read-only-ui`: When a list's status is `completed`, the manager view suppresses all add-item UI elements (frequent-item chip grid, search/quick-add input bar, autocomplete dropdown) so the page becomes read-only for item addition.

### Modified Capabilities
<!-- No existing spec-level requirements are changing — this is a new guard on an existing page. -->

## Impact

- **Frontend only**: `frontend/src/pages/ListDetail.jsx` — conditionalize the bottom quick-add bar render on `!isCompleted`.
- No API, model, or backend changes.
- The receipt-upload button (already gated on `isCompleted`) and the existing item list remain visible and functional.
