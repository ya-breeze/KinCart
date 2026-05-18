## Context

`ListDetail.jsx` is the single-page view for a shopping list. It has two render branches: manager view (lines ~587+) and shopper view. The manager view always renders the bottom quick-add bar (frequent-item chips + search input) regardless of list status. The `isCompleted` flag already exists at line 589 and is used only to gate the receipt-upload button.

## Goals / Non-Goals

**Goals:**
- Hide the entire bottom quick-add bar (chips + input + autocomplete) when `isCompleted === true`
- Keep all existing item rows visible (the completed list is read-only, not invisible)
- Keep the receipt-upload button and status badge fully functional

**Non-Goals:**
- No changes to backend, API, or data model
- No hiding of items already on the list
- No role-based variation (manager is the only role that sees the add bar — shopper never does)
- No animation or transition on hide

## Decisions

**Wrap the quick-add bar in a `!isCompleted` guard**
The bottom quick-add `<div>` (lines 777–866) already sits inside the manager render branch. A single conditional `{!isCompleted && ( … )}` around the outer div eliminates all three sub-components (chips, search input, autocomplete) in one place. This is the minimal, readable change — no state/ref cleanup needed because React unmounts the whole subtree.

Alternative considered: disable inputs rather than hide. Rejected — disabled inputs on a completed list still take up space and invite confusion; hiding is cleaner UX.

## Risks / Trade-offs

- [Risk]: `chipsContainerRef` and `queryInputRef` are React refs attached to DOM nodes inside the hidden subtree. When `isCompleted` is true, the nodes are unmounted so the refs become `null`. The existing code does not read these refs during render or in effects that run when status is `completed`, so no null-deref risk. → No mitigation needed.
- [Trade-off]: If a user changes list status from `completed` back to an earlier status (possible via the status cycle button), the add bar reappears immediately — no stale state to clear. This is the correct behavior.
