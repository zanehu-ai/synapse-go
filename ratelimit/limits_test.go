package ratelimit

import "testing"

// TC-HAPPY-LIMITS-001: DefaultLimits returns sensible defaults
func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.RPM != 60 {
		t.Errorf("RPM = %d, want 60", limits.RPM)
	}
	if limits.TPM != 100000 {
		t.Errorf("TPM = %d, want 100000", limits.TPM)
	}
	if limits.Concurrency != 10 {
		t.Errorf("Concurrency = %d, want 10", limits.Concurrency)
	}
}
