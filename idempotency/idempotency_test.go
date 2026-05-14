package idempotency

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeKey(t *testing.T) {
	got, err := NormalizeKey(" request-123 ")
	if err != nil {
		t.Fatalf("NormalizeKey returned error: %v", err)
	}
	if got != "request-123" {
		t.Fatalf("NormalizeKey = %q, want request-123", got)
	}

	for _, raw := range []string{"", "has space", "line\nbreak", strings.Repeat("x", MaxKeyLength+1)} {
		if _, err := NormalizeKey(raw); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("NormalizeKey(%q) error = %v, want ErrInvalidKey", raw, err)
		}
	}
}

func TestBodyAndRequestHashAreStable(t *testing.T) {
	firstBodyHash := BodyHash([]byte("abc"))
	secondBodyHash := BodyHash([]byte("abc"))
	if firstBodyHash != secondBodyHash {
		t.Fatal("BodyHash should be stable")
	}
	if RequestHash("post", "api/v1/items", []byte("abc")) != RequestHash("POST", "/api/v1/items", []byte("abc")) {
		t.Fatal("RequestHash should normalize method and path")
	}
	if RequestHash("POST", "/api/v1/items", []byte("abc")) == RequestHash("POST", "/api/v1/items", []byte("def")) {
		t.Fatal("RequestHash should include body hash")
	}
}

func TestStorageKeyExcludesBodyAndNormalizesScope(t *testing.T) {
	scope := RequestScope{
		TenantID:    " t-1 ",
		PrincipalID: " p-1 ",
		Method:      "post",
		Path:        "api/v1/items",
		Key:         " request-123 ",
	}
	got, err := StorageKey(scope)
	if err != nil {
		t.Fatalf("StorageKey returned error: %v", err)
	}
	want, err := StorageKey(RequestScope{
		TenantID:    "t-1",
		PrincipalID: "p-1",
		Method:      "POST",
		Path:        "/api/v1/items",
		Key:         "request-123",
	})
	if err != nil {
		t.Fatalf("StorageKey returned error: %v", err)
	}
	if got != want {
		t.Fatalf("StorageKey normalization mismatch: %q != %q", got, want)
	}
	if !strings.HasPrefix(got, "idem:v1:") {
		t.Fatalf("StorageKey = %q, want idem:v1 prefix", got)
	}
}

func TestStorageKeyRejectsInvalidScope(t *testing.T) {
	_, err := StorageKey(RequestScope{TenantID: "t-1", Method: "POST", Path: "/x", Key: "k"})
	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("StorageKey error = %v, want ErrInvalidScope", err)
	}
}
