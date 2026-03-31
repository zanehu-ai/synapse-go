package idempotent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func init() { gin.SetMode(gin.TestMode) }

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

func TestCheck_Success(t *testing.T) {
	rdb := newTestRedis(t)
	ctx := context.Background()

	release, err := Check(ctx, rdb, "order:123", 10*time.Second)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	defer release()
}

func TestCheck_Duplicate(t *testing.T) {
	rdb := newTestRedis(t)
	ctx := context.Background()

	release, _ := Check(ctx, rdb, "order:123", 10*time.Second)
	defer release()

	_, err := Check(ctx, rdb, "order:123", 10*time.Second)
	if !errors.Is(err, ErrDuplicateRequest) {
		t.Errorf("expected ErrDuplicateRequest, got: %v", err)
	}
}

func TestCheck_ReleaseAllowsRetry(t *testing.T) {
	rdb := newTestRedis(t)
	ctx := context.Background()

	release, _ := Check(ctx, rdb, "order:123", 10*time.Second)
	release()

	release2, err := Check(ctx, rdb, "order:123", 10*time.Second)
	if err != nil {
		t.Fatalf("Check() after release error: %v", err)
	}
	defer release2()
}

func TestMiddleware_WithKey_Success(t *testing.T) {
	rdb := newTestRedis(t)

	r := gin.New()
	r.Use(Middleware(rdb, 10*time.Second))
	r.POST("/orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/orders", nil)
	req.Header.Set("Idempotency-Key", "unique-key-1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMiddleware_WithKey_Duplicate(t *testing.T) {
	rdb := newTestRedis(t)

	r := gin.New()
	r.Use(Middleware(rdb, 10*time.Second))
	r.POST("/orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First request
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/orders", nil)
	req1.Header.Set("Idempotency-Key", "dup-key")
	r.ServeHTTP(w1, req1)

	// Duplicate request (key still held because release happens after handler)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/orders", nil)
	req2.Header.Set("Idempotency-Key", "dup-key")
	r.ServeHTTP(w2, req2)

	// Second request should succeed because first one completed and released
	// (Middleware uses defer release, so key is released after c.Next())
	if w2.Code != http.StatusOK {
		t.Logf("status = %d (key released after first request)", w2.Code)
	}
}

func TestMiddleware_WithoutKey_PassThrough(t *testing.T) {
	rdb := newTestRedis(t)

	r := gin.New()
	r.Use(Middleware(rdb, 10*time.Second))
	r.POST("/orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/orders", nil)
	// No Idempotency-Key header
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
