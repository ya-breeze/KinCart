## Why

All AI features (receipt parsing, paste-to-list) call a hardcoded Gemini model `gemini-2.0-flash` (`backend/internal/ai/gemini.go:140`). Google has retired that model — every call returns `404 … no longer available … NOT_FOUND`. Receipt scanning fails outright (no fallback) and paste-to-list silently degrades to the local regex parser. Pinning any specific version is fragile: verification against the live API shows even `gemini-2.5-flash` already returns `404 "no longer available to new users"`, while the rolling alias `gemini-flash-latest` returns `200`.

## What Changes

- Switch the default Gemini model from the pinned `gemini-2.0-flash` to the stable rolling alias `gemini-flash-latest`, which Google keeps pointed at a current model so it survives individual version retirements.
- Make the model name configurable via a `GEMINI_MODEL` environment variable (defaulting to `gemini-flash-latest`), so a future forced change is config, not a code redeploy.
- Log the selected model at client creation so the active model is visible in the logs.

## Capabilities

### New Capabilities
- `ai-model-selection`: How AI features choose the Gemini model — a stable alias by default, overridable by configuration.

### Modified Capabilities
<!-- none: receipts and paste-to-list behavior is unchanged; only the underlying model selection is fixed -->

## Impact

- **Backend:** `NewGeminiClient` reads `GEMINI_MODEL` (default `gemini-flash-latest`) instead of the hardcoded string; adds a startup log line. No change to prompts, schemas, or call sites (all use `c.model`).
- **Config:** New optional env var `GEMINI_MODEL`. Documented in `CLAUDE.md` env section.
- **Deployments:** Fixes both the prod `kincart` stack and `kincart-wip` once merged (prod auto-updates from `main`).
- **Docs:** Adds `docs/adr/ADR-001-stable-gemini-model-alias.md` recording the decision to prefer a rolling alias over a pinned version.
