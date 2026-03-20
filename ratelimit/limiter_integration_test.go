package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(rdb.Context()).Err(); err != nil {
		t.Fatalf("redis not available: %v", err)
	}
	return rdb
}

func cleanupRateLimitKeys(t *testing.T, rdb *redis.Client, userID int64) {
	t.Helper()
	ctx := context.Background()
	prefix := fmt.Sprintf("user:%d", userID)
	minuteKey := time.Now().Unix() / 60
	rdb.Del(ctx,
		fmt.Sprintf("rl:rpm:%s:%d", prefix, minuteKey),
		fmt.Sprintf("rl:tpm:%s:%d", prefix, minuteKey),
		fmt.Sprintf("rl:conc:%s", prefix),
	)
}

// TC-HAPPY-LIMITER-INT-001: CheckPreRequest allows within limits
func TestCheckPreRequest_Redis_Allowed(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8181)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8181, AuthType: "user"}
	limits := RateLimits{RPM: 60, TPM: 100000, Concurrency: 10}

	result := limiter.CheckPreRequest(context.Background(), identity, limits, 500)

	if !result.Allowed {
		t.Fatalf("expected allowed, got denied: %s", result.DenyReason)
	}
	if result.RemainingRequests != 59 {
		t.Errorf("RemainingRequests = %d, want 59", result.RemainingRequests)
	}
	if result.RemainingTokens != 99500 {
		t.Errorf("RemainingTokens = %d, want 99500", result.RemainingTokens)
	}
	if !result.ConcurrencyAcquired {
		t.Error("expected ConcurrencyAcquired=true")
	}
	if result.MinuteKey == 0 {
		t.Error("expected non-zero MinuteKey")
	}
}

// TC-HAPPY-LIMITER-INT-002: RPM exceeded returns denied
func TestCheckPreRequest_Redis_RPMExceeded(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8182)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8182, AuthType: "user"}
	limits := RateLimits{RPM: 2, TPM: 100000, Concurrency: 10}

	// Use up the RPM limit
	for i := 0; i < 2; i++ {
		r := limiter.CheckPreRequest(context.Background(), identity, limits, 100)
		if !r.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		// Release concurrency so it doesn't block
		limiter.ReleaseConcurrency(context.Background(), identity)
	}

	// 3rd request should be denied
	result := limiter.CheckPreRequest(context.Background(), identity, limits, 100)
	if result.Allowed {
		t.Fatal("expected denied for RPM exceeded")
	}
	if result.DenyReason != "rpm" {
		t.Errorf("DenyReason = %q, want %q", result.DenyReason, "rpm")
	}
	if result.RetryAfterSeconds < 1 {
		t.Error("expected RetryAfterSeconds >= 1")
	}
}

// TC-HAPPY-LIMITER-INT-003: TPM exceeded returns denied
func TestCheckPreRequest_Redis_TPMExceeded(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8183)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8183, AuthType: "user"}
	limits := RateLimits{RPM: 100, TPM: 1000, Concurrency: 10}

	// Use up most of TPM
	r := limiter.CheckPreRequest(context.Background(), identity, limits, 900)
	if !r.Allowed {
		t.Fatal("first request should be allowed")
	}
	limiter.ReleaseConcurrency(context.Background(), identity)

	// This should exceed TPM (900 + 200 > 1000)
	result := limiter.CheckPreRequest(context.Background(), identity, limits, 200)
	if result.Allowed {
		t.Fatal("expected denied for TPM exceeded")
	}
	if result.DenyReason != "tpm" {
		t.Errorf("DenyReason = %q, want %q", result.DenyReason, "tpm")
	}
}

// TC-HAPPY-LIMITER-INT-004: concurrency limit enforced
func TestCheckPreRequest_Redis_ConcurrencyExceeded(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8184)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8184, AuthType: "user"}
	limits := RateLimits{RPM: 100, TPM: 100000, Concurrency: 2}

	// Acquire 2 concurrency slots (don't release)
	for i := 0; i < 2; i++ {
		r := limiter.CheckPreRequest(context.Background(), identity, limits, 10)
		if !r.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 3rd should be denied for concurrency
	result := limiter.CheckPreRequest(context.Background(), identity, limits, 10)
	if result.Allowed {
		t.Fatal("expected denied for concurrency exceeded")
	}
	if result.DenyReason != "concurrency" {
		t.Errorf("DenyReason = %q, want %q", result.DenyReason, "concurrency")
	}
}

// TC-HAPPY-LIMITER-INT-005: TrackPostResponse adjusts TPM counter
func TestTrackPostResponse_Redis_AdjustsTPM(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8185)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8185, AuthType: "user"}
	limits := RateLimits{RPM: 100, TPM: 100000, Concurrency: 10}

	// Pre-request with estimated 500 tokens
	result := limiter.CheckPreRequest(context.Background(), identity, limits, 500)
	if !result.Allowed {
		t.Fatal("expected allowed")
	}

	// Actual was only 200 tokens — should adjust
	limiter.TrackPostResponse(context.Background(), identity, 500, 200, result.MinuteKey, result.ConcurrencyAcquired)

	// After adjustment, remaining should be higher (corrected by +300)
	result2 := limiter.CheckPreRequest(context.Background(), identity, limits, 100)
	if !result2.Allowed {
		t.Fatal("expected allowed after TPM correction")
	}
	// Remaining should reflect the correction: 100000 - 200 - 100 = 99700
	if result2.RemainingTokens != 99700 {
		t.Errorf("RemainingTokens = %d, want 99700", result2.RemainingTokens)
	}
}

// TC-HAPPY-LIMITER-INT-006: ReleaseConcurrency frees a slot
func TestReleaseConcurrency_Redis(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()
	defer cleanupRateLimitKeys(t, rdb, 8186)

	limiter := NewLimiter(rdb)
	identity := RequestIdentity{UserID: 8186, AuthType: "user"}
	limits := RateLimits{RPM: 100, TPM: 100000, Concurrency: 1}

	// Acquire the only slot
	r := limiter.CheckPreRequest(context.Background(), identity, limits, 10)
	if !r.Allowed || !r.ConcurrencyAcquired {
		t.Fatal("first request should be allowed with concurrency")
	}

	// Release the slot
	limiter.ReleaseConcurrency(context.Background(), identity)

	// Should be able to acquire again
	r2 := limiter.CheckPreRequest(context.Background(), identity, limits, 10)
	if !r2.Allowed {
		t.Fatalf("expected allowed after release, got denied: %s", r2.DenyReason)
	}
}
