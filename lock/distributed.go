package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// ErrLocked is returned when a lock key is already held by another owner.
	ErrLocked = errors.New("lock: already held by another owner")
	// ErrNotOwner is returned when release or renew sees a different owner token.
	ErrNotOwner = errors.New("lock: not owner")
	// ErrAlreadyReleased is returned when Release or Renew is called after release.
	ErrAlreadyReleased = errors.New("lock: handle already released")
	// ErrInvalidTTL is returned when ttl <= 0.
	ErrInvalidTTL = errors.New("lock: ttl must be positive")
	// ErrEmptyKey is returned when key is empty.
	ErrEmptyKey = errors.New("lock: key must not be empty")
	// ErrNilClient is returned when a Redis-backed locker has no client.
	ErrNilClient = errors.New("lock: redis client is nil")
)

// DistributedLocker acquires Redis locks with explicit owner handles.
type DistributedLocker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (LockHandle, error)
}

// LockHandle represents a held lock and supports owner-safe release and renew.
type LockHandle interface {
	Release(ctx context.Context) error
	Renew(ctx context.Context, ttl time.Duration) error
	Key() string
	Token() string
}

// DistributedOption configures a Redis distributed locker.
type DistributedOption func(*distributedConfig)

type distributedConfig struct {
	keyPrefix  string
	tokenBytes int
}

func defaultDistributedConfig() distributedConfig {
	return distributedConfig{
		keyPrefix:  "synapse:lock:",
		tokenBytes: 16,
	}
}

// WithKeyPrefix overrides the default "synapse:lock:" prefix.
func WithKeyPrefix(prefix string) DistributedOption {
	return func(c *distributedConfig) { c.keyPrefix = prefix }
}

// WithTokenBytes overrides the generated owner-token byte length. Values below
// 8 are clamped to 8.
func WithTokenBytes(n int) DistributedOption {
	return func(c *distributedConfig) {
		if n < 8 {
			n = 8
		}
		c.tokenBytes = n
	}
}

type redisDistributedLocker struct {
	client *redis.Client
	cfg    distributedConfig
}

// NewDistributed creates a Redis distributed locker. It does not replace the
// legacy New/TryLock API; use this API when callers need renew and owner-token
// diagnostics.
func NewDistributed(client *redis.Client, opts ...DistributedOption) DistributedLocker {
	cfg := defaultDistributedConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &redisDistributedLocker{client: client, cfg: cfg}
}

var (
	unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

	renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`)
)

func (l *redisDistributedLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (LockHandle, error) {
	if l == nil || l.client == nil {
		return nil, ErrNilClient
	}
	if key == "" {
		return nil, ErrEmptyKey
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	token, err := newOwnerToken(l.cfg.tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("lock: generate token: %w", err)
	}
	fullKey := l.cfg.keyPrefix + key
	ok, err := l.client.SetNX(ctx, fullKey, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("lock: setnx %q: %w", fullKey, err)
	}
	if !ok {
		return nil, ErrLocked
	}
	return &redisDistributedHandle{
		client:  l.client,
		fullKey: fullKey,
		key:     key,
		token:   token,
	}, nil
}

type redisDistributedHandle struct {
	client   *redis.Client
	fullKey  string
	key      string
	token    string
	released atomic.Bool
}

func (h *redisDistributedHandle) Key() string   { return h.key }
func (h *redisDistributedHandle) Token() string { return h.token }

func (h *redisDistributedHandle) Release(ctx context.Context) error {
	if h == nil || h.client == nil {
		return ErrNilClient
	}
	if !h.released.CompareAndSwap(false, true) {
		return ErrAlreadyReleased
	}
	res, err := unlockScript.Run(ctx, h.client, []string{h.fullKey}, h.token).Int64()
	if err != nil {
		h.released.Store(false)
		return fmt.Errorf("lock: unlock script %q: %w", h.fullKey, err)
	}
	if res != 1 {
		return ErrNotOwner
	}
	return nil
}

func (h *redisDistributedHandle) Renew(ctx context.Context, ttl time.Duration) error {
	if h == nil || h.client == nil {
		return ErrNilClient
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if h.released.Load() {
		return ErrAlreadyReleased
	}
	ms := ttl.Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	res, err := renewScript.Run(ctx, h.client, []string{h.fullKey}, h.token, ms).Int64()
	if err != nil {
		return fmt.Errorf("lock: renew script %q: %w", h.fullKey, err)
	}
	if res != 1 {
		return ErrNotOwner
	}
	return nil
}

func newOwnerToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
