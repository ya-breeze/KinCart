# Feature: Paste-to-List

## Requirement: Parse freeform text into items
The manager can paste a freeform shopping list and have it parsed into structured items.

### Scenario: Open paste panel
- **GIVEN** the manager is on the list detail page of a non-completed list
- **WHEN** the manager clicks the "Paste Items" button (or pastes multi-line text into the quick-add bar)
- **THEN** a panel or modal opens with a textarea for entering the text

### Scenario: Parse produces a preview of items
- **GIVEN** the manager has entered multi-line text such as "Apple 2kg\nMilk 1L\nBread"
- **WHEN** the manager clicks "Parse"
- **THEN** a preview shows three rows: Apple (qty 2, unit kg), Milk (qty 1, unit l), Bread (qty 1, unit pcs)

### Scenario: Parsed items can be edited before adding
- **GIVEN** the parse preview is shown
- **WHEN** the manager changes the quantity or unit on a row
- **THEN** the edited value is used when the items are added to the list

### Scenario: Individual parsed items can be removed
- **GIVEN** the parse preview is shown
- **WHEN** the manager removes one row
- **THEN** that item is not added when "Add All" is clicked

### Scenario: Add All bulk-creates items
- **WHEN** the manager clicks "Add All"
- **THEN** all remaining preview items are added to the list in a single batch

### Scenario: Price pre-filled from alias history when shop is selected
- **GIVEN** the manager selects a shop before parsing
- **AND** the family has an alias for a parsed item at that shop
- **THEN** the parsed item row shows the shop-specific price as a suggestion

---

## Requirement: Gemini-powered parsing
Parsing uses the Gemini API to handle multilingual, freeform, and retail-style input.

### Scenario: Handles quantity-first format
- **GIVEN** input "2 яблока 1.50"
- **THEN** the parsed item has name="яблока", quantity=2, price=1.50

### Scenario: Handles retail promo notation
- **GIVEN** input "кефир 4+2"
- **THEN** two items are parsed: "кефир" (qty 4) and "кефир" (qty 2), or a single item with qty 6

### Scenario: Graceful degradation without Gemini API key
- **GIVEN** `GEMINI_API_KEY` is not configured
- **WHEN** the manager clicks "Parse"
- **THEN** a 503 error is shown and no items are added

---

## Requirement: Text size limits

### Scenario: Oversized input rejected
- **WHEN** the manager submits text larger than 100 KB
- **THEN** the server returns 413 and no parsing is attempted

### Scenario: Empty text rejected
- **WHEN** the manager submits an empty or whitespace-only string
- **THEN** an error is shown ("receipt_text is required")
