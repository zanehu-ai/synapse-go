package webhook

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerifyHeader(t *testing.T) {
	now := time.Unix(1_777_000_000, 0)
	header, err := SignHeader("secret", now, []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("SignHeader returned error: %v", err)
	}
	if err := VerifyHeader("secret", header, []byte(`{"ok":true}`), now, 5*time.Minute); err != nil {
		t.Fatalf("VerifyHeader returned error: %v", err)
	}
}

func TestVerifyHeaderRejectsTamperedPayload(t *testing.T) {
	now := time.Unix(1_777_000_000, 0)
	header, err := SignHeader("secret", now, []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("SignHeader returned error: %v", err)
	}
	if err := VerifyHeader("secret", header, []byte(`{"ok":false}`), now, 5*time.Minute); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyHeader error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyHeaderRejectsTimestampSkew(t *testing.T) {
	now := time.Unix(1_777_000_000, 0)
	header, err := SignHeader("secret", now.Add(-10*time.Minute), []byte(`{}`))
	if err != nil {
		t.Fatalf("SignHeader returned error: %v", err)
	}
	if err := VerifyHeader("secret", header, []byte(`{}`), now, time.Minute); !errors.Is(err, ErrTimestampSkew) {
		t.Fatalf("VerifyHeader error = %v, want ErrTimestampSkew", err)
	}
}

func TestParseHeaderRejectsMalformedSignature(t *testing.T) {
	_, _, err := ParseHeader("t=123,v1=not-hex")
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("ParseHeader error = %v, want ErrInvalidSignature", err)
	}
}

func TestSignHeaderRejectsBlankSecret(t *testing.T) {
	_, err := SignHeader(" ", time.Now(), []byte(`{}`))
	if !errors.Is(err, ErrInvalidSecret) {
		t.Fatalf("SignHeader error = %v, want ErrInvalidSecret", err)
	}
}

func TestHeaderShape(t *testing.T) {
	header, err := SignHeader("secret", time.Unix(123, 0), []byte("body"))
	if err != nil {
		t.Fatalf("SignHeader returned error: %v", err)
	}
	if !strings.HasPrefix(header, "t=123,v1=") {
		t.Fatalf("header = %q, want t/v1 shape", header)
	}
}

func TestComputeSignatureDeterministic(t *testing.T) {
	got := ComputeSignature("secret", 123, []byte("body"))
	want := "fa1c6a291939b89044717d74b55fe23e137acf3edd31d2992540f037f035357a"
	if got != want {
		t.Fatalf("ComputeSignature() = %q, want %q", got, want)
	}
}
