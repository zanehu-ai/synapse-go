package notification

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateMessageAllowsMinimalValidMessage(t *testing.T) {
	msg := Message{
		TenantID:             "t-1",
		RecipientPrincipalID: "p-1",
		Type:                 "Platform.File.Completed",
		Subject:              "Upload complete",
		Payload:              json.RawMessage(`{"file_id":"f-1"}`),
	}
	if err := ValidateMessage(msg, ValidationLimits{}); err != nil {
		t.Fatalf("ValidateMessage returned error: %v", err)
	}
}

func TestValidateMessageRejectsInvalidType(t *testing.T) {
	msg := Message{TenantID: "t-1", RecipientPrincipalID: "p-1", Type: "1.bad"}
	if err := ValidateMessage(msg, ValidationLimits{}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("ValidateMessage error = %v, want ErrInvalidMessage", err)
	}
}

func TestValidateMessageRejectsInvalidPayload(t *testing.T) {
	msg := Message{
		TenantID:             "t-1",
		RecipientPrincipalID: "p-1",
		Type:                 "platform.file.completed",
		Payload:              json.RawMessage(`{"broken"`),
	}
	if err := ValidateMessage(msg, ValidationLimits{}); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("ValidateMessage error = %v, want ErrInvalidPayload", err)
	}
}

func TestValidateMessageAppliesPayloadLimit(t *testing.T) {
	msg := Message{
		TenantID:             "t-1",
		RecipientPrincipalID: "p-1",
		Type:                 "platform.file.completed",
		Payload:              json.RawMessage(`{"x":"12345"}`),
	}
	if err := ValidateMessage(msg, ValidationLimits{MaxPayloadBytes: 4}); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("ValidateMessage error = %v, want ErrInvalidPayload", err)
	}
}
