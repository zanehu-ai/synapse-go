package notification

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultMaxTypeLength    = 128
	DefaultMaxPayloadBytes  = 16 << 10
	DefaultMaxSubjectLength = 200
)

var (
	ErrInvalidMessage = errors.New("notification: invalid message")
	ErrInvalidPayload = errors.New("notification: invalid payload")
)

type Message struct {
	TenantID             string
	RecipientPrincipalID string
	Type                 string
	Subject              string
	Payload              json.RawMessage
	CreatedAt            time.Time
}

type ValidationLimits struct {
	MaxTypeLength    int
	MaxPayloadBytes  int
	MaxSubjectLength int
}

var typePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

func ValidateMessage(msg Message, limits ValidationLimits) error {
	limits = limits.withDefaults()
	if strings.TrimSpace(msg.TenantID) == "" || strings.TrimSpace(msg.RecipientPrincipalID) == "" {
		return ErrInvalidMessage
	}
	msgType := NormalizeType(msg.Type)
	if msgType == "" || len(msgType) > limits.MaxTypeLength || !typePattern.MatchString(msgType) {
		return ErrInvalidMessage
	}
	if len([]rune(strings.TrimSpace(msg.Subject))) > limits.MaxSubjectLength {
		return ErrInvalidMessage
	}
	if len(msg.Payload) > limits.MaxPayloadBytes {
		return ErrInvalidPayload
	}
	if len(msg.Payload) > 0 && !json.Valid(msg.Payload) {
		return ErrInvalidPayload
	}
	return nil
}

func NormalizeType(msgType string) string {
	return strings.ToLower(strings.TrimSpace(msgType))
}

func (l ValidationLimits) withDefaults() ValidationLimits {
	if l.MaxTypeLength == 0 {
		l.MaxTypeLength = DefaultMaxTypeLength
	}
	if l.MaxPayloadBytes == 0 {
		l.MaxPayloadBytes = DefaultMaxPayloadBytes
	}
	if l.MaxSubjectLength == 0 {
		l.MaxSubjectLength = DefaultMaxSubjectLength
	}
	return l
}
