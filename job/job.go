package job

import (
	"errors"
	"math"
	"strings"
	"time"
)

var (
	ErrInvalidLease         = errors.New("job: invalid lease")
	ErrInvalidBackoffPolicy = errors.New("job: invalid backoff policy")
)

type Lease struct {
	JobName    string
	Owner      string
	AcquiredAt time.Time
	ExpiresAt  time.Time
}

func NewLease(jobName, owner string, now time.Time, ttl time.Duration) (Lease, error) {
	if strings.TrimSpace(jobName) == "" || strings.TrimSpace(owner) == "" || ttl <= 0 || now.IsZero() {
		return Lease{}, ErrInvalidLease
	}
	return Lease{
		JobName:    strings.TrimSpace(jobName),
		Owner:      strings.TrimSpace(owner),
		AcquiredAt: now.UTC(),
		ExpiresAt:  now.UTC().Add(ttl),
	}, nil
}

func (l Lease) Expired(now time.Time) bool {
	return !now.UTC().Before(l.ExpiresAt)
}

func (l Lease) HeldBy(owner string, now time.Time) bool {
	return strings.TrimSpace(owner) == l.Owner && !l.Expired(now)
}

func CanAcquire(existing *Lease, owner string, now time.Time) bool {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return false
	}
	if existing == nil {
		return true
	}
	return existing.HeldBy(owner, now) || existing.Expired(now)
}

type BackoffPolicy struct {
	Base       time.Duration
	Max        time.Duration
	Multiplier float64
}

func DefaultBackoffPolicy() BackoffPolicy {
	return BackoffPolicy{
		Base:       time.Second,
		Max:        5 * time.Minute,
		Multiplier: 2,
	}
}

func (p BackoffPolicy) WithDefaults() BackoffPolicy {
	defaults := DefaultBackoffPolicy()
	if p.Base == 0 {
		p.Base = defaults.Base
	}
	if p.Max == 0 {
		p.Max = defaults.Max
	}
	if p.Multiplier == 0 {
		p.Multiplier = defaults.Multiplier
	}
	return p
}

func (p BackoffPolicy) Validate() error {
	p = p.WithDefaults()
	if p.Base <= 0 || p.Max <= 0 || p.Multiplier < 1 || p.Base > p.Max {
		return ErrInvalidBackoffPolicy
	}
	return nil
}

func (p BackoffPolicy) Delay(attempt int) (time.Duration, error) {
	p = p.WithDefaults()
	if err := p.Validate(); err != nil {
		return 0, err
	}
	if attempt < 0 {
		attempt = 0
	}
	delay := float64(p.Base)
	if attempt > 0 {
		delay *= math.Pow(p.Multiplier, float64(attempt))
	}
	if delay >= float64(p.Max) {
		return p.Max, nil
	}
	return time.Duration(delay), nil
}

func (p BackoffPolicy) NextRunAfter(now time.Time, attempt int) (time.Time, error) {
	delay, err := p.Delay(attempt)
	if err != nil {
		return time.Time{}, err
	}
	return now.UTC().Add(delay), nil
}
