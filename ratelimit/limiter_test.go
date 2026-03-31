package ratelimit

import (
	"context"
	"testing"
)

// TC-HAPPY-LIMITER-001: nil Redis degrades to allow
func TestNewLimiter_NilRedis(t *testing.T) {
	limiter := NewLimiter(nil)
	identity := RequestIdentity{UserID: 1, AuthType: "user"}
	limits := DefaultLimits()

	result := limiter.CheckPreRequest(context.TODO(), identity, limits, 100)
	if !result.Allowed {
		t.Fatal("expected request to be allowed when redis is nil")
	}
}

// TC-HAPPY-LIMITER-002: nil Redis returns correct remaining values
func TestNewLimiter_NilRedisRemainingValues(t *testing.T) {
	limiter := NewLimiter(nil)
	identity := RequestIdentity{UserID: 1, AuthType: "user"}
	limits := RateLimits{RPM: 60, TPM: 100000, Concurrency: 10}

	result := limiter.CheckPreRequest(context.TODO(), identity, limits, 100)
	if result.RemainingRequests != 60 {
		t.Errorf("RemainingRequests = %d, want 60", result.RemainingRequests)
	}
	if result.RemainingTokens != 100000 {
		t.Errorf("RemainingTokens = %d, want 100000", result.RemainingTokens)
	}
}

// TC-SAFETY-LIMITER-001: nil Redis → ConcurrencyAcquired must be false
func TestNilLimiter_ConcurrencyNotAcquired(t *testing.T) {
	limiter := NewLimiter(nil)
	identity := RequestIdentity{UserID: 1, AuthType: "user"}
	limits := DefaultLimits()

	result := limiter.CheckPreRequest(context.TODO(), identity, limits, 100)
	if result.ConcurrencyAcquired {
		t.Fatal("expected ConcurrencyAcquired=false when Redis is nil")
	}
}

// TC-SAFETY-LIMITER-002: nil Redis → TrackPostResponse doesn't panic
func TestNilLimiter_TrackPostResponseNoPanic(t *testing.T) {
	limiter := NewLimiter(nil)
	identity := RequestIdentity{UserID: 1, AuthType: "user"}
	limiter.TrackPostResponse(context.TODO(), identity, 100, 200, 12345, true)
	limiter.TrackPostResponse(context.TODO(), identity, 100, 200, 12345, false)
}

// TC-SAFETY-LIMITER-003: nil Redis → ReleaseConcurrency doesn't panic
func TestNilLimiter_ReleaseConcurrencyNoPanic(t *testing.T) {
	limiter := NewLimiter(nil)
	identity := RequestIdentity{UserID: 1, AuthType: "user"}
	limiter.ReleaseConcurrency(context.TODO(), identity)
}

// TC-BOUNDARY-LIMITER-001: key prefix selection
func TestKeyPrefix_Selection(t *testing.T) {
	limiter := NewLimiter(nil)

	tests := []struct {
		name     string
		identity RequestIdentity
		want     string
	}{
		{
			name:     "API key takes priority",
			identity: RequestIdentity{UserID: 1, TenantID: 2, APIKeyID: 3},
			want:     "apikey:3",
		},
		{
			name:     "tenant when no API key",
			identity: RequestIdentity{UserID: 1, TenantID: 2, APIKeyID: 0},
			want:     "tenant:2",
		},
		{
			name:     "user when no tenant or API key",
			identity: RequestIdentity{UserID: 1, TenantID: 0, APIKeyID: 0},
			want:     "user:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limiter.KeyPrefix(tt.identity)
			if got != tt.want {
				t.Errorf("KeyPrefix = %q, want %q", got, tt.want)
			}
		})
	}
}
