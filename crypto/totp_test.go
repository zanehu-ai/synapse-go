package crypto

import (
	"testing"
	"time"
)

func TestTOTPCodeRFCVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if got := TOTPCode(secret, time.Unix(59, 0)); got != "287082" {
		t.Fatalf("TOTPCode = %q, want 287082", got)
	}
}

func TestValidateTOTPWindow(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Unix(90, 0)
	code := TOTPCode(secret, now.Add(-30*time.Second))
	if !ValidateTOTP(secret, code, now, 1) {
		t.Fatal("ValidateTOTP should accept previous window")
	}
	if ValidateTOTP(secret, code, now, 0) {
		t.Fatal("ValidateTOTP should reject previous window when windowSteps=0")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" || TOTPCode(secret, time.Now()) == "" {
		t.Fatalf("generated secret is not usable: %q", secret)
	}
}
