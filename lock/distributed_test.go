package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func newDistributedTestLocker(t *testing.T, opts ...DistributedOption) (DistributedLocker, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewDistributed(client, opts...), mr
}

func TestDistributedAcquireRelease(t *testing.T) {
	locker, mr := newDistributedTestLocker(t)
	ctx := context.Background()

	h, err := locker.Acquire(ctx, "balance:42", time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error: %v", err)
	}
	if h.Key() != "balance:42" {
		t.Fatalf("Key() = %q", h.Key())
	}
	if h.Token() == "" {
		t.Fatal("Token() should not be empty")
	}
	if !mr.Exists("synapse:lock:balance:42") {
		t.Fatal("lock key should exist")
	}
	if err := h.Release(ctx); err != nil {
		t.Fatalf("Release() error: %v", err)
	}
	if mr.Exists("synapse:lock:balance:42") {
		t.Fatal("lock key should be deleted")
	}
	if err := h.Release(ctx); !errors.Is(err, ErrAlreadyReleased) {
		t.Fatalf("second Release() = %v, want ErrAlreadyReleased", err)
	}
}

func TestDistributedAcquireContention(t *testing.T) {
	locker, _ := newDistributedTestLocker(t)
	ctx := context.Background()
	if _, err := locker.Acquire(ctx, "k", time.Minute); err != nil {
		t.Fatalf("first Acquire() error: %v", err)
	}
	if h, err := locker.Acquire(ctx, "k", time.Minute); !errors.Is(err, ErrLocked) || h != nil {
		t.Fatalf("second Acquire() handle=%v err=%v, want nil ErrLocked", h, err)
	}
}

func TestDistributedRenew(t *testing.T) {
	locker, mr := newDistributedTestLocker(t)
	ctx := context.Background()
	h, err := locker.Acquire(ctx, "renew", time.Second)
	if err != nil {
		t.Fatalf("Acquire() error: %v", err)
	}
	mr.FastForward(800 * time.Millisecond)
	if err := h.Renew(ctx, time.Minute); err != nil {
		t.Fatalf("Renew() error: %v", err)
	}
	mr.FastForward(2 * time.Second)
	if !mr.Exists("synapse:lock:renew") {
		t.Fatal("lock key should still exist after renew")
	}
}

func TestDistributedCustomPrefixAndInvalidArgs(t *testing.T) {
	locker, mr := newDistributedTestLocker(t, WithKeyPrefix("custom:"))
	ctx := context.Background()
	if _, err := locker.Acquire(ctx, "", time.Second); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("empty key err = %v, want ErrEmptyKey", err)
	}
	if _, err := locker.Acquire(ctx, "k", 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("zero ttl err = %v, want ErrInvalidTTL", err)
	}
	h, err := locker.Acquire(ctx, "k", time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error: %v", err)
	}
	if !mr.Exists("custom:k") {
		t.Fatal("custom prefix key should exist")
	}
	if err := h.Renew(ctx, 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("zero renew ttl err = %v, want ErrInvalidTTL", err)
	}
}

func TestDistributedAcquireRejectsNilRedisClient(t *testing.T) {
	locker := NewDistributed(nil)
	_, err := locker.Acquire(context.Background(), "job", time.Second)
	if !errors.Is(err, ErrNilClient) {
		t.Fatalf("Acquire error = %v, want ErrNilClient", err)
	}
}
