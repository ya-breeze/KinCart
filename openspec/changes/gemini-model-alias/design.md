## Context

`GeminiClient` holds a `model string` used in every `client.Models.GenerateContent(ctx, c.model, …)` call (`gemini.go:98,231,276,351`). `NewGeminiClient` hardcodes `model: "gemini-2.0-flash"`. Google retired that model; live-API verification with the deployed key shows:

- `gemini-2.0-flash` → 404 (retired)
- `gemini-2.5-flash` → 404 ("no longer available to new users")
- `gemini-flash-latest` → 200 (works)

`gemini-flash-latest` is a provider-maintained rolling alias that tracks a current Flash model, so it does not require code changes when a specific version is retired.

## Goals / Non-Goals

**Goals:**
- Restore AI features by targeting a currently-served model.
- Prefer a stable alias so this class of breakage does not recur silently.
- Allow an operator to override the model without a redeploy.

**Non-Goals:**
- No change to prompts, schemas, or parsing logic.
- No change to the existing paste-to-list fallback behavior.
- Not adding a receipt-parsing fallback here (tracked separately if desired).

## Decisions

- **Default to `gemini-flash-latest`, overridable by `GEMINI_MODEL`.** A single read in `NewGeminiClient`: `model := os.Getenv("GEMINI_MODEL"); if model == "" { model = defaultGeminiModel }` with `const defaultGeminiModel = "gemini-flash-latest"`. Rationale: the alias is the durable target; the env var is an operational escape hatch. Alternative considered: pin to `gemini-2.5-flash` — rejected, it already 404s for this key and would recur.
- **Log the selected model.** One `slog.Info("Gemini client initialized", "model", model)` so the active model is observable, since an alias hides the concrete version.
- **ADR to capture the "alias over pin" principle.** New `docs/adr/ADR-001-stable-gemini-model-alias.md` (Status: Proposed → Accepted on merge), the first ADR in this repo.

## Risks / Trade-offs

- [A rolling alias can change the underlying model unexpectedly, shifting output/latency] → Acceptable for this app's parsing use; the env override lets an operator pin a specific version if a regression appears.
- [The alias could itself change name in future] → Mitigated by the env override; no code change needed to switch.

## Migration Plan

- No data migration. Deploy sets/inherits `GEMINI_MODEL` (optional). WIP stack env can add `GEMINI_MODEL=gemini-flash-latest` explicitly for clarity; prod inherits the default on next auto-update from `main`.
