package job

import (
	"errors"
	"testing"
	"time"
)

func TestLeaseLifecycle(t *testing.T) {
	now := time.Date(2026, 4, 23, 1, 2, 3, 0, time.UTC)
	lease, err := NewLease("dispatch-webhooks", "worker-1", now, time.Minute)
	if err != nil {
		t.Fatalf("NewLease returned error: %v", err)
	}
	if lease.Expired(now.Add(30 * time.Second)) {
		t.Fatal("lease should not be expired before ttl")
	}
	if !lease.Expired(now.Add(time.Minute)) {
		t.Fatal("lease should be expired at ttl boundary")
	}
	if !lease.HeldBy("worker-1", now.Add(30*time.Second)) {
		t.Fatal("lease should be held by owner")
	}
}

func TestCanAcquire(t *testing.T) {
	now := time.Date(2026, 4, 23, 1, 2, 3, 0, time.UTC)
	lease, err := NewLease("job", "worker-1", now, time.Minute)
	if err != nil {
		t.Fatalf("NewLease returned error: %v", err)
	}
	if CanAcquire(&lease, "worker-2", now.Add(30*time.Second)) {
		t.Fatal("different owner should not acquire active lease")
	}
	if !CanAcquire(&lease, "worker-2", now.Add(2*time.Minute)) {
		t.Fatal("different owner should acquire expired lease")
	}
	if !CanAcquire(&lease, "worker-1", now.Add(30*time.Second)) {
		t.Fatal("same owner should renew active lease")
	}
}

func TestBackoffDelayCapsAtMax(t *testing.T) {
	policy := BackoffPolicy{Base: time.Second, Max: 10 * time.Second, Multiplier: 2}
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{10, 10 * time.Second},
	}
	for _, tc := range cases {
		got, err := policy.Delay(tc.attempt)
		if err != nil {
			t.Fatalf("Delay returned error: %v", err)
		}
		if got != tc.want {
			t.Fatalf("Delay(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestBackoffRejectsInvalidPolicy(t *testing.T) {
	_, err := BackoffPolicy{Base: 10 * time.Second, Max: time.Second, Multiplier: 2}.Delay(0)
	if !errors.Is(err, ErrInvalidBackoffPolicy) {
		t.Fatalf("Delay error = %v, want ErrInvalidBackoffPolicy", err)
	}
}
