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
