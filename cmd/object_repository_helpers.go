package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadSpecFile(path string) (map[string]any, error) {
	var data []byte
	var err error
	if strings.TrimSpace(path) == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	payload := make(map[string]any)
	if err := json.Unmarshal(data, &payload); err == nil {
		return payload, nil
	}

	var yamlPayload map[string]any
	if err := yaml.Unmarshal(data, &yamlPayload); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}

	normalized := normalizeYAMLMap(yamlPayload)
	result, ok := normalized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode spec: expected mapping at root")
	}
	return result, nil
}

func normalizeYAMLMap(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = normalizeYAMLMap(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[fmt.Sprint(key)] = normalizeYAMLMap(val)
		}
		return out
	case []any:
		for i := range v {
			v[i] = normalizeYAMLMap(v[i])
		}
		return v
	default:
		return v
	}
}

func deepCopyMap(src map[string]any) (map[string]any, error) {
	if src == nil {
		return make(map[string]any), nil
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	dst := make(map[string]any)
	if err := json.Unmarshal(bytes, &dst); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return dst, nil
}

func applyOverridesToMap(payload map[string]any, overrides map[string]string) error {
	for path, value := range overrides {
		if err := setNestedField(payload, strings.Split(path, "."), parseScalar(value)); err != nil {
			return err
		}
	}
	return nil
}

func setNestedField(target map[string]any, path []string, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("override path cannot be empty")
	}
	current := target
	for i, key := range path {
		if i == len(path)-1 {
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
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("override path %s conflicts with existing value", strings.Join(path[:i+1], "."))
		}
		current = child
	}
	return nil
}

func parseScalar(value string) any {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return value
}

func deriveObjectTarget(repoType string, payload map[string]any) string {
	switch repoType {
	case "S3Compatible", "AmazonS3", "AmazonS3Glacier", "WasabiCloud", "GoogleCloud", "IBMCloud":
		if bucket := asMap(payload["bucket"]); bucket != nil {
			bucketName := asString(bucket["bucketName"])
			folder := asString(bucket["folderName"])
			if bucketName != "" {
				if folder != "" {
					return fmt.Sprintf("bucket %s/%s", bucketName, folder)
				}
				return fmt.Sprintf("bucket %s", bucketName)
			}
		}
	case "AzureBlob", "AzureArchive", "AzureDataBox":
		if container := asMap(payload["container"]); container != nil {
			containerName := asString(container["containerName"])
			folder := asString(container["folderName"])
			if containerName != "" {
				if folder != "" {
					return fmt.Sprintf("container %s/%s", containerName, folder)
				}
				return fmt.Sprintf("container %s", containerName)
			}
		}
	case "VeeamDataCloudVault":
		if container := asMap(payload["container"]); container != nil {
			folder := asString(container["folder"])
			if folder != "" {
				return fmt.Sprintf("vault folder %s", folder)
			}
		}
	}
	return ""
}

func nestedValue(payload map[string]any, path ...string) (any, bool) {
	if len(path) == 0 {
		return payload, true
	}

	current := any(payload)
	for _, key := range path {
		nextMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := nextMap[key]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func nestedString(payload map[string]any, path ...string) string {
	if value, ok := nestedValue(payload, path...); ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

func nestedBool(payload map[string]any, path ...string) (bool, bool) {
	if value, ok := nestedValue(payload, path...); ok {
		switch typed := value.(type) {
		case bool:
			return typed, true
		case *bool:
			if typed == nil {
				return false, false
			}
			return *typed, true
		}
	}
	return false, false
}

func nestedStringSlice(payload map[string]any, path ...string) []string {
	value, ok := nestedValue(payload, path...)
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, v := range typed {
			result = append(result, fmt.Sprint(v))
		}
		return result
	default:
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			result := make([]string, val.Len())
			for i := 0; i < val.Len(); i++ {
				result[i] = fmt.Sprint(val.Index(i).Interface())
			}
			return result
		}
	}
	return nil
}
