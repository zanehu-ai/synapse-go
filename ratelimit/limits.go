package ratelimit

import "time"

// RateLimits holds the resolved rate limits for a request.
type RateLimits struct {
	RPM         int // requests per minute
	TPM         int // tokens per minute
	Concurrency int // max concurrent requests
}

// CheckResult holds the result of a rate limit check.
type CheckResult struct {
	Allowed             bool
	DenyReason          string // "rpm", "tpm", "concurrency"
	ConcurrencyAcquired bool   // true if a concurrency slot was successfully acquired
	RemainingRequests   int
	RemainingTokens     int
	LimitRequests       int
	LimitTokens         int
	ResetAt             time.Time
	RetryAfterSeconds   int
	MinuteKey           int64 // the minute key used for RPM/TPM counters
}

// RequestIdentity identifies the entity making a request for rate limiting purposes.
type RequestIdentity struct {
	// One of these will be set
	UserID   int64
	TenantID int64
	APIKeyID int64

	// Context
	Model    string
	AuthType string // "user" or "tenant"
}

// DefaultLimits returns global defaults when no specific limits are configured.
func DefaultLimits() RateLimits {
	return RateLimits{
		RPM:         60,
		TPM:         100000,
		Concurrency: 10,
	}
}
