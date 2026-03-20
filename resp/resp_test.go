package resp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TC-HAPPY-RESP-001: Success returns 200 with code=0
func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	Success(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != CodeSuccess {
		t.Errorf("code = %d, want %d", resp.Code, CodeSuccess)
	}
	if resp.Message != "success" {
		t.Errorf("message = %q, want %q", resp.Message, "success")
	}
}

// TC-HAPPY-RESP-002: Created returns 201
func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)

	Created(c, gin.H{"id": 1})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

// TC-HAPPY-RESP-003: Error returns custom status and code
func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	Error(c, http.StatusBadRequest, CodeBadRequest, "bad input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != CodeBadRequest {
		t.Errorf("code = %d, want %d", resp.Code, CodeBadRequest)
	}
}

// TC-HAPPY-RESP-004: InternalError returns 500 with CodeInternal
func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	InternalError(c, errors.New("db connection failed"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != CodeInternal {
		t.Errorf("code = %d, want %d", resp.Code, CodeInternal)
	}
	if resp.Message != "internal server error" {
		t.Errorf("message = %q, want %q", resp.Message, "internal server error")
	}
}

// TC-HAPPY-RESP-005: SuccessPage includes pagination
func TestSuccessPage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	SuccessPage(c, []string{"a", "b"}, 100, 1, 20)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp.Data.(map[string]interface{})
	if data["total"].(float64) != 100 {
		t.Errorf("total = %v, want 100", data["total"])
	}
	if data["page"].(float64) != 1 {
		t.Errorf("page = %v, want 1", data["page"])
	}
	if data["page_size"].(float64) != 20 {
		t.Errorf("page_size = %v, want 20", data["page_size"])
	}
}

// TC-HAPPY-RESP-006: trace_id included when set
func TestSuccess_WithTraceID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("trace_id", "abc-123")

	Success(c, nil)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TraceID != "abc-123" {
		t.Errorf("trace_id = %q, want %q", resp.TraceID, "abc-123")
	}
}

// TC-HAPPY-RESP-007: ParsePageParams defaults
func TestParsePageParams_Defaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	pp := ParsePageParams(c)
	if pp.Page != 1 {
		t.Errorf("Page = %d, want 1", pp.Page)
	}
	if pp.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", pp.PageSize)
	}
}

// TC-BOUNDARY-RESP-001: ParsePageParams clamps invalid values
func TestParsePageParams_ClampsInvalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?page=-1&page_size=999", nil)

	pp := ParsePageParams(c)
	if pp.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped)", pp.Page)
	}
	if pp.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20 (clamped from 999)", pp.PageSize)
	}
}

// TC-BOUNDARY-RESP-002: response codes have expected values
func TestResponseCodes(t *testing.T) {
	if CodeSuccess != 0 {
		t.Errorf("CodeSuccess = %d, want 0", CodeSuccess)
	}
	if CodeBadRequest != 1001 {
		t.Errorf("CodeBadRequest = %d, want 1001", CodeBadRequest)
	}
	if CodeNotFound != 1002 {
		t.Errorf("CodeNotFound = %d, want 1002", CodeNotFound)
	}
	if CodeInternal != 5001 {
		t.Errorf("CodeInternal = %d, want 5001", CodeInternal)
	}
	if CodeUnauthorized != 2001 {
		t.Errorf("CodeUnauthorized = %d, want 2001", CodeUnauthorized)
	}
}
