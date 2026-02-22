---
description: Analyze session to identify new facts, preferences, or updates and save to topical memory.
---

# Learn Prompt

Analyze the current session to identify any new facts, preferences, or project updates that differ from your existing knowledge. Save these specific updates into the relevant topical memory files in `.agent/memory/`.

## Instructions

1.  **Analyze Session**: Review the conversation history and actions taken in the current session.
2.  **Identify Updates**:
    -   **Facts**: New information about codebase, architecture, or tools.
    -   **Preferences**: User UI choices, coding styles, or workflow habits.
    -   **Project Updates**: Completed milestones, new features, or logical changes.
3.  **Cross-Reference**: Compare these findings with the existing memory in `.agent/memory/`.
4.  **Update Memory**:
    -   Categorize the update into one of the existing files: `architecture.md`, `logic.md`, `frontend.md`, or `testing.md`.
    -   If the update doesn't fit, create a new topical file (e.g., `api.md`).
    -   Update the relevant file by adding a concise entry. Prefer rules and logic over session history.
    -   Ensure formatting remains consistent.
    -   Do not duplicate information already present.
