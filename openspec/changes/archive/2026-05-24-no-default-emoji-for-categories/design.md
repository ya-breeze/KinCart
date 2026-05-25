## Context

`frontend/src/utils/categoryEmoji.js` exports `getCategoryEmoji(name, icon)` which is called in four files:

- `ConfirmSheet.jsx` (×2) — category picker and selected-category badge
- `SettingsPage.jsx` (×1) — settings category list
- `ListDetail.jsx` (×4) — category group headers and the frequent-items section

Current logic: if `icon` is set and not the legacy `'package'` sentinel, return it; otherwise do keyword matching on `name`; final fallback is `'📦'`.

The change removes the keyword matching table and the `'📦'` fallback. When no explicit icon is stored, callers receive an empty string and must handle the no-emoji case themselves (typically by rendering nothing or a neutral placeholder).

## Goals / Non-Goals

**Goals:**
- `getCategoryEmoji` returns `''` when no explicit icon is set (no keyword guessing, no default box)
- All call sites render gracefully when the emoji is empty
- The legacy `'package'` sentinel is still treated as empty

**Non-Goals:**
- Changing the backend model or API — `icon` stays a nullable string column
- Adding a new "no icon" indicator icon — blank rendering is sufficient
- Migrating existing rows that have a guessed emoji stored from before this change

## Decisions

**Remove `CATEGORY_EMOJI_MAP` entirely** rather than keeping it dead code.
No other call site uses it; deleting it reduces surface area and avoids confusion.

**Return `''` (empty string) instead of `null`/`undefined`.**
All callers do `{getCategoryEmoji(...)} {cat.name}` — rendering an empty string produces clean output without needing JSX guards. Returning `null` would require callers to add explicit checks.

**No visual placeholder** (no grey box, no `?`, no neutral icon).
The category name alone is sufficient to identify the group. Adding a placeholder risks looking like a broken/missing emoji to the user.

## Risks / Trade-offs

- [UI regression at call sites that always prepend the emoji] → Mitigation: audit all 7 call sites and verify they render cleanly with an empty string (no orphaned space before the name)
- [Existing categories that previously showed a keyword-matched emoji will now show nothing] → Accepted: this is the desired behavior; the spec scenario is being replaced, not fixed
