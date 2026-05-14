package idempotency

// frozen reference
// Algorithm  : SHA-256 (crypto/sha256)
// Input      : raw request body bytes — NO JSON canonicalization, NO key sorting
// Encoding   : lowercase hex, always prefixed with "sha256:"
// Mismatch   : callers compare stored hash to BodyHash(body); different bodies → different hash → 409
// Java parity: IdempotencyHashUtil.bodyHash(byte[]) MUST produce identical strings
// Fixture    : testdata/hash_vectors.json (≥9 cases; run go test ./idempotency/... to verify)

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type hashVector struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputUTF8   string `json:"input_utf8"`
	Hash        string `json:"hash"`
}

type hashVectorsFile struct {
	Version int          `json:"version"`
	Cases   []hashVector `json:"cases"`
}

// TestHashVectorsFixture loads testdata/hash_vectors.json and asserts that
// BodyHash produces byte-equivalent output for every fixture row. This test
// is the Go-side source-of-truth for the cross-language conformance suite;
// the Java side loads the same JSON from
// templates/game/backend/platform-core/platform-common/src/test/resources/
// conformance/idempotency_hash_vectors.json.
func TestHashVectorsFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/hash_vectors.json")
	if err != nil {
		t.Fatalf("cannot read testdata/hash_vectors.json: %v", err)
	}

	var f hashVectorsFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("cannot parse testdata/hash_vectors.json: %v", err)
	}

	if f.Version != 1 {
		t.Fatalf("fixture version = %d, want 1", f.Version)
	}
	if len(f.Cases) < 8 {
		t.Fatalf("fixture has %d cases, want ≥8 (regression guard)", len(f.Cases))
	}

	for _, tc := range f.Cases {
		tc := tc // capture
		t.Run(tc.Name, func(t *testing.T) {
			got := BodyHash([]byte(tc.InputUTF8))
			if got != tc.Hash {
				t.Errorf("BodyHash(%q) = %q, fixture wants %q", tc.InputUTF8, got, tc.Hash)
			}
			// encoding invariant: prefix + lowercase hex only
			if !strings.HasPrefix(got, "sha256:") {
				t.Errorf("BodyHash output missing sha256: prefix: %q", got)
			}
			hex := strings.TrimPrefix(got, "sha256:")
			for _, c := range hex {
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
					t.Errorf("BodyHash hex contains non-lowercase-hex char %q in %q", c, got)
					break
				}
			}
		})
	}
}

// TestBodyHashWhitespaceSignificance confirms the "no canonicalization" invariant:
// semantically identical JSON with different whitespace produces different hashes.
func TestBodyHashWhitespaceSignificance(t *testing.T) {
	compact := BodyHash([]byte(`{"amount":100,"currency":"USD"}`))
	spaced := BodyHash([]byte(`{"amount": 100, "currency": "USD"}`))
	if compact == spaced {
		t.Error("BodyHash must NOT canonicalize JSON: compact and spaced forms should differ")
	}
}

// TestBodyHashKeyOrderSignificance confirms different key order → different hash.
func TestBodyHashKeyOrderSignificance(t *testing.T) {
	h1 := BodyHash([]byte(`{"amount":100,"currency":"USD"}`))
	h2 := BodyHash([]byte(`{"currency":"USD","amount":100}`))
	if h1 == h2 {
		t.Error("BodyHash must NOT sort keys: different key-order JSON should differ")
	}
}

// TestBodyHashSingleByteTamper ensures a 1-byte body change produces a different hash.
func TestBodyHashSingleByteTamper(t *testing.T) {
	orig := BodyHash([]byte(`{"amount":100,"currency":"USD"}`))
	tampered := BodyHash([]byte(`{"amount":101,"currency":"USD"}`))
	if orig == tampered {
		t.Error("BodyHash should detect single-byte tamper")
	}
}
