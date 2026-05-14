// Package jobpayload contains small helpers for typed job payload values.
package jobpayload

import "encoding/json"

// Int reads an integer value from a job payload, accepting common JSON numeric
// shapes used by manual job runs and decoded API payloads.
func Int(payload map[string]any, key string, def int) int {
	if payload == nil {
		return def
	}
	switch v := payload[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return int(n)
		}
	}
	return def
}
