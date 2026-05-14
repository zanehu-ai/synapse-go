package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(rdb.Context()).Err(); err != nil {
		t.Fatalf("redis not available: %v", err)
	}
	return rdb
}

// TC-HAPPY-AUTH-INT-001: LoginRateLimit allows requests under limit
func TestLoginRateLimit_Redis_AllowsUnderLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("requires external service")
	}
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()

	// Clean up test keys
	rdb.Del(rdb.Context(), "login_fail:192.0.2.1")
	defer rdb.Del(rdb.Context(), "login_fail:192.0.2.1")

	handler := LoginRateLimit(rdb, 5, 15*time.Minute)

	r := gin.New()
	r.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TC-HAPPY-AUTH-INT-002: LoginRateLimit blocks after max attempts
func TestLoginRateLimit_Redis_BlocksAfterMax(t *testing.T) {
	if testing.Short() {
		t.Skip("requires external service")
	}
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()

	key := "login_fail:192.0.2.2"
	rdb.Del(rdb.Context(), key)
	defer rdb.Del(rdb.Context(), key)

	handler := LoginRateLimit(rdb, 3, 15*time.Minute)

	r := gin.New()
	r.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bad creds"})
	})

	// Simulate 3 failed logins
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.0.2.2:12345"
		r.ServeHTTP(w, req)
	}

	// 4th attempt should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.0.2.2:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

// TC-HAPPY-AUTH-INT-003: LoginRateLimit clears counter on success
func TestLoginRateLimit_Redis_ClearsOnSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("requires external service")
	}
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()

	key := "login_fail:192.0.2.3"
	rdb.Del(rdb.Context(), key)
	defer rdb.Del(rdb.Context(), key)

	handler := LoginRateLimit(rdb, 5, 15*time.Minute)

	// Simulate 2 failed logins
	rFail := gin.New()
	rFail.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{})
	})
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.0.2.3:12345"
		rFail.ServeHTTP(w, req)
	}

	// Successful login should clear counter
	rOK := gin.New()
	rOK.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.0.2.3:12345"
	rOK.ServeHTTP(w, req)

	count, _ := rdb.Get(rdb.Context(), key).Int64()
	if count != 0 {
		t.Errorf("counter = %d after success, want 0", count)
	}
}

// TC-HAPPY-AUTH-INT-004: IPRateLimit allows requests under RPM
func TestIPRateLimit_Redis_AllowsUnderLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("requires external service")
	}
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()

	rdb.Del(rdb.Context(), "ip_rl:192.0.2.10")
	defer rdb.Del(rdb.Context(), "ip_rl:192.0.2.10")

	handler := IPRateLimit(rdb, 60)

	r := gin.New()
	r.GET("/api", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TC-HAPPY-AUTH-INT-005: IPRateLimit blocks after exceeding RPM
func TestIPRateLimit_Redis_BlocksOverLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("requires external service")
	}
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()

	key := "ip_rl:192.0.2.11"
	rdb.Del(rdb.Context(), key)
	defer rdb.Del(rdb.Context(), key)

	rpm := 3
	handler := IPRateLimit(rdb, rpm)

	r := gin.New()
	r.GET("/api", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Exhaust the limit
	for i := 0; i < rpm; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api", nil)
		req.RemoteAddr = "192.0.2.11:12345"
		r.ServeHTTP(w, req)
	}

	// Next request should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "192.0.2.11:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}
