package runtimeguard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const redisLimiterScript = `
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local ttl = redis.call("PTTL", KEYS[1])
if current >= limit then
	if ttl < 0 then
		redis.call("PEXPIRE", KEYS[1], window)
		ttl = window
	end
	return {0, current, ttl}
end
current = redis.call("INCR", KEYS[1])
if current == 1 or ttl < 0 then
	redis.call("PEXPIRE", KEYS[1], window)
	ttl = window
end
return {1, current, ttl}
`

const redisLockReleaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// RedisRuntimeStore is the minimal Redis command surface used by runtimeguard.
type RedisRuntimeStore interface {
	EvalIntSlice(ctx context.Context, script string, keys []string, args ...string) ([]int64, error)
	EvalInt(ctx context.Context, script string, keys []string, args ...string) (int64, error)
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
}

// GoRedisStore adapts github.com/go-redis/redis/v8 to RedisRuntimeStore.
type GoRedisStore struct {
	Client *redis.Client
}

// EvalIntSlice runs EVAL and expects an array of integers.
func (s GoRedisStore) EvalIntSlice(ctx context.Context, script string, keys []string, args ...string) ([]int64, error) {
	if s.Client == nil {
		return nil, errors.New("runtimeguard: redis client is nil")
	}
	rawArgs := make([]any, len(args))
	for i, arg := range args {
		rawArgs[i] = arg
	}
	value, err := s.Client.Eval(ctx, script, keys, rawArgs...).Result()
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok {
		return nil, errors.New("runtimeguard: redis response is not an array")
	}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		n, err := redisValueInt(item)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// EvalInt runs EVAL and expects a single integer.
func (s GoRedisStore) EvalInt(ctx context.Context, script string, keys []string, args ...string) (int64, error) {
	if s.Client == nil {
		return 0, errors.New("runtimeguard: redis client is nil")
	}
	rawArgs := make([]any, len(args))
	for i, arg := range args {
		rawArgs[i] = arg
	}
	value, err := s.Client.Eval(ctx, script, keys, rawArgs...).Result()
	if err != nil {
		return 0, err
	}
	return redisValueInt(value)
}

// SetNX sets key to value with ttl only when key does not already exist.
func (s GoRedisStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if s.Client == nil {
		return false, errors.New("runtimeguard: redis client is nil")
	}
	return s.Client.SetNX(ctx, key, value, ttl).Result()
}

// RedisLimiterOptions configures RedisLimiter.
type RedisLimiterOptions struct {
	KeyPrefix string
}

// RedisLimiter is a fixed-window limiter backed by Redis.
type RedisLimiter struct {
	store     RedisRuntimeStore
	keyPrefix string
}

// NewRedisLimiter creates a Redis-backed fixed-window limiter.
func NewRedisLimiter(store RedisRuntimeStore, opts RedisLimiterOptions) *RedisLimiter {
	return &RedisLimiter{store: store, keyPrefix: redisKeyPrefix(opts.KeyPrefix)}
}

// Allow consumes one event for key if the Redis fixed window has capacity.
func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	key = strings.TrimSpace(key)
	if key == "" || limit <= 0 || window <= 0 {
		return Decision{Allowed: false, RetryAfter: window}, nil
	}
	if l == nil || l.store == nil {
		return Decision{Allowed: false, RetryAfter: window}, errors.New("runtimeguard: redis limiter store is nil")
	}

	result, err := l.store.EvalIntSlice(ctx, redisLimiterScript, []string{l.limiterKey(key)},
		strconv.FormatInt(durationMillis(window), 10),
		strconv.Itoa(limit),
	)
	if err != nil {
		return Decision{Allowed: false, RetryAfter: window}, err
	}
	if len(result) != 3 {
		return Decision{Allowed: false, RetryAfter: window}, errors.New("runtimeguard: invalid redis limiter response")
	}
	return redisLimiterDecision(result, limit, window), nil
}

func (l *RedisLimiter) limiterKey(key string) string {
	return l.keyPrefix + "limiter:" + key
}

// RedisLockOptions configures RedisLockManager.
type RedisLockOptions struct {
	KeyPrefix string
}

// RedisLockManager coordinates short-lived Redis locks across processes.
type RedisLockManager struct {
	store     RedisRuntimeStore
	keyPrefix string
}

// RedisLock is an acquired Redis lock. Token is required to release it.
type RedisLock struct {
	Key   string
	Owner string
	Token string

	manager *RedisLockManager
}

// NewRedisLockManager creates a Redis-backed lock manager.
func NewRedisLockManager(store RedisRuntimeStore, opts RedisLockOptions) *RedisLockManager {
	return &RedisLockManager{store: store, keyPrefix: redisKeyPrefix(opts.KeyPrefix)}
}

// Acquire attempts to hold key for ttl. Redis errors are returned so callers
// can fail closed on sensitive paths.
func (m *RedisLockManager) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (RedisLock, bool, error) {
	key = strings.TrimSpace(key)
	owner = strings.TrimSpace(owner)
	if key == "" || owner == "" || ttl <= 0 {
		return RedisLock{}, false, nil
	}
	if m == nil || m.store == nil {
		return RedisLock{}, false, errors.New("runtimeguard: redis lock store is nil")
	}

	token, err := newLockToken()
	if err != nil {
		return RedisLock{}, false, err
	}
	acquired, err := m.store.SetNX(ctx, m.lockKey(key), token, ttl)
	if err != nil || !acquired {
		return RedisLock{}, false, err
	}
	return RedisLock{Key: key, Owner: owner, Token: token, manager: m}, true, nil
}

// Release deletes key only when token matches the lock value set by Acquire.
func (m *RedisLockManager) Release(ctx context.Context, key, token string) (bool, error) {
	key = strings.TrimSpace(key)
	token = strings.TrimSpace(token)
	if key == "" || token == "" {
		return false, nil
	}
	if m == nil || m.store == nil {
		return false, errors.New("runtimeguard: redis lock store is nil")
	}
	deleted, err := m.store.EvalInt(ctx, redisLockReleaseScript, []string{m.lockKey(key)}, token)
	if err != nil {
		return false, err
	}
	return deleted == 1, nil
}

// Release deletes this lock if its token still owns the Redis key.
func (l RedisLock) Release(ctx context.Context) (bool, error) {
	if l.manager == nil {
		return false, errors.New("runtimeguard: redis lock manager is nil")
	}
	return l.manager.Release(ctx, l.Key, l.Token)
}

func (m *RedisLockManager) lockKey(key string) string {
	return m.keyPrefix + "lock:" + key
}

func redisLimiterDecision(result []int64, limit int, window time.Duration) Decision {
	count := int(result[1])
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := time.Duration(result[2]) * time.Millisecond
	if retryAfter < 0 {
		retryAfter = 0
	}
	if result[0] == 1 {
		return Decision{Allowed: true, Remaining: remaining}
	}
	if retryAfter == 0 {
		retryAfter = window
	}
	return Decision{Allowed: false, Remaining: 0, RetryAfter: retryAfter}
}

func redisKeyPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "runtimeguard:"
	}
	if strings.HasSuffix(prefix, ":") {
		return prefix
	}
	return prefix + ":"
}

func durationMillis(d time.Duration) int64 {
	ms := d / time.Millisecond
	if ms <= 0 {
		return 1
	}
	return int64(ms)
}

func newLockToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(token[:]), nil
}

func redisValueInt(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, errors.New("runtimeguard: redis response is not an integer")
	}
}
