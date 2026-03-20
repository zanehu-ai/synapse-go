package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
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
	if claims["user_id"].(float64) != 42 {
		t.Errorf("user_id = %v, want 42", claims["user_id"])
	}
	if claims["username"].(string) != "alice" {
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
