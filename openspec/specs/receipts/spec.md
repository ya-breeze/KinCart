# Feature: Receipt Scanning & Matching

## Requirement: Upload a receipt
The manager can upload a receipt after shopping to record actual prices.

### Scenario: Receipt upload button visible only on completed list
- **GIVEN** a list with status `preparing` or `ready for shopping`
- **THEN** the "Upload Receipt" button is NOT visible

### Scenario: Receipt upload button visible on completed list
- **GIVEN** a list with status `completed`
- **THEN** the "Upload Receipt" button is visible

### Scenario: Upload an image receipt
- **WHEN** the manager uploads a JPEG, PNG, or WebP image (≤ 10 MB)
- **THEN** the file is accepted and processing begins

### Scenario: Upload a text receipt
- **WHEN** the manager uploads a `.txt` file (≤ 100 KB)
- **THEN** the file is accepted and processing begins

### Scenario: Upload rejected for oversized image
- **WHEN** the manager uploads an image larger than 10 MB
- **THEN** an error is shown and the file is not saved

### Scenario: Upload rejected for unsupported file type
- **WHEN** the manager uploads a `.pdf` or other unsupported format
- **THEN** an error is shown and the file is not saved

### Scenario: Processing indicator shown while receipt is being parsed
- **WHEN** a receipt is uploaded and being processed
- **THEN** a status indicator ("Processing receipt…") is visible to the manager

---

## Requirement: Receipt parsing outcome
After upload the server parses the receipt and determines a review status.

### Scenario: Parsed receipt with unmatched items opens match modal
- **GIVEN** Gemini successfully parses the receipt
- **AND** some items have no automatic match or planned items remain unmatked
- **WHEN** processing completes with status `pending_review`
- **THEN** the ReceiptMatchModal opens automatically

### Scenario: Fully auto-matched receipt skips manual review
- **GIVEN** all receipt items have been matched automatically with high confidence
- **WHEN** processing completes with status `parsed`
- **THEN** the match modal does NOT open; a success toast is shown instead

### Scenario: Receipt queued when Gemini unavailable
- **GIVEN** `GEMINI_API_KEY` is not set
- **WHEN** a receipt is uploaded
- **THEN** status is set to `queued` and a toast informs the manager it will be processed later

---

## Requirement: Receipt match review
The manager reviews AI-suggested item matches before they are applied.

### Scenario: Auto-matched items shown pre-confirmed
- **GIVEN** a receipt item was matched via alias with confidence ≥ 90%
- **THEN** the row shows the match as confirmed (green) without manager action

### Scenario: Suggested items shown for low-confidence matches
- **GIVEN** Gemini returns a match with confidence < 90%
- **THEN** the row shows the suggestion and the manager must manually accept or change it

### Scenario: Unmatched receipt item can be linked to a planned item
- **GIVEN** a receipt item has no match
- **WHEN** the manager selects a planned item from the dropdown
- **THEN** the receipt item is linked to that planned item

### Scenario: Receipt item can be linked to an already-bought item
- **GIVEN** a planned item is marked as bought but not yet receipt-linked
- **WHEN** the manager selects it from the "already bought" section
- **THEN** the receipt item is linked to it (no duplicate created)

### Scenario: Receipt item can be dismissed
- **WHEN** the manager dismisses a receipt item
- **THEN** it is marked as "extra" (not on the planned list) and no planned item is created

### Scenario: Confirm button disabled until all items have a decision
- **GIVEN** at least one receipt item is still unmatched and undismissed
- **THEN** the "Confirm" button is disabled

### Scenario: Confirm applies all decisions
- **WHEN** the manager clicks "Confirm"
- **THEN** all matches are saved, aliases are upserted, ItemFrequency is updated, and the modal closes

### Scenario: Undo last decision
- **WHEN** the manager clicks "Undo"
- **THEN** the most recent decision is reversed and the row returns to its previous state

### Scenario: Planned item deleted after being matched to a receipt item via link-alias
- **GIVEN** the manager links a receipt item to a planned item using "link alias"
- **THEN** the planned item is deleted (to avoid duplicate entries)

---

## Requirement: Alias creation from confirmed matches
Confirming a match builds the alias index for future auto-matching.

### Scenario: New alias created on first confirmation
- **GIVEN** no alias exists for (planned="Milk", receipt="Parmalat UHT 1L")
- **WHEN** the manager confirms this match
- **THEN** a new alias is created with purchase_count=1

### Scenario: Existing alias purchase count incremented
- **GIVEN** alias (planned="Milk", receipt="Parmalat UHT 1L") exists with count=3
- **WHEN** the same match is confirmed again
- **THEN** purchase_count becomes 4

---

## Requirement: Background receipt processing
Queued receipts are retried automatically when Gemini becomes available.

### Scenario: Queued receipt processed on next scheduler tick
- **GIVEN** a receipt has status `queued`
- **WHEN** the background job runs (every 10 minutes) and Gemini is available
- **THEN** the receipt is parsed and its status updated to `pending_review` or `parsed`
