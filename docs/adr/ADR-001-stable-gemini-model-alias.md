# ADR-001: Use a stable Gemini model alias instead of a pinned version

- **Status:** Proposed
- **Date:** 2026-07-19

## Context

All AI features (receipt parsing, paste-to-list) call Google's Gemini API with a
model name hardcoded in `backend/internal/ai/gemini.go`. The name was pinned to a
specific version, `gemini-2.0-flash`.

Google retired that version. Every AI call began returning
`404 … "This model models/gemini-2.0-flash is no longer available" … NOT_FOUND`.
Impact:

- **Receipt scanning** has no fallback, so it failed outright.
- **Paste-to-list** silently degraded to the local regex parser, losing AI quality,
  multilingual handling, promo-notation parsing, and price hints.

Verification against the live API with the deployed key showed that pinning a newer
version is not a reliable fix either:

| Model | `generateContent` |
|-------|-------------------|
| `gemini-2.0-flash` | 404 — retired |
| `gemini-2.5-flash` | 404 — "no longer available to new users" |
| `gemini-flash-latest` | 200 — works |

`gemini-flash-latest` is a provider-maintained rolling alias that tracks a current
Flash model.

## Decision

Default the Gemini model to the stable rolling alias **`gemini-flash-latest`** rather
than a pinned version, and make the name configurable via the **`GEMINI_MODEL`**
environment variable (used as the default when unset). The selected model is logged
at client startup.

## Consequences

- AI features survive the retirement of any single model version without a code change.
- If the alias ever needs to be overridden (regression, or the alias itself changes),
  an operator sets `GEMINI_MODEL` and restarts — no redeploy of code.
- Trade-off: a rolling alias can change the underlying model over time, which may shift
  output or latency. This is acceptable for parsing; the env override allows pinning a
  specific version if a regression appears.
