package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"
)

const (
	MaxKeyLength = 255
	hashPrefix   = "sha256:"
)

var (
	ErrInvalidKey   = errors.New("idempotency: invalid key")
	ErrInvalidScope = errors.New("idempotency: invalid scope")
)

// RequestScope is the stable tenant/principal/request boundary for a write.
// BodyHash is intentionally excluded from StorageKey and used by callers to
// detect a replay with the same key but a different request body.
type RequestScope struct {
	TenantID    string
	PrincipalID string
	Method      string
	Path        string
	Key         string
}

// NormalizeKey trims and validates an Idempotency-Key header value.
func NormalizeKey(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if key == "" || len(key) > MaxKeyLength {
		return "", ErrInvalidKey
	}
	for _, r := range key {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", ErrInvalidKey
		}
	}
	return key, nil
}

// BodyHash returns a stable SHA-256 digest for a request body.
func BodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hashPrefix + hex.EncodeToString(sum[:])
}

// ReaderHash consumes r and returns its SHA-256 digest.
func ReaderHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hashPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

// RequestHash hashes the normalized method, path and body hash. It is useful
// for detecting key reuse with a different payload.
func RequestHash(method, path string, body []byte) string {
	return HashParts(normalizeMethod(method), normalizePath(path), BodyHash(body))
}

// StorageKey returns a compact deterministic key for idempotency stores.
func StorageKey(scope RequestScope) (string, error) {
	normalized, err := NormalizeScope(scope)
	if err != nil {
		return "", err
	}
	return "idem:v1:" + trimHashPrefix(HashParts(
		normalized.TenantID,
		normalized.PrincipalID,
		normalized.Method,
		normalized.Path,
		normalized.Key,
	)), nil
}

// NormalizeScope validates and canonicalizes a request scope.
func NormalizeScope(scope RequestScope) (RequestScope, error) {
	key, err := NormalizeKey(scope.Key)
	if err != nil {
		return RequestScope{}, err
	}
	tenantID := strings.TrimSpace(scope.TenantID)
	principalID := strings.TrimSpace(scope.PrincipalID)
	method := normalizeMethod(scope.Method)
	path := normalizePath(scope.Path)
	if tenantID == "" || principalID == "" || method == "" || path == "" {
		return RequestScope{}, ErrInvalidScope
	}
	return RequestScope{
		TenantID:    tenantID,
		PrincipalID: principalID,
		Method:      method,
		Path:        path,
		Key:         key,
	}, nil
}

// HashParts hashes string parts with length prefixes to avoid delimiter
// ambiguity between adjacent fields.
func HashParts(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(h, "%d:", len(part))
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hashPrefix + hex.EncodeToString(h.Sum(nil))
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err == nil && req.URL != nil && req.URL.EscapedPath() != "" {
		path = req.URL.EscapedPath()
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func trimHashPrefix(hash string) string {
	return strings.TrimPrefix(hash, hashPrefix)
}
