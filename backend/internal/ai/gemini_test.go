package ai

import "testing"

func TestResolveGeminiModel(t *testing.T) {
	t.Run("defaults to stable alias when unset", func(t *testing.T) {
		t.Setenv("GEMINI_MODEL", "")
		if got := resolveGeminiModel(); got != defaultGeminiModel {
			t.Fatalf("expected %q, got %q", defaultGeminiModel, got)
		}
	})

	t.Run("uses env override when set", func(t *testing.T) {
		t.Setenv("GEMINI_MODEL", "gemini-custom-test")
		if got := resolveGeminiModel(); got != "gemini-custom-test" {
			t.Fatalf("expected env override, got %q", got)
		}
	})

	t.Run("default is a rolling alias, not a pinned version", func(t *testing.T) {
		// A pinned version (e.g. gemini-2.0-flash) can be retired by the
		// provider; the default must be the "-latest" alias to avoid that.
		if defaultGeminiModel != "gemini-flash-latest" {
			t.Fatalf("default model should be the stable alias, got %q", defaultGeminiModel)
		}
	})
}
