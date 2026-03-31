package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── GenerateToken Tests ─────────────────────────────────────────

func TestGenerateToken_Valid(t *testing.T) {
	secret := "test-secret"
	claims := jwt.MapClaims{"user_id": float64(42), "role": "admin"}

	token, err := GenerateToken(claims, secret, 2*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}
	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// Verify the token is parseable
	parsed, err := ParseJWT(token, secret)
	if err != nil {
		t.Fatalf("ParseJWT(generated token) error: %v", err)
	}
	uid, _ := parsed["user_id"].(float64)
	if uid != 42 {
		t.Errorf("user_id = %v, want 42", parsed["user_id"])
	}
}

func TestGenerateToken_ContainsExpAndIat(t *testing.T) {
	secret := "test-secret"
	claims := jwt.MapClaims{"user_id": float64(1)}

	token, _ := GenerateToken(claims, secret, 1*time.Hour)
	parsed, _ := ParseJWT(token, secret)

	if _, ok := parsed["exp"]; !ok {
		t.Error("token missing exp claim")
	}
	if _, ok := parsed["iat"]; !ok {
		t.Error("token missing iat claim")
	}
}

// TC-HAPPY-AUTH-001: ExtractBearerToken with valid Bearer header
func TestExtractBearerToken_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer my-token-123")

	token, ok := ExtractBearerToken(c)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if token != "my-token-123" {
		t.Errorf("token = %q, want %q", token, "my-token-123")
	}
}

// TC-EXCEPTION-AUTH-001: ExtractBearerToken with missing header
func TestExtractBearerToken_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	_, ok := ExtractBearerToken(c)
	if ok {
		t.Fatal("expected ok=false for missing header")
	}
}

// TC-EXCEPTION-AUTH-002: ExtractBearerToken with Basic auth
func TestExtractBearerToken_BasicAuth(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Basic abc123")

	_, ok := ExtractBearerToken(c)
	if ok {
		t.Fatal("expected ok=false for Basic auth")
	}
}

// TC-EXCEPTION-AUTH-003: ExtractBearerToken with just "Bearer" (no token)
func TestExtractBearerToken_BearerOnly(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer")

	_, ok := ExtractBearerToken(c)
	if ok {
		t.Fatal("expected ok=false for Bearer without token")
	}
}

// TC-HAPPY-AUTH-002: ParseJWT with valid token
func TestParseJWT_Valid(t *testing.T) {
	secret := "test-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(42),
		"username": "alice",
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	claims, err := ParseJWT(signed, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uid, ok := claims["user_id"].(float64)
	if !ok || uid != 42 {
		t.Errorf("user_id = %v, want 42", claims["user_id"])
	}
	uname, ok := claims["username"].(string)
	if !ok || uname != "alice" {
		t.Errorf("username = %v, want alice", claims["username"])
	}
}

// TC-EXCEPTION-AUTH-004: ParseJWT with wrong secret
func TestParseJWT_WrongSecret(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": float64(1)})
	signed, _ := token.SignedString([]byte("real-secret"))

	_, err := ParseJWT(signed, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

// TC-EXCEPTION-AUTH-005: ParseJWT with garbage token
func TestParseJWT_InvalidToken(t *testing.T) {
	_, err := ParseJWT("not.a.jwt", "secret")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// TC-HAPPY-AUTH-003: LoginRateLimit passes when under limit
func TestLoginRateLimit_NilRedis_Passes(t *testing.T) {
	handler := LoginRateLimit(nil, 5, 0)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", nil)

	handler(c)

	if c.IsAborted() {
		t.Error("expected request to pass with nil redis")
	}
}

// TC-HAPPY-AUTH-004: IPRateLimit passes when under limit
func TestIPRateLimit_NilRedis_Passes(t *testing.T) {
	handler := IPRateLimit(nil, 60)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api", nil)

	handler(c)

	if c.IsAborted() {
		t.Error("expected request to pass with nil redis")
	}
}

// TC-HAPPY-AUTH-005: CORS middleware allows registered origin
func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	handler := CORSMiddleware("http://localhost:3000,http://example.com")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	handler(c)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("ACAO = %q, want %q", w.Header().Get("Access-Control-Allow-Origin"), "http://localhost:3000")
	}
}

// TC-EXCEPTION-AUTH-006: CORS middleware blocks unknown origin
func TestCORSMiddleware_BlockedOrigin(t *testing.T) {
	handler := CORSMiddleware("http://localhost:3000")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Origin", "http://evil.com")

	handler(c)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no ACAO header for blocked origin")
	}
}

// TC-HAPPY-AUTH-006: CORS OPTIONS returns 204
func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := CORSMiddleware("http://localhost:3000")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodOptions, "/", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	handler(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TC-HAPPY-AUTH-007: RequestID middleware generates UUID
func TestRequestIDMiddleware_GeneratesUUID(t *testing.T) {
	handler := RequestIDMiddleware()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler(c)

	traceID := c.GetString("trace_id")
	if traceID == "" {
		t.Error("expected non-empty trace_id")
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
}

// TC-HAPPY-AUTH-008: RequestID middleware preserves provided ID
func TestRequestIDMiddleware_PreservesProvided(t *testing.T) {
	handler := RequestIDMiddleware()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Request-ID", "my-custom-id")

	handler(c)

	if c.GetString("trace_id") != "my-custom-id" {
		t.Errorf("trace_id = %q, want %q", c.GetString("trace_id"), "my-custom-id")
	}
}

// ── Miniredis unit tests (no external Redis required) ───────────

func newMiniredis(t *testing.T) *redis.Client {
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

func TestLoginRateLimit_BlocksAfterMax(t *testing.T) {
	rdb := newMiniredis(t)
	handler := LoginRateLimit(rdb, 3, 15*time.Minute)

	r := gin.New()
	r.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bad creds"})
	})

	// 3 failed login attempts
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
	}

	// 4th should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestLoginRateLimit_ClearsOnSuccess(t *testing.T) {
	rdb := newMiniredis(t)
	handler := LoginRateLimit(rdb, 5, 15*time.Minute)

	// 2 failed logins
	rFail := gin.New()
	rFail.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{})
	})
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rFail.ServeHTTP(w, req)
	}

	// Successful login clears counter
	rOK := gin.New()
	rOK.POST("/login", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rOK.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Counter should be cleared — next failed attempts start from 0
	count, _ := rdb.Get(rdb.Context(), "login_fail:10.0.0.2").Int64()
	if count != 0 {
		t.Errorf("counter = %d after success, want 0", count)
	}
}

func TestIPRateLimit_BlocksOverLimit(t *testing.T) {
	rdb := newMiniredis(t)
	rpm := 3
	handler := IPRateLimit(rdb, rpm)

	r := gin.New()
	r.GET("/api", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Exhaust limit
	for i := 0; i < rpm; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		r.ServeHTTP(w, req)
	}

	// Next should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.0.0.3:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}
