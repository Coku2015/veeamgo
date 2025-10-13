package client

import (
	"encoding/json"
	"fmt"
)

// decodeRawMessage converts a JSON payload into both a generic map and a typed value.
// The returned map preserves the original fields so commands can re-emit untouched JSON.
func decodeRawMessage[T any](data json.RawMessage, target *T) (map[string]any, error) {
	raw := make(map[string]any)
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode raw payload: %w", err)
	}
	if target != nil {
		if err := json.Unmarshal(data, target); err != nil {
			return nil, fmt.Errorf("decode typed payload: %w", err)
		}
	}
	return raw, nil
}

// decodeObject copies a JSON object into both a generic map and a typed value.
func decodeObject[T any](payload map[string]any, target *T) (map[string]any, error) {
	if target == nil {
		return payload, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}
