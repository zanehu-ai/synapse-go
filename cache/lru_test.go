package cache

import (
	"testing"
	"time"
)

func TestLRU_GetAfterSet(t *testing.T) {
	c := New[string, int](4, time.Minute)
	c.Set("a", 1)
	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Fatalf("expected 1,true; got %d,%v", v, ok)
	}
}

func TestLRU_MissReturnsFalse(t *testing.T) {
	c := New[string, int](4, time.Minute)
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}

func TestLRU_TTLExpiry(t *testing.T) {
	c := New[string, int](4, 50*time.Millisecond)
	now := time.Now()
	c.now = func() time.Time { return now }
	c.Set("a", 1)
	c.now = func() time.Time { return now.Add(60 * time.Millisecond) }
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected expired miss")
	}
	if c.Len() != 0 {
		t.Fatalf("expected 0 entries after expiry get, got %d", c.Len())
	}
}

func TestLRU_ZeroTTLNeverExpires(t *testing.T) {
	c := New[string, int](4, 0)
	now := time.Now()
	c.now = func() time.Time { return now }
	c.Set("a", 1)
	c.now = func() time.Time { return now.Add(24 * time.Hour) }
	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Fatalf("expected non-expiring entry, got %d,%v", v, ok)
	}
}

func TestLRU_EvictsLeastRecentlyUsed(t *testing.T) {
	c := New[string, int](2, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Get("a")    // a most recently used
	c.Set("c", 3) // should evict b
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected b evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected a retained")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected c retained")
	}
}

func TestLRU_SetUpdatesValueAndRefreshesTTL(t *testing.T) {
	c := New[string, int](4, 100*time.Millisecond)
	now := time.Now()
	c.now = func() time.Time { return now }
	c.Set("a", 1)
	c.now = func() time.Time { return now.Add(60 * time.Millisecond) }
	c.Set("a", 2)
	c.now = func() time.Time { return now.Add(150 * time.Millisecond) }
	v, ok := c.Get("a")
	if !ok || v != 2 {
		t.Fatalf("expected refreshed value 2,true; got %d,%v", v, ok)
	}
}

func TestLRU_Delete(t *testing.T) {
	c := New[string, int](4, time.Minute)
	c.Set("a", 1)
	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected deleted")
	}
	c.Delete("nonexistent") // no panic
}

func TestLRU_Purge(t *testing.T) {
	c := New[string, int](4, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Purge()
	if c.Len() != 0 {
		t.Fatalf("expected 0 after purge, got %d", c.Len())
	}
}

func TestLRU_ZeroCapacityDisables(t *testing.T) {
	c := New[string, int](0, time.Minute)
	c.Set("a", 1)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected disabled cache to miss")
	}
	if c.Len() != 0 {
		t.Fatal("expected zero len")
	}
}
