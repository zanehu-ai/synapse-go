package jobpayload

import (
	"encoding/json"
	"testing"
)

func TestIntAcceptsCommonJSONNumberShapes(t *testing.T) {
	payload := map[string]any{
		"int":   3,
		"int64": int64(4),
		"float": 5.0,
		"json":  json.Number("6"),
	}
	for key, want := range map[string]int{"int": 3, "int64": 4, "float": 5, "json": 6, "missing": 9} {
		if got := Int(payload, key, 9); got != want {
			t.Fatalf("Int(%q) = %d, want %d", key, got, want)
		}
	}
}

func TestIntRejectsOverflowAndFractionalValues(t *testing.T) {
	payload := map[string]any{
		"overflow": json.Number("9223372036854775808"),
		"fraction": 1.5,
		"jsonfrac": json.Number("1.5"),
	}
	for _, key := range []string{"overflow", "fraction", "jsonfrac"} {
		if got := Int(payload, key, 7); got != 7 {
			t.Fatalf("Int(%q) = %d, want default", key, got)
		}
	}
}
