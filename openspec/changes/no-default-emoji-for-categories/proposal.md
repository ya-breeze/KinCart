## Why

Categories currently display a keyword-based fallback emoji (e.g., 🥛 for "Dairy") when no emoji is explicitly chosen. This auto-guessing is unreliable and misleading — it makes the UI appear configured when it isn't, and silently assigns wrong icons to categories whose names don't match the keyword list.

## What Changes

- Remove the keyword-based emoji fallback logic from the frontend
- Categories created without an emoji display no emoji (or a neutral placeholder) instead of a guessed one
- The "Create category without emoji falls back to keyword icon" scenario is replaced by "Create category without emoji shows no emoji"

## Capabilities

### New Capabilities

*(none)*

### Modified Capabilities

- `categories`: The requirement for keyword-based emoji fallback is being removed; a category with no emoji must not auto-assign one

## Impact

- Frontend: emoji fallback/keyword-matching code removed from category display components
- Spec: `categories/spec.md` scenario updated to reflect new no-emoji behavior
- No backend or API changes required
