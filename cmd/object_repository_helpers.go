package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}
	return payload, nil
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
