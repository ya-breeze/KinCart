## 1. Frontend Change

- [x] 1.1 In `frontend/src/pages/ListDetail.jsx`, wrap the bottom quick-add bar `<div>` (the one starting at line ~777 with the frequent-item chips and search input) in a `{!isCompleted && ( … )}` conditional so it is not rendered when the list status is `completed`

## 2. Testing

- [x] 2.1 Manually verify on the WIP stack: open a completed list as manager — confirm chips and input bar are gone; open a non-completed list — confirm they are still present
- [x] 2.2 Add a Playwright E2E test in `e2e/tests/` that: logs in as manager, marks a list as completed, and asserts the quick-add input is not visible
