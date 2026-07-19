## ADDED Requirements

### Requirement: Configurable Gemini model with a stable default

AI features SHALL select the Gemini model from the `GEMINI_MODEL` environment variable, defaulting to the rolling alias `gemini-flash-latest` when the variable is unset. The default SHALL be a stable alias rather than a pinned version, so that retirement of a specific model version does not break AI features.

#### Scenario: Default model when unset
- **GIVEN** `GEMINI_MODEL` is not set
- **WHEN** the Gemini client is created
- **THEN** it uses `gemini-flash-latest`

#### Scenario: Override via environment
- **GIVEN** `GEMINI_MODEL` is set to a specific model name
- **WHEN** the Gemini client is created
- **THEN** it uses that configured model for all AI calls

#### Scenario: Selected model is logged
- **WHEN** the Gemini client is created
- **THEN** the active model name is written to the logs

#### Scenario: A retired pinned model can be replaced without a code change
- **GIVEN** the configured/default model is later retired by the provider
- **WHEN** an operator sets `GEMINI_MODEL` to a supported model and restarts
- **THEN** AI features use the new model without any code change
