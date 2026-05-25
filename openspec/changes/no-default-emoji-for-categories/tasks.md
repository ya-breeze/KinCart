## 1. Update utility

- [ ] 1.1 In `frontend/src/utils/categoryEmoji.js`, remove `CATEGORY_EMOJI_MAP` and rewrite `getCategoryEmoji` to return `icon.trim()` when a valid icon is present, otherwise `''`

## 2. Verify call sites render correctly with empty emoji

- [ ] 2.1 `ConfirmSheet.jsx` line 26 — selected-category badge: confirm empty string renders cleanly (no orphaned space)
- [ ] 2.2 `ConfirmSheet.jsx` line 168 — category picker list: confirm empty string before category name looks correct
- [ ] 2.3 `SettingsPage.jsx` line 376 — settings category list: confirm empty string before category name looks correct
- [ ] 2.4 `ListDetail.jsx` line 670 — category group header icon span: wrap in conditional so the span is omitted when empty
- [ ] 2.5 `ListDetail.jsx` line 688 — category group header name prefix: confirm empty string renders cleanly
- [ ] 2.6 `ListDetail.jsx` line 744 — inline category picker: confirm empty string before category name looks correct
- [ ] 2.7 `ListDetail.jsx` line 848 — frequent-items section: confirm empty string inside icon container renders cleanly

## 3. Update spec

- [ ] 3.1 Apply delta spec: replace the "Create category without emoji falls back to keyword icon" scenario with "Create category without emoji shows no emoji" in `openspec/specs/categories/spec.md` and remove the keyword-fallback requirement text

## 4. Tests

- [ ] 4.1 Update or add a unit test for `getCategoryEmoji` covering: explicit icon returned as-is, legacy `'package'` sentinel treated as empty, no icon returns `''`, name with matching keyword returns `''`
- [ ] 4.2 Run `make test-frontend` and confirm all tests pass
- [ ] 4.3 Deploy to WIP stack and run E2E tests: `BASE_URL=<wip-url> npx playwright test --reporter=line`
