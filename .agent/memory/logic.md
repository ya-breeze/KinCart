# Business Logic & Processing

## 1. Shopping Lists
- **Item Linking**: Items can be linked to flyer deals (`flyer_item_id`).
- **Protected Fields**: Linked items protect `name`, `price`, and `local_photo_path` to prevent deal data corruption. Users must "Unlink" to edit these manually.
- **Duplication Strategy**:
    - **Keep**: Flyer deals (`FlyerItemID`) relevance.
    - **Drop**: Past purchase history (`ReceiptItemID`) context.

## 2. Receipt Parsing
- **Gemini Integration**: Uses Gemini 1.5 Flash for multi-modal parsing of images and PDFs.
- **Offline Support**: If the API key is missing, receipts are queued (`Status="new"`) and immediate feedback is given.
- **Service Pattern**: Dependencies are injected via a `ReceiptParser` interface for testability.

## 3. Categories & Sorting
- **Optionality**: Categories are optional. Items without one are "Uncategorized".
- **Family Scoping**: Categories and shops must belong to the current family context.
- **Custom Ordering**: Categories are ordered per shop to optimize the shopping path.
