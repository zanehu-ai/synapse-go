package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSMS_Send(t *testing.T) {
	var received bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"Code": "OK", "Message": "OK"})
	}))
	defer srv.Close()

	provider := NewAliyunSMS(AliyunSMSConfig{
		Endpoint:     srv.URL,
		SignName:     "TestSign",
		TemplateCode: "SMS_001",
	})

	n := NewSMS(provider)
	err := n.Send(context.Background(), Message{To: "+8613800138000", Body: "123456"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if !received {
		t.Error("SMS provider not called")
	}
}

func TestNewSMS_AliyunError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"Code":    "isv.BUSINESS_LIMIT_CONTROL",
			"Message": "触发业务限流",
		})
	}))
	defer srv.Close()

	provider := NewAliyunSMS(AliyunSMSConfig{Endpoint: srv.URL})
	n := NewSMS(provider)

	err := n.Send(context.Background(), Message{To: "+8613800138000", Body: "123456"})
	if err == nil {
		t.Error("expected error for Aliyun business error")
	}
}

func TestNewSMS_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	provider := NewAliyunSMS(AliyunSMSConfig{Endpoint: srv.URL})
	n := NewSMS(provider)

	err := n.Send(context.Background(), Message{To: "+8613800138000", Body: "123456"})
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestNoopSMSProvider(t *testing.T) {
	provider := &NoopSMSProvider{}
	n := NewSMS(provider)

	err := n.Send(context.Background(), Message{To: "+8613800138000", Body: "test"})
	if err != nil {
		t.Errorf("NoopSMS should not error: %v", err)
	}
}

func TestAliyunSMS_UnmarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	provider := NewAliyunSMS(AliyunSMSConfig{Endpoint: srv.URL})
	err := provider.SendSMS(context.Background(), "+8613800138000", "123456")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestAliyunSMS_DefaultEndpoint(t *testing.T) {
	provider := NewAliyunSMS(AliyunSMSConfig{})
	p := provider.(*aliyunSMSProvider)
	if p.config.Endpoint != "https://dysmsapi.aliyuncs.com" {
		t.Errorf("endpoint = %q, want default", p.config.Endpoint)
	}
}
