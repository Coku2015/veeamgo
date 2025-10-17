package jobconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Mode represents the operation being performed with a blueprint.
type Mode string

const (
	// ModeCreate signals validation should ensure create-safe fields.
	ModeCreate Mode = "create"
	// ModeUpdate signals validation should ensure update-safe fields.
	ModeUpdate Mode = "update"
)

// Blueprint mirrors a job specification that can be sent to VBR REST API.
type Blueprint struct {
	Spec map[string]any
}

// LoadFile reads a blueprint from the provided path (YAML or JSON).
func LoadFile(path string) (*Blueprint, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read blueprint: %w", err)
	}
	return Parse(data)
}

// Parse converts YAML or JSON payloads into a blueprint.
func Parse(data []byte) (*Blueprint, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, errors.New("blueprint payload is empty")
	}

	var parsed map[string]any
	switch firstChar(trimmed) {
	case '{', '[':
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("parse blueprint json: %w", err)
		}
	default:
		var generic any
		if err := yaml.Unmarshal(data, &generic); err != nil {
			return nil, fmt.Errorf("parse blueprint yaml: %w", err)
		}
		converted, err := normalizeValue(generic)
		if err != nil {
			return nil, err
		}
		var ok bool
		if parsed, ok = converted.(map[string]any); !ok {
			return nil, fmt.Errorf("blueprint root must be an object")
		}
	}

	return &Blueprint{Spec: parsed}, nil
}

// Clone performs a deep copy of the underlying spec.
func (b *Blueprint) Clone() *Blueprint {
	if b == nil {
		return nil
	}
	return &Blueprint{Spec: deepCopyMap(b.Spec)}
}

// ApplyOverrides merges CLI overrides (dot-separated paths) into the spec.
func (b *Blueprint) ApplyOverrides(overrides map[string]string) error {
	if b == nil {
		return errors.New("blueprint is nil")
	}
	for rawPath, rawValue := range overrides {
		path := sanitizePath(rawPath)
		if len(path) == 0 {
			return fmt.Errorf("override %q is invalid", rawPath)
		}
		value, err := parseOverrideValue(rawValue)
		if err != nil {
			return fmt.Errorf("parse override %q: %w", rawPath, err)
		}
		if err := setPathValue(b.Spec, path, value); err != nil {
			return fmt.Errorf("apply override %q: %w", rawPath, err)
		}
	}
	return nil
}

// Validate performs lightweight checks ensuring mandatory fields exist.
func (b *Blueprint) Validate(mode Mode) error {
	if b == nil {
		return errors.New("blueprint is nil")
	}

	name := findString(b.Spec, "name")
	if strings.TrimSpace(name) == "" {
		return errors.New("job name (spec.name) must be provided")
	}

	jobType := findString(b.Spec, "type")
	if strings.TrimSpace(jobType) == "" {
		return errors.New("job type (spec.type) must be provided")
	}

	if mode == ModeUpdate {
		id := findString(b.Spec, "id")
		if strings.TrimSpace(id) == "" {
			return errors.New("job id (spec.id) must be present when editing")
		}
	}

	return nil
}

// CreatePayload returns a deep copy suitable for job creation.
func (b *Blueprint) CreatePayload() map[string]any {
	return deepCopyMap(b.Spec)
}

// UpdatePayload merges the blueprint on top of the live model, preserving identifiers.
func (b *Blueprint) UpdatePayload(existing map[string]any) (map[string]any, error) {
	if b == nil {
		return nil, errors.New("blueprint is nil")
	}
	if existing == nil {
		return nil, errors.New("existing job model is nil")
	}

	merged := deepCopyMap(existing)
	cleanSpec := deepCopyMap(b.Spec)

	// Keep existing ID to avoid accidental modifications.
	if existingID := findString(existing, "id"); existingID != "" {
		if specID := findString(cleanSpec, "id"); specID == "" {
			setPathValue(cleanSpec, []string{"id"}, existingID) //nolint:errcheck // id path is simple
		}
	}

	if err := mergeMaps(merged, cleanSpec); err != nil {
		return nil, err
	}

	pruneReadOnlyFields(merged)
	return merged, nil
}

// FromExisting normalizes the server model into a reusable blueprint.
func FromExisting(model map[string]any) *Blueprint {
	if model == nil {
		return &Blueprint{Spec: map[string]any{}}
	}
	copy := deepCopyMap(model)
	pruneReadOnlyFields(copy)
	return &Blueprint{Spec: copy}
}

func firstChar(s string) rune {
	for _, r := range s {
		if !isWhitespace(r) {
			return r
		}
	}
	return 0
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
}

func normalizeValue(v any) (any, error) {
	switch typed := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			normalized, err := normalizeValue(value)
			if err != nil {
				return nil, err
			}
			result[key] = normalized
		}
		return result, nil
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("blueprint contains non-string key: %v", key)
			}
			normalized, err := normalizeValue(value)
			if err != nil {
				return nil, err
			}
			result[keyStr] = normalized
		}
		return result, nil
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized, err := normalizeValue(item)
			if err != nil {
				return nil, err
			}
			items = append(items, normalized)
		}
		return items, nil
	default:
		return typed, nil
	}
}

func deepCopyMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	dest := make(map[string]any, len(source))
	for key, value := range source {
		dest[key] = deepCopyValue(value)
	}
	return dest
}

func deepCopyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case []any:
		copied := make([]any, len(typed))
		for idx, item := range typed {
			copied[idx] = deepCopyValue(item)
		}
		return copied
	default:
		return typed
	}
}

func sanitizePath(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	segments := strings.Split(raw, ".")
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		out = append(out, segment)
	}
	return out
}

func setPathValue(target map[string]any, path []string, value any) error {
	if len(path) == 0 {
		return errors.New("path cannot be empty")
	}
	current := target
	for idx, rawKey := range path {
		key := matchKey(current, rawKey)
		isLast := idx == len(path)-1

		if isLast {
			current[key] = value
			return nil
		}

		next, ok := current[key]
		if !ok {
			child := make(map[string]any)
			current[key] = child
			current = child
			continue
		}

		asMap, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %s conflicts with non-object value", strings.Join(path[:idx+1], "."))
		}
		current = asMap
	}
	return nil
}

func matchKey(m map[string]any, candidate string) string {
	if m == nil {
		return candidate
	}
	if _, ok := m[candidate]; ok {
		return candidate
	}
	for existing := range m {
		if strings.EqualFold(existing, candidate) {
			return existing
		}
	}
	return candidate
}

func parseOverrideValue(input string) (any, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}
	switch strings.ToLower(trimmed) {
	case "null":
		return nil, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "\"") {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded, nil
		}
	}

	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f, nil
	}

	return input, nil
}

func findString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	for existing, value := range m {
		if !strings.EqualFold(existing, key) {
			continue
		}
		if asString, ok := value.(string); ok {
			return asString
		}
	}
	return ""
}

func mergeMaps(dest, src map[string]any) error {
	if dest == nil || src == nil {
		return nil
	}
	for key, value := range src {
		if value == nil {
			dest[key] = nil
			continue
		}

		existing, ok := dest[key]
		if !ok {
			dest[key] = deepCopyValue(value)
			continue
		}

		srcMap, srcIsMap := value.(map[string]any)
		destMap, destIsMap := existing.(map[string]any)
		if srcIsMap && destIsMap {
			if err := mergeMaps(destMap, srcMap); err != nil {
				return err
			}
			continue
		}

		dest[key] = deepCopyValue(value)
	}
	return nil
}

func pruneReadOnlyFields(m map[string]any) {
	if m == nil {
		return
	}
	for _, key := range []string{
		"links",
		"lastResult",
		"lastRun",
		"nextRun",
		"nextRunPolicy",
		"progress",
		"progressPercent",
		"statistics",
		"status",
		"warnings",
	} {
		deleteCaseInsensitive(m, key)
	}

	for _, value := range m {
		switch typed := value.(type) {
		case map[string]any:
			pruneReadOnlyFields(typed)
		case []any:
			for _, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					pruneReadOnlyFields(nested)
				}
			}
		}
	}
}

func deleteCaseInsensitive(m map[string]any, target string) {
	if m == nil {
		return
	}
	for key := range m {
		if strings.EqualFold(key, target) {
			delete(m, key)
			return
		}
	}
}
