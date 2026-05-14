package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testSecret is a deliberately-shaped placeholder that is ≥32 chars and does
// NOT appear in knownBadSecrets, so it passes ValidateSecretStrength without
// triggering gitleaks JWT/HMAC entropy heuristics.
const testSecret = "test_jwt_secret_key_0000000000000000"

func newTestSvc() *TokenService {
	svc, err := NewTokenService(testSecret, "synapse-test", 3600, 86400)
	if err != nil {
		panic("newTestSvc: " + err.Error())
	}
	return svc
}

func TestIssueParsePlatformToken(t *testing.T) {
	svc := newTestSvc()
	tok, err := svc.IssuePlatformToken(1, []string{"platform.admin"}, 1)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := svc.ParsePlatformToken(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.PrincipalID != 1 {
		t.Errorf("principal_id: want 1 got %d", claims.PrincipalID)
	}
	if claims.TokenType != "platform" {
		t.Errorf("token_type: want platform got %s", claims.TokenType)
	}
}

func TestIssueParseTenantToken(t *testing.T) {
	svc := newTestSvc()
	actorID := uint64(99)
	tok, err := svc.IssueTenantToken(10, "demo", 5, "tenant_admin", []string{"tenant.admin"}, 1, &actorID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := svc.ParseTenantToken(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.TenantCode != "demo" {
		t.Errorf("tenant_code: want demo got %s", claims.TenantCode)
	}
	if claims.ActorPrincipalID == nil || *claims.ActorPrincipalID != 99 {
		t.Error("actor_principal_id not preserved")
	}
}

func TestWrongTypeParseFails(t *testing.T) {
	svc := newTestSvc()
	platTok, _ := svc.IssuePlatformToken(1, nil, 1)
	if _, err := svc.ParseTenantToken(platTok); err != ErrWrongType {
		t.Errorf("want ErrWrongType, got %v", err)
	}
}

// ── ValidateSecretStrength / NewTokenService validation tests ──────────────

func TestNewTokenService_StrongSecretAccepted(t *testing.T) {
	svc, err := NewTokenService(testSecret, "issuer", 3600, 86400)
	if err != nil {
		t.Fatalf("expected strong secret to be accepted, got error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil TokenService")
	}
}

func TestNewTokenServiceWithAudience(t *testing.T) {
	svc, err := NewTokenServiceWithAudience(testSecret, "issuer", "service-a", 3600, 86400)
	if err != nil {
		t.Fatalf("expected explicit audience to be accepted, got error: %v", err)
	}
	if svc.Audience() != "service-a" {
		t.Fatalf("audience = %q, want service-a", svc.Audience())
	}
	tok, err := svc.IssuePlatformToken(1, []string{"platform.admin"}, 1)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := svc.ParsePlatformToken(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !containsAudience(claims.Audience, "service-a") {
		t.Fatalf("token missing explicit audience, got %v", claims.Audience)
	}
}

func TestNewTokenService_BlankAudienceRejected(t *testing.T) {
	_, err := NewTokenServiceWithAudience(testSecret, "issuer", " ", 3600, 86400)
	if err == nil {
		t.Fatal("expected error for blank audience")
	}
}

func TestNewTokenService_ShortSecretRejected(t *testing.T) {
	_, err := NewTokenService("tooshort", "issuer", 3600, 86400)
	if err == nil {
		t.Fatal("expected error for short secret, got nil")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("error should mention minimum length, got: %v", err)
	}
}

func TestNewTokenService_ExactlyMinLengthAccepted(t *testing.T) {
	// 32 chars exactly — must be accepted (and not in blacklist)
	secret32 := "aaaaaaaaaabbbbbbbbbbccccccccccdd" // 32 chars, not in blacklist
	if len(secret32) != 32 {
		t.Fatalf("test setup: secret32 length = %d, want 32", len(secret32))
	}
	_, err := NewTokenService(secret32, "issuer", 3600, 86400)
	if err != nil {
		t.Fatalf("32-char non-placeholder secret should be accepted, got: %v", err)
	}
}

func TestNewTokenService_BlacklistRejected(t *testing.T) {
	cases := []struct {
		name   string
		secret string
	}{
		{"your_production_jwt_secret_key_here", "your_production_jwt_secret_key_here"},
		{"dev_jwt_secret_key_please_change_in_production", "dev_jwt_secret_key_please_change_in_production"},
		{"dev_jwt_secret_key_please_change_in_production_32chars", "dev_jwt_secret_key_please_change_in_production_32chars"},
		{"test_jwt_secret_key_32chars_for_testing", "test_jwt_secret_key_32chars_for_testing"},
		{"change_me", "change_me"},
		{"secret", "secret"},
		{"jwt_secret", "jwt_secret"},
		{"default_secret", "default_secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewTokenService(tc.secret, "issuer", 3600, 86400)
			if err == nil {
				t.Fatalf("placeholder secret %q should be rejected, got nil error", tc.secret)
			}
		})
	}
}

func TestNewTokenService_BlacklistCaseInsensitive(t *testing.T) {
	// Java mirrors exact case-insensitive match — "SECRET" should also be rejected.
	_, err := NewTokenService("SECRET", "issuer", 3600, 86400)
	if err == nil {
		t.Fatal("upper-case placeholder SECRET should be rejected")
	}
}

func TestNewTokenService_LongStringContainingPlaceholderAccepted(t *testing.T) {
	// Blacklist uses exact match, NOT substring. A long secret that happens to
	// contain a blacklisted word should NOT be rejected.
	// This mirrors Java isDefaultPlaceholder() which also uses exact equality.
	longSecret := "PROD_jwt_secret_0000000000000000000000" // 38 chars, not in blacklist
	_, err := NewTokenService(longSecret, "issuer", 3600, 86400)
	if err != nil {
		t.Fatalf("long secret containing word 'secret' should be accepted (exact match only), got: %v", err)
	}
}

// ── audience claim (Audit MEDIUM #7) ───────────────────────────────────────

// TestIssuedTokensCarryExpectedAudience guards against accidentally dropping
// the `aud` claim on the issuer side. All three token types (platform,
// tenant, step-up) must include ExpectedAudience so future cross-service
// deployments can validate it.
func TestIssuedTokensCarryExpectedAudience(t *testing.T) {
	svc := newTestSvc()
	platTok, _ := svc.IssuePlatformToken(1, []string{"platform.admin"}, 1)
	tenantTok, _ := svc.IssueTenantToken(10, "demo", 5, "tenant_admin", []string{"tenant.admin"}, 1, nil)
	stepUpTok, _ := svc.IssueStepUpToken(7, "test.scope", 60*time.Second)

	platClaims, err := svc.ParsePlatformToken(platTok)
	if err != nil {
		t.Fatalf("platform parse: %v", err)
	}
	if !containsAudience(platClaims.Audience, ExpectedAudience) {
		t.Errorf("platform token missing audience %q (got %v)", ExpectedAudience, platClaims.Audience)
	}
	tenantClaims, err := svc.ParseTenantToken(tenantTok)
	if err != nil {
		t.Fatalf("tenant parse: %v", err)
	}
	if !containsAudience(tenantClaims.Audience, ExpectedAudience) {
		t.Errorf("tenant token missing audience %q (got %v)", ExpectedAudience, tenantClaims.Audience)
	}
	stepUpClaims, err := svc.ParseStepUpToken(stepUpTok)
	if err != nil {
		t.Fatalf("step-up parse: %v", err)
	}
	if !containsAudience(stepUpClaims.Audience, ExpectedAudience) {
		t.Errorf("step-up token missing audience %q (got %v)", ExpectedAudience, stepUpClaims.Audience)
	}
}

// TestParseRejectsWrongAudience: a token forged with a different `aud`
// (think: token from a sibling service that shares the secret) must be
// rejected.
func TestParseRejectsWrongAudience(t *testing.T) {
	svc := newTestSvc()
	now := time.Now()
	bad := PlatformClaims{
		PrincipalID:   1,
		PrincipalType: "platform_admin",
		TokenType:     "platform",
		TokenVersion:  1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "synapse-test",
			Audience:  jwt.ClaimStrings{"some-other-service"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, bad).SignedString(svc.secret)
	if err != nil {
		t.Fatalf("forge token: %v", err)
	}
	if _, err := svc.ParsePlatformToken(tok); err != ErrInvalidToken {
		t.Errorf("wrong audience must yield ErrInvalidToken, got %v", err)
	}
}

// TestParseRejectsTokenWithoutAudience covers the strict audience policy:
// tokens that carry no `aud` claim must not be accepted by control-plane APIs.
func TestParseRejectsTokenWithoutAudience(t *testing.T) {
	svc := newTestSvc()
	now := time.Now()
	legacy := PlatformClaims{
		PrincipalID:   1,
		PrincipalType: "platform_admin",
		TokenType:     "platform",
		TokenVersion:  1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "synapse-test",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, legacy).SignedString(svc.secret)
	if err != nil {
		t.Fatalf("forge legacy token: %v", err)
	}
	if _, err := svc.ParsePlatformToken(tok); err != ErrInvalidToken {
		t.Errorf("no-aud token must yield ErrInvalidToken, got %v", err)
	}
}

func containsAudience(aud jwt.ClaimStrings, want string) bool {
	for _, a := range aud {
		if a == want {
			return true
		}
	}
	return false
}
