package timeutil

import (
	"fmt"
	"time"
)

// ParseDateRange parses "from" and "to" RFC3339 strings.
// Defaults to the current calendar month if empty.
func ParseDateRange(fromStr, toStr string) (from, to time.Time, err error) {
	now := time.Now()
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to = from.AddDate(0, 1, 0).Add(-time.Second)

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return from, to, fmt.Errorf("invalid from format, use RFC3339")
		}
		from = t
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return from, to, fmt.Errorf("invalid to format, use RFC3339")
		}
		to = t
	}
	return from, to, nil
}
