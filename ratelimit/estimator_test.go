package ratelimit

import (
	"testing"
)

// TC-HAPPY-ESTIMATOR-001: basic chat message estimation
func TestEstimateTotalTokens_BasicMessage(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, how are you?"}]}`)

	tokens := EstimateTotalTokens(body)
	if tokens < 1 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
	// Input: "Hello, how are you?" = 19 chars / 4 ≈ 4-5 tokens; Output: 500 default
	if tokens < 500 || tokens > 600 {
		t.Fatalf("expected ~500 total tokens, got %d", tokens)
	}
}

// TC-HAPPY-ESTIMATOR-002: with explicit max_tokens
func TestEstimateTotalTokens_WithMaxTokens(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Hi"}],"max_tokens":50}`)
	total := EstimateTotalTokens(body)
	// Input: "Hi" = 1 token (min), Output: 50
	if total < 50 {
		t.Fatalf("expected at least 50, got %d", total)
	}
}

// TC-HAPPY-ESTIMATOR-003: large max_tokens capped at 500
func TestEstimateTotalTokens_LargeMaxTokensCapped(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Hi"}],"max_tokens":10000}`)
	total := EstimateTotalTokens(body)
	// Output should be capped at 500, input is ~1
	if total > 510 {
		t.Fatalf("expected output capped at 500, total=%d", total)
	}
}

// TC-BOUNDARY-ESTIMATOR-001: empty body
func TestEstimateTotalTokens_EmptyBody(t *testing.T) {
	tokens := EstimateTotalTokens([]byte{})
	// Should not panic, returns some default
	if tokens < 1 {
		t.Fatalf("expected at least 1 token for empty body, got %d", tokens)
	}
}

// TC-BOUNDARY-ESTIMATOR-002: invalid JSON
func TestEstimateTotalTokens_InvalidJSON(t *testing.T) {
	tokens := EstimateTotalTokens([]byte("not json at all"))
	// Should not panic, falls back to body length / 4
	if tokens < 1 {
		t.Fatalf("expected positive tokens for invalid JSON, got %d", tokens)
	}
}

// TC-BOUNDARY-ESTIMATOR-003: very large input capped at maxInputTokenEstimate
func TestEstimateTotalTokens_LargeInputCapped(t *testing.T) {
	// Just above the cap: 128001 * 4 = 512004 chars → would be 128001 tokens before cap
	largeContent := make([]byte, 512004)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	body := append([]byte(`{"messages":[{"role":"user","content":"`), largeContent...)
	body = append(body, []byte(`"}]}`)...)
	tokens := EstimateTotalTokens(body)
	// Input should be capped at 128000, output at 500
	if tokens > maxInputTokenEstimate+500+10 {
		t.Fatalf("expected input capped at %d, total=%d", maxInputTokenEstimate, tokens)
	}
}

// TC-EXCEPTION-ESTIMATOR-001: nil body
func TestEstimateTotalTokens_NilBody(t *testing.T) {
	tokens := EstimateTotalTokens(nil)
	if tokens < 1 {
		t.Fatalf("expected at least 1 token for nil body, got %d", tokens)
	}
}
