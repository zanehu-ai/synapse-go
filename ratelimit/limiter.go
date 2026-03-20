package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// Limiter performs multi-dimensional rate limiting using Redis.
type Limiter struct {
	rdb *redis.Client
}

func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

const keyTTL = 120 // seconds, covers the 60s window + buffer

// CheckPreRequest validates RPM and TPM limits and acquires a concurrency slot.
// Returns a CheckResult. If not allowed, the caller should return 429.
func (l *Limiter) CheckPreRequest(ctx context.Context, identity RequestIdentity, limits RateLimits, estimatedTokens int) CheckResult {
	if l.rdb == nil {
		// Redis unavailable → degrade to allow
		return CheckResult{
			Allowed:           true,
			RemainingRequests: limits.RPM,
			RemainingTokens:   limits.TPM,
			LimitRequests:     limits.RPM,
			LimitTokens:       limits.TPM,
		}
	}

	minuteKey := time.Now().Unix() / 60
	prefix := l.keyPrefix(identity)

	rpmKey := fmt.Sprintf("rl:rpm:%s:%d", prefix, minuteKey)
	tpmKey := fmt.Sprintf("rl:tpm:%s:%d", prefix, minuteKey)
	concKey := fmt.Sprintf("rl:conc:%s", prefix)

	// Check RPM + TPM atomically
	keys := []string{rpmKey, tpmKey}
	args := []interface{}{
		limits.RPM, 1, keyTTL, // RPM: limit, increment by 1, TTL
		limits.TPM, estimatedTokens, keyTTL, // TPM: limit, increment by estimated tokens, TTL
	}

	result, err := l.rdb.Eval(ctx, luaCheckAndIncrement, keys, args...).Int64Slice()
	if err != nil {
		// Redis error → degrade to allow
		return CheckResult{
			Allowed:           true,
			RemainingRequests: limits.RPM,
			RemainingTokens:   limits.TPM,
			LimitRequests:     limits.RPM,
			LimitTokens:       limits.TPM,
			MinuteKey:         minuteKey,
		}
	}

	allowed := result[0] == 1
	if !allowed {
		denyIndex := int(result[1])
		reason := "rpm"
		if denyIndex == 1 {
			reason = "tpm"
		}

		resetAt := time.Unix((minuteKey+1)*60, 0)
		retryAfter := int(time.Until(resetAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		return CheckResult{
			Allowed:           false,
			DenyReason:        reason,
			LimitRequests:     limits.RPM,
			LimitTokens:       limits.TPM,
			RemainingRequests: 0,
			RemainingTokens:   0,
			ResetAt:           resetAt,
			RetryAfterSeconds: retryAfter,
			MinuteKey:         minuteKey,
		}
	}

	remainingRequests := int(result[2])
	remainingTokens := int(result[3])

	// Check concurrency
	concResult, err := l.rdb.Eval(ctx, luaConcurrencyAcquire, []string{concKey},
		limits.Concurrency, keyTTL).Int64()
	if err != nil || concResult == 0 {
		if err == nil {
			// Concurrency denied — roll back RPM/TPM increments
			pipe := l.rdb.Pipeline()
			pipe.DecrBy(ctx, rpmKey, 1)
			pipe.DecrBy(ctx, tpmKey, int64(estimatedTokens))
			_, _ = pipe.Exec(ctx)

			return CheckResult{
				Allowed:           false,
				DenyReason:        "concurrency",
				LimitRequests:     limits.RPM,
				LimitTokens:       limits.TPM,
				RemainingRequests: remainingRequests,
				RemainingTokens:   remainingTokens,
				RetryAfterSeconds: 1,
				MinuteKey:         minuteKey,
			}
		}
		// Redis error on concurrency check → allow (degraded), but mark as not acquired
	}

	concurrencyAcquired := err == nil && concResult == 1

	resetAt := time.Unix((minuteKey+1)*60, 0)

	return CheckResult{
		Allowed:             true,
		ConcurrencyAcquired: concurrencyAcquired,
		RemainingRequests:   remainingRequests,
		RemainingTokens:     remainingTokens,
		LimitRequests:       limits.RPM,
		LimitTokens:         limits.TPM,
		ResetAt:             resetAt,
		MinuteKey:           minuteKey,
	}
}

// TrackPostResponse adjusts TPM counter with actual token count and optionally releases concurrency slot.
// Uses the original minuteKey from the pre-request check to correct the right minute window.
func (l *Limiter) TrackPostResponse(ctx context.Context, identity RequestIdentity, estimatedTokens, actualTokens int, minuteKey int64, releaseConcurrency bool) {
	if l.rdb == nil {
		return
	}

	prefix := l.keyPrefix(identity)

	// Release concurrency slot only if it was actually acquired
	if releaseConcurrency {
		concKey := fmt.Sprintf("rl:conc:%s", prefix)
		l.rdb.Eval(ctx, luaConcurrencyRelease, []string{concKey})
	}

	// Correct TPM counter if actual differs from estimate
	delta := int64(actualTokens - estimatedTokens)
	if delta != 0 {
		tpmKey := fmt.Sprintf("rl:tpm:%s:%d", prefix, minuteKey)
		l.rdb.IncrBy(ctx, tpmKey, delta)
	}
}

// ReleaseConcurrency releases a concurrency slot without adjusting TPM.
// Used when a request fails before generating tokens.
func (l *Limiter) ReleaseConcurrency(ctx context.Context, identity RequestIdentity) {
	if l.rdb == nil {
		return
	}
	prefix := l.keyPrefix(identity)
	concKey := fmt.Sprintf("rl:conc:%s", prefix)
	l.rdb.Eval(ctx, luaConcurrencyRelease, []string{concKey})
}

func (l *Limiter) keyPrefix(identity RequestIdentity) string {
	if identity.APIKeyID > 0 {
		return "apikey:" + strconv.FormatInt(identity.APIKeyID, 10)
	}
	if identity.TenantID > 0 {
		return "tenant:" + strconv.FormatInt(identity.TenantID, 10)
	}
	return "user:" + strconv.FormatInt(identity.UserID, 10)
}
