package timeutil

import (
	"testing"
	"time"
)

// TC-HAPPY-TIMEUTIL-001: both from and to provided
func TestParseDateRange_BothProvided(t *testing.T) {
	from, to, err := ParseDateRange("2024-01-15T00:00:00Z", "2024-01-31T23:59:59Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Day() != 15 || from.Month() != time.January {
		t.Errorf("from = %v, want 2024-01-15", from)
	}
	if to.Day() != 31 {
		t.Errorf("to = %v, want 2024-01-31", to)
	}
}

// TC-HAPPY-TIMEUTIL-002: empty strings default to current month
func TestParseDateRange_EmptyDefaults(t *testing.T) {
	from, to, err := ParseDateRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	if from.Day() != 1 || from.Month() != now.Month() || from.Year() != now.Year() {
		t.Errorf("from should be first day of current month, got %v", from)
	}
	if to.Before(from) {
		t.Errorf("to (%v) should be after from (%v)", to, from)
	}
}

// TC-HAPPY-TIMEUTIL-003: only from provided
func TestParseDateRange_OnlyFrom(t *testing.T) {
	from, to, err := ParseDateRange("2024-06-10T00:00:00Z", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Month() != time.June || from.Day() != 10 {
		t.Errorf("from = %v, want 2024-06-10", from)
	}
	// to should still be end of current month
	now := time.Now()
	if to.Month() != now.Month() {
		t.Errorf("to month = %v, want %v", to.Month(), now.Month())
	}
}

// TC-EXCEPTION-TIMEUTIL-001: invalid from format
func TestParseDateRange_InvalidFrom(t *testing.T) {
	_, _, err := ParseDateRange("not-a-date", "")
	if err == nil {
		t.Error("expected error for invalid from format")
	}
}

// TC-EXCEPTION-TIMEUTIL-002: invalid to format
func TestParseDateRange_InvalidTo(t *testing.T) {
	_, _, err := ParseDateRange("", "2024/01/01")
	if err == nil {
		t.Error("expected error for invalid to format")
	}
}

// TC-BOUNDARY-TIMEUTIL-001: end of month boundary
func TestParseDateRange_EndOfMonth(t *testing.T) {
	from, to, err := ParseDateRange("2024-02-01T00:00:00Z", "2024-02-29T23:59:59Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Month() != time.February || to.Month() != time.February {
		t.Errorf("expected February, got from=%v to=%v", from.Month(), to.Month())
	}
	if to.Day() != 29 {
		t.Errorf("expected day 29 (leap year), got %d", to.Day())
	}
}
