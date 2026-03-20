package mailer

import "testing"

// TC-HAPPY-MAILER-001: New with empty host returns NoopMailer
func TestNew_EmptyHost(t *testing.T) {
	m := New("", "587", "", "", "")
	if _, ok := m.(*NoopMailer); !ok {
		t.Errorf("expected NoopMailer, got %T", m)
	}
}

// TC-HAPPY-MAILER-002: New with host returns SMTPMailer
func TestNew_WithHost(t *testing.T) {
	m := New("smtp.example.com", "587", "user", "pass", "from@example.com")
	if _, ok := m.(*SMTPMailer); !ok {
		t.Errorf("expected SMTPMailer, got %T", m)
	}
}

// TC-HAPPY-MAILER-003: NoopMailer.Send returns nil
func TestNoopMailer_Send(t *testing.T) {
	m := &NoopMailer{}
	if err := m.Send("to@test.com", "subject", "body"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
