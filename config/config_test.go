package config

import "testing"

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		setVal     string
		defaultVal string
		want       string
	}{
		// TC-HAPPY-CONFIG-001: env var set returns value
		{"env set", "TEST_818_SHARED_SET", "hello", "default", "hello"},
		// TC-HAPPY-CONFIG-002: env var not set returns default
		{"env not set", "TEST_818_SHARED_NOTSET_UNIQUE", "", "fallback", "fallback"},
		// TC-BOUNDARY-CONFIG-001: env var set to empty returns default
		{"env empty", "TEST_818_SHARED_EMPTY", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setVal != "" {
				t.Setenv(tt.key, tt.setVal)
			}
			got := GetEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnv(%q, %q) = %q, want %q", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

// TC-HAPPY-CONFIG-003: env var with spaces preserved
func TestGetEnv_PreservesSpaces(t *testing.T) {
	t.Setenv("TEST_818_SPACES", "  spaced  ")

	got := GetEnv("TEST_818_SPACES", "default")
	if got != "  spaced  " {
		t.Errorf("GetEnv should preserve spaces, got %q", got)
	}
}
