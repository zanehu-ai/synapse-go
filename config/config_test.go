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

func TestGetEnvInt(t *testing.T) {
	t.Setenv("APP_PORT", "8080")
	if got := GetEnvInt("APP_PORT", 3000); got != 8080 {
		t.Fatalf("GetEnvInt = %d, want 8080", got)
	}
	t.Setenv("APP_BAD_PORT", "bad")
	if got := GetEnvInt("APP_BAD_PORT", 3000); got != 3000 {
		t.Fatalf("GetEnvInt fallback = %d, want 3000", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	t.Setenv("APP_ENABLED", "true")
	if !GetEnvBool("APP_ENABLED", false) {
		t.Fatal("GetEnvBool = false, want true")
	}
	t.Setenv("APP_BAD_BOOL", "maybe")
	if got := GetEnvBool("APP_BAD_BOOL", true); !got {
		t.Fatal("GetEnvBool fallback = false, want true")
	}
}

func TestGetEnvCSV(t *testing.T) {
	t.Setenv("APP_CIDRS", "10.0.0.0/8, 192.0.2.0/24, ,")
	got := GetEnvCSV("APP_CIDRS")
	want := []string{"10.0.0.0/8", "192.0.2.0/24"}
	if len(got) != len(want) {
		t.Fatalf("GetEnvCSV len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("GetEnvCSV[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
