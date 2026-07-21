## ADDED Requirements

### Requirement: Paste preview enriches unit and category

The paste-to-list parse preview SHALL enrich each parsed item with a remembered or inferred unit and category, in addition to the existing price hint.

#### Scenario: Preview shows remembered unit and category
- **GIVEN** a pasted item name that exists in purchase history
- **WHEN** the parse preview is generated
- **THEN** each parsed item includes the remembered unit and category (per the item-defaults resolution, using the target list's shop where available)

#### Scenario: Preview falls back to AI for unseen items
- **GIVEN** a pasted item name with no purchase history
- **WHEN** the parse preview is generated and the AI service is available
- **THEN** the parsed item includes an AI-suggested unit and category

#### Scenario: Confirmed items keep their previewed defaults
- **WHEN** the user confirms the parsed items into the list
- **THEN** the created items carry the previewed unit and category
