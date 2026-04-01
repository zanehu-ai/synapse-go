package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockMailer struct {
	sent []string
	err  error
}

func (m *mockMailer) Send(to, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, to+":"+subject)
	return nil
}

// failNMailer fails the first failCount calls, then succeeds.
type failNMailer struct {
	failCount int
	calls     int
}

func (m *failNMailer) Send(_, _, _ string) error {
	m.calls++
	if m.calls <= m.failCount {
		return errors.New("transient failure")
	}
	return nil
}

func TestEmailNotifier_Send(t *testing.T) {
	m := &mockMailer{}
	n := NewEmail(m)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if len(m.sent) != 1 || m.sent[0] != "a@b.com:hi" {
		t.Errorf("sent = %v", m.sent)
	}
}

func TestEmailNotifier_Error(t *testing.T) {
	m := &mockMailer{err: errors.New("smtp error")}
	n := NewEmail(m)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestWebhookNotifier_Send(t *testing.T) {
	var received bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected JSON content type")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhook(srv.URL)
	err := n.Send(context.Background(), Message{Subject: "test", Body: "body"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if !received {
		t.Error("webhook not called")
	}
}

func TestWebhookNotifier_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := NewWebhook(srv.URL)
	err := n.Send(context.Background(), Message{Subject: "test", Body: "body"})
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	m := &mockMailer{}
	n := WithRetry(NewEmail(m), 2)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if len(m.sent) != 1 {
		t.Errorf("expected 1 send, got %d", len(m.sent))
	}
}

func TestWithRetry_SucceedsAfterRetries(t *testing.T) {
	fm := &failNMailer{failCount: 2}
	n := WithRetry(NewEmail(fm), 3)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if fm.calls != 3 {
		t.Errorf("expected 3 calls (2 fail + 1 success), got %d", fm.calls)
	}
}

func TestWithRetry_AllRetriesExhausted(t *testing.T) {
	m := &mockMailer{err: errors.New("persistent failure")}
	n := WithRetry(NewEmail(m), 2)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err == nil {
		t.Error("expected error when all retries exhausted")
	}
}

func TestWithRetry_NegativeMaxRetries_StillSendsOnce(t *testing.T) {
	m := &mockMailer{}
	n := WithRetry(NewEmail(m), -1)

	err := n.Send(context.Background(), Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if len(m.sent) != 1 {
		t.Errorf("expected 1 send, got %d", len(m.sent))
	}
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	m := &mockMailer{err: errors.New("fail")}
	n := WithRetry(NewEmail(m), 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := n.Send(ctx, Message{To: "a@b.com", Subject: "hi", Body: "hello"})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
