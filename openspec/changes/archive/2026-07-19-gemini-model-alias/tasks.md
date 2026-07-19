## 1. Backend

- [x] 1.1 Add `const defaultGeminiModel = "gemini-flash-latest"` in `internal/ai/gemini.go`
- [x] 1.2 In `NewGeminiClient`, read `GEMINI_MODEL` env (fallback to `defaultGeminiModel`) instead of the hardcoded `gemini-2.0-flash`
- [x] 1.3 Log the selected model at client creation (`slog.Debug` on the per-request client to avoid noise)
- [x] 1.4 Add shared `ai.ResolveModel(envVar, fallback)`; route the flyer parser (`internal/flyers/parser.go`) through it via `GEMINI_FLYER_MODEL` (default `gemini-flash-latest`) instead of the hardcoded `gemini-3-flash-preview`

## 2. Tests

- [x] 2.1 Test: model resolves to `gemini-flash-latest` when `GEMINI_MODEL` unset
- [x] 2.2 Test: model uses the env value when `GEMINI_MODEL` is set

## 3. Docs

- [x] 3.1 Add `docs/adr/ADR-001-stable-gemini-model-alias.md` (Status: Proposed)
- [x] 3.2 Document `GEMINI_MODEL` in `CLAUDE.md` env vars section

## 4. Verification

- [x] 4.1 `go build`, `go vet`, and `go test ./internal/ai/` pass (golangci-lint not installed in this env; gofmt clean)
- [x] 4.2 Deployed to `kincart-wip`; paste-to-list verified using Gemini (promo `4+2` split, `potatoes`→`potato`/kg; 3.5s round-trip; no NOT_FOUND/fallback in logs). Receipt path uses the same now-resolving client/model (`gemini-flash-latest`, verified serving); not exercised with a real receipt image.
- [x] 4.3 Flip ADR status to Accepted on merge to main
