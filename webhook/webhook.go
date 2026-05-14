// FROZEN: This file is the reference implementation for cross-language signing parity.
// Do NOT modify signing logic (ComputeSignature, SignHeader wire format, or the
// "timestamp." separator). Any algorithm change is a v2 migration requiring a new
// SignatureVersion constant and a separate ADR.
// Last frozen: 2026-05-08, commit tied to feat/phase-a-w2-webhook-outbound.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SignatureVersion = "v1"
	TimestampPrefix  = "t="
	SignaturePrefix  = SignatureVersion + "="
)

var (
	ErrInvalidSecret    = errors.New("webhook: invalid secret")
	ErrInvalidSignature = errors.New("webhook: invalid signature")
	ErrTimestampSkew    = errors.New("webhook: timestamp outside tolerance")
)

func SignHeader(secret string, timestamp time.Time, payload []byte) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrInvalidSecret
	}
	unix := timestamp.Unix()
	return fmt.Sprintf("t=%d,v1=%s", unix, ComputeSignature(secret, unix, payload)), nil
}

func ComputeSignature(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyHeader(secret string, header string, payload []byte, now time.Time, tolerance time.Duration) error {
	if strings.TrimSpace(secret) == "" {
		return ErrInvalidSecret
	}
	timestamp, signature, err := ParseHeader(header)
	if err != nil {
		return err
	}
	if tolerance > 0 && absDuration(now.Sub(time.Unix(timestamp, 0))) > tolerance {
		return ErrTimestampSkew
	}
	expected := ComputeSignature(secret, timestamp, payload)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

func ParseHeader(header string) (int64, string, error) {
	var timestamp int64
	var signature string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, TimestampPrefix):
			parsed, err := strconv.ParseInt(strings.TrimPrefix(part, TimestampPrefix), 10, 64)
			if err != nil || parsed <= 0 {
				return 0, "", ErrInvalidSignature
			}
			timestamp = parsed
		case strings.HasPrefix(part, SignaturePrefix):
			signature = strings.TrimPrefix(part, SignaturePrefix)
		}
	}
	if timestamp == 0 || signature == "" {
		return 0, "", ErrInvalidSignature
	}
	if _, err := hex.DecodeString(signature); err != nil {
		return 0, "", ErrInvalidSignature
	}
	return timestamp, signature, nil
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
