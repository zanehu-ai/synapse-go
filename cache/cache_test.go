package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(s.Close)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestRedisCache_SetAndGet(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewRedis(rdb, "test")
	ctx := context.Background()

	if err := c.Set(ctx, "key1", "value1", 10*time.Second); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	val, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if val != "value1" {
		t.Errorf("Get() = %q, want %q", val, "value1")
	}
}

func TestRedisCache_GetMiss(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewRedis(rdb, "test")

	_, err := c.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for cache miss")
	}
}

func TestRedisCache_Del(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewRedis(rdb, "test")
	ctx := context.Background()

	if err := c.Set(ctx, "key1", "value1", 10*time.Second); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	if err := c.Del(ctx, "key1"); err != nil {
		t.Fatalf("Del() error: %v", err)
	}

	_, err := c.Get(ctx, "key1")
	if err == nil {
		t.Error("expected error after Del")
	}
}

func TestRedisCache_PrefixIsolation(t *testing.T) {
	rdb := newTestRedis(t)
	c1 := NewRedis(rdb, "app1")
	c2 := NewRedis(rdb, "app2")
	ctx := context.Background()

	if err := c1.Set(ctx, "key", "from-app1", 10*time.Second); err != nil {
		t.Fatalf("c1.Set() error: %v", err)
	}
	if err := c2.Set(ctx, "key", "from-app2", 10*time.Second); err != nil {
		t.Fatalf("c2.Set() error: %v", err)
	}

	v1, _ := c1.Get(ctx, "key")
	v2, _ := c2.Get(ctx, "key")

	if v1 != "from-app1" {
		t.Errorf("app1 = %q", v1)
	}
	if v2 != "from-app2" {
		t.Errorf("app2 = %q", v2)
	}
}

func TestGetOrLoad_CacheHit(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewRedis(rdb, "test")
	ctx := context.Background()

	if err := c.Set(ctx, "cached", "existing", 10*time.Second); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	loaded := false
	val, err := GetOrLoad(ctx, c, "cached", 10*time.Second, func() (string, error) {
		loaded = true
		return "new", nil
	})

	if err != nil {
		t.Fatalf("GetOrLoad() error: %v", err)
	}
	if loaded {
		t.Error("loader should not be called on cache hit")
	}
	if val != "existing" {
		t.Errorf("val = %q, want %q", val, "existing")
	}
}

func TestGetOrLoad_CacheMiss(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewRedis(rdb, "test")
	ctx := context.Background()

	val, err := GetOrLoad(ctx, c, "miss", 10*time.Second, func() (string, error) {
		return "loaded", nil
	})

	if err != nil {
		t.Fatalf("GetOrLoad() error: %v", err)
	}
	if val != "loaded" {
		t.Errorf("val = %q, want %q", val, "loaded")
	}

	// Should be cached now
	cached, _ := c.Get(ctx, "miss")
	if cached != "loaded" {
		t.Errorf("cached = %q, want %q", cached, "loaded")
	}
}
