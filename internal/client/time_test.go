package client

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAPITimeUnmarshalFlexibleFormats(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		layout string
	}{
		{"RFC3339", "2025-10-11T23:23:09Z", time.RFC3339},
		{"RFC3339Nano", "2025-10-11T23:23:09.337188Z", time.RFC3339Nano},
		{"FractionalNoZone", "2025-10-11T23:23:09.337188", "2006-01-02T15:04:05.999999"},
	}

	for _, tc := range cases {
		var parsed APITime
		payload := []byte("\"" + tc.value + "\"")
		if err := json.Unmarshal(payload, &parsed); err != nil {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
		if parsed.IsZero() {
			t.Fatalf("%s: parsed time is zero", tc.name)
		}
	}
}
