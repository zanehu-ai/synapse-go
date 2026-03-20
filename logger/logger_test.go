package logger

import "testing"

// TC-HAPPY-LOGGER-001: New returns non-nil logger in development
func TestNew_Development(t *testing.T) {
	log := New("development")
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TC-HAPPY-LOGGER-002: New returns non-nil logger in production
func TestNew_Production(t *testing.T) {
	log := New("production")
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TC-HAPPY-LOGGER-003: package-level functions don't panic
func TestPackageLevelFunctions_NoPanic(t *testing.T) {
	New("development") // initialize first
	Info("test info")
	Error("test error")
	Warn("test warn")
	Debug("test debug")
}

// TC-HAPPY-LOGGER-004: field helpers return correct zap.Field types
func TestFieldHelpers(t *testing.T) {
	f1 := String("key", "val")
	if f1.Key != "key" {
		t.Errorf("String key = %q, want %q", f1.Key, "key")
	}

	f2 := Int("count", 42)
	if f2.Key != "count" || f2.Integer != 42 {
		t.Errorf("Int field unexpected: %v", f2)
	}

	f3 := Int64("id", 100)
	if f3.Key != "id" || f3.Integer != 100 {
		t.Errorf("Int64 field unexpected: %v", f3)
	}

	f4 := Bool("flag", true)
	if f4.Key != "flag" {
		t.Errorf("Bool key = %q, want %q", f4.Key, "flag")
	}

	f5 := Any("data", map[string]int{"a": 1})
	if f5.Key != "data" {
		t.Errorf("Any key = %q, want %q", f5.Key, "data")
	}
}

// TC-BOUNDARY-LOGGER-001: New overwrites previous logger
func TestNew_Overwrites(t *testing.T) {
	log1 := New("development")
	log2 := New("production")

	if log1 == log2 {
		t.Error("expected different logger instances")
	}
	// After second New(), global should return production logger
	got := get()
	if got != log2 {
		t.Error("expected get() to return latest logger")
	}
}
