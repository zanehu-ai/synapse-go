package runtimeguard

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRedisLimiterRejectsPastWindowCapacity(t *testing.T) {
	store := newFakeRuntimeRedis()
	limiter := NewRedisLimiter(store, RedisLimiterOptions{KeyPrefix: "test:"})

	for i := 0; i < 2; i++ {
		decision, err := limiter.Allow(context.Background(), "voucher:tenant:1", 2, time.Minute)
		if err != nil || !decision.Allowed {
			t.Fatalf("allow %d = %+v err=%v, want allowed", i, decision, err)
		}
	}

	decision, err := limiter.Allow(context.Background(), "voucher:tenant:1", 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Remaining != 0 || decision.RetryAfter != time.Minute {
		t.Fatalf("decision = %+v, want rejected with one-minute retry", decision)
	}
}

func TestRedisLimiterReturnsErrorWhenRedisUnavailable(t *testing.T) {
	redisErr := errors.New("redis unavailable")
	limiter := NewRedisLimiter(&fakeRuntimeRedis{err: redisErr}, RedisLimiterOptions{})

	decision, err := limiter.Allow(context.Background(), "customer_auth:ip:203.0.113.10", 1, time.Minute)
	if !errors.Is(err, redisErr) {
		t.Fatalf("err = %v, want %v", err, redisErr)
	}
	if decision.Allowed {
		t.Fatalf("decision = %+v, want fail-closed rejection", decision)
	}
}

func TestRedisLockManagerRequiresReleaseToken(t *testing.T) {
	store := newFakeRuntimeRedis()
	locks := NewRedisLockManager(store, RedisLockOptions{KeyPrefix: "test:"})

	lock, ok, err := locks.Acquire(context.Background(), "job:webhook", "worker-a", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first acquire = lock=%+v ok=%v err=%v, want acquired", lock, ok, err)
	}
	if lock.Token == "" {
		t.Fatal("lock token should not be empty")
	}

	if _, ok, err := locks.Acquire(context.Background(), "job:webhook", "worker-b", time.Minute); err != nil || ok {
		t.Fatalf("competing acquire ok=%v err=%v, want not acquired without error", ok, err)
	}

	released, err := locks.Release(context.Background(), "job:webhook", "wrong-token")
	if err != nil {
		t.Fatal(err)
	}
	if released {
		t.Fatal("release with wrong token should not delete lock")
	}
	if _, ok, err := locks.Acquire(context.Background(), "job:webhook", "worker-b", time.Minute); err != nil || ok {
		t.Fatalf("acquire after wrong release ok=%v err=%v, want still locked", ok, err)
	}

	released, err = lock.Release(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !released {
		t.Fatal("release with owner token should delete lock")
	}
	if _, ok, err := locks.Acquire(context.Background(), "job:webhook", "worker-b", time.Minute); err != nil || !ok {
		t.Fatalf("acquire after release ok=%v err=%v, want acquired", ok, err)
	}
}

func TestRedisLockManagerReturnsErrorWhenRedisUnavailable(t *testing.T) {
	redisErr := errors.New("redis unavailable")
	locks := NewRedisLockManager(&fakeRuntimeRedis{err: redisErr}, RedisLockOptions{})

	lock, ok, err := locks.Acquire(context.Background(), "job:webhook", "worker-a", time.Minute)
	if !errors.Is(err, redisErr) {
		t.Fatalf("err = %v, want %v", err, redisErr)
	}
	if ok || lock.Token != "" {
		t.Fatalf("lock=%+v ok=%v, want not acquired", lock, ok)
	}
}

type fakeRuntimeRedis struct {
	mu       sync.Mutex
	counters map[string]fakeCounter
	locks    map[string]string
	err      error
}

type fakeCounter struct {
	count int64
	ttl   time.Duration
}

func newFakeRuntimeRedis() *fakeRuntimeRedis {
	return &fakeRuntimeRedis{
		counters: map[string]fakeCounter{},
		locks:    map[string]string{},
	}
}

func (f *fakeRuntimeRedis) EvalIntSlice(_ context.Context, _ string, keys []string, args ...string) ([]int64, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	window, _ := strconv.ParseInt(args[0], 10, 64)
	limit, _ := strconv.ParseInt(args[1], 10, 64)
	counter := f.counters[keys[0]]
	if counter.count >= limit {
		return []int64{0, counter.count, int64(counter.ttl / time.Millisecond)}, nil
	}
	counter.count++
	counter.ttl = time.Duration(window) * time.Millisecond
	f.counters[keys[0]] = counter
	return []int64{1, counter.count, window}, nil
}

func (f *fakeRuntimeRedis) EvalInt(_ context.Context, _ string, keys []string, args ...string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.locks[keys[0]] != args[0] {
		return 0, nil
	}
	delete(f.locks, keys[0])
	return 1, nil
}

func (f *fakeRuntimeRedis) SetNX(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.locks[key]; ok {
		return false, nil
	}
	f.locks[key] = value
	return true, nil
}
