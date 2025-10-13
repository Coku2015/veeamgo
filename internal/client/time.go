package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// APITime unmarshals timestamps returned by the VBR REST API, accepting both
// RFC3339 / RFC3339Nano values and fractional timestamps without an explicit timezone.
type APITime struct {
	time.Time
}

func (t *APITime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || strings.EqualFold(s, "null") {
		t.Time = time.Time{}
		return nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
		if parsed, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("parse api time: %q", s)
}

func (t APITime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}

// Ptr returns a pointer to a copy of the time.Time value or nil when zero.
func (t *APITime) Ptr() *time.Time {
	if t == nil {
		return nil
	}
	if t.Time.IsZero() {
		return nil
	}
	clone := t.Time
	return &clone
}
