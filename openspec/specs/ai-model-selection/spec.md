# AI Model Selection

## Purpose
Choose the Gemini model for AI features from configuration, defaulting to a stable rolling alias so a provider retiring a specific model version does not break AI features.

## Requirements

### Requirement: Configurable Gemini model with a stable default

Every Gemini call site SHALL select its model from an environment variable, defaulting to the rolling alias `gemini-flash-latest` when unset. Defaults SHALL be stable aliases rather than pinned versions, so that retirement of a specific model version does not break AI features. No call site SHALL hardcode a specific model version.

#### Scenario: Default model when unset
- **GIVEN** `GEMINI_MODEL` is not set
- **WHEN** the receipt/paste Gemini client is created
- **THEN** it uses `gemini-flash-latest`

#### Scenario: Override via environment
- **GIVEN** `GEMINI_MODEL` is set to a specific model name
- **WHEN** the receipt/paste Gemini client is created
- **THEN** it uses that configured model for its AI calls

#### Scenario: Flyer parsing has its own configurable model
- **GIVEN** the flyer parser is created
- **WHEN** `GEMINI_FLYER_MODEL` is unset
- **THEN** the flyer parser uses `gemini-flash-latest`
- **AND** setting `GEMINI_FLYER_MODEL` overrides it independently of `GEMINI_MODEL`

#### Scenario: Selected model is logged
- **WHEN** a Gemini client or the flyer parser is created
- **THEN** the active model name is written to the logs

#### Scenario: A retired pinned model can be replaced without a code change
- **GIVEN** the configured/default model is later retired by the provider
- **WHEN** an operator sets the relevant env var to a supported model and restarts
- **THEN** AI features use the new model without any code change
