---
description: Analyze session to identify new facts, preferences, or updates and save to memory.
---

# Learn Prompt

Analyze the current session to identify any new facts, preferences, or project updates that differ from your existing knowledge. Save these specific updates into your 'memory' in `.agent/memory.md` for future context.

## Instructions

1.  **Analyze Session**: Review the conversation history and actions taken in the current session.
2.  **Identify Updates**:
    -   **Facts**: New information about the codebase, architecture, or tools.
    -   **Preferences**: User-specified coding styles, UI choices, or workflow habits.
    -   **Project Updates**: Completed milestones, new features, or logical changes.
3.  **Cross-Reference**: Compare these findings with the existing entries in `.agent/memory.md`.
4.  **Update Memory**:
    -   If new information is found, append it to `.agent/memory.md` under a new session heading (e.g., `## Session: [YYYY-MM-DD] - [Brief Title]`).
    -   Ensure the formatting remains consistent with the existing structure.
    -   Do not duplicate information already present in the memory.
