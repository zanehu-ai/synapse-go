package ginutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TC-HAPPY-GINUTIL-001: ParseIDParam with valid numeric ID
func TestParseIDParam_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/items/:id", func(c *gin.Context) {
		id, ok := ParseIDParam(c)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if id != 42 {
			t.Errorf("id = %d, want 42", id)
		}
	})

	c.Request = httptest.NewRequest("GET", "/items/42", nil)
	r.ServeHTTP(w, c.Request)
}

// TC-EXCEPTION-GINUTIL-001: ParseIDParam with non-numeric ID
func TestParseIDParam_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/items/:id", func(c *gin.Context) {
		_, ok := ParseIDParam(c)
		if ok {
			t.Fatal("expected ok=false for non-numeric id")
		}
	})

	c.Request = httptest.NewRequest("GET", "/items/abc", nil)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	code, ok := resp["code"].(float64)
	if !ok {
		t.Fatalf("expected code field, got %v", resp)
	}
	if code != 1001 {
		t.Errorf("error code = %v, want 1001", code)
	}
}

// TC-HAPPY-GINUTIL-002: ParseParam with custom name
func TestParseParam_CustomName(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/users/:user_id", func(c *gin.Context) {
		id, ok := ParseParam(c, "user_id")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if id != 99 {
			t.Errorf("user_id = %d, want 99", id)
		}
	})

	c.Request = httptest.NewRequest("GET", "/users/99", nil)
	r.ServeHTTP(w, c.Request)
}

// TC-BOUNDARY-GINUTIL-001: ParseIDParam with zero
func TestParseIDParam_Zero(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/items/:id", func(c *gin.Context) {
		id, ok := ParseIDParam(c)
		if !ok {
			t.Fatal("expected ok=true for 0")
		}
		if id != 0 {
			t.Errorf("id = %d, want 0", id)
		}
	})

	req := httptest.NewRequest("GET", "/items/0", nil)
	r.ServeHTTP(w, req)
}

// TC-BOUNDARY-GINUTIL-002: ParseIDParam with negative number
func TestParseIDParam_Negative(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/items/:id", func(c *gin.Context) {
		id, ok := ParseIDParam(c)
		if !ok {
			t.Fatal("expected ok=true for negative")
		}
		if id != -1 {
			t.Errorf("id = %d, want -1", id)
		}
	})

	req := httptest.NewRequest("GET", "/items/-1", nil)
	r.ServeHTTP(w, req)
}
