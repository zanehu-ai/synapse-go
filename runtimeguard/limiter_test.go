package runtimeguard

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiterRejectsPastWindowCapacity(t *testing.T) {
	limiter := NewMemoryLimiter()
	limiter.now = func() time.Time { return time.Unix(100, 0) }
	for i := 0; i < 2; i++ {
		decision, err := limiter.Allow(context.Background(), "voucher:1", 2, time.Minute)
		if err != nil || !decision.Allowed {
			t.Fatalf("allow %d = %+v err=%v, want allowed", i, decision, err)
		}
	}
	decision, err := limiter.Allow(context.Background(), "voucher:1", 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.RetryAfter <= 0 {
		t.Fatalf("decision = %+v, want rejected with retry", decision)
	}
}

func TestLockManagerRejectsCompetingOwner(t *testing.T) {
	locks := NewLockManager()
	locks.now = func() time.Time { return time.Unix(100, 0) }
	unlock, ok := locks.TryLock("job:webhook", "worker-a", time.Minute)
	if !ok {
		t.Fatal("first lock should succeed")
	}
	if _, ok := locks.TryLock("job:webhook", "worker-b", time.Minute); ok {
		t.Fatal("second owner should be rejected")
	}
	unlock()
	if _, ok := locks.TryLock("job:webhook", "worker-b", time.Minute); !ok {
		t.Fatal("second owner should succeed after unlock")
	}
}
