package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestPublishBackupContent(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/dataIntegration/publish" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		payload := Session{
			ID:           "session-publish-1",
			Name:         "Publish disks",
			SessionType:  "PublishBackupContentViaMount",
			State:        "InProgress",
			CreationTime: time.Date(2025, 10, 21, 12, 0, 0, 0, time.UTC),
		}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"restorePointId":            "58d8ce8c-2e04-47c7-8f11-a17875b1eed3",
		"type":                      "FUSELinuxMount",
		"targetServerName":          "linbase03",
		"targetServerCredentialsId": "27f962c5-86d6-4a7c-b450-5faa99b08d03",
		"diskNames":                 []string{"Disk 1", "Disk 2"},
	}

	session, err := client.PublishBackupContent(context.Background(), spec)
	if err != nil {
		t.Fatalf("PublishBackupContent returned error: %v", err)
	}
	if session == nil || session.ID != "session-publish-1" {
		t.Fatalf("unexpected session: %+v", session)
	}

	if captured == nil {
		t.Fatalf("expected request payload to be captured")
	}
	if captured["type"] != "FUSELinuxMount" {
		t.Fatalf("unexpected type %v", captured["type"])
	}
	if captured["targetServerName"] != "linbase03" {
		t.Fatalf("unexpected targetServerName %v", captured["targetServerName"])
	}
	if _, ok := captured["diskNames"].([]any); !ok {
		t.Fatalf("expected diskNames to be encoded as array, got %T", captured["diskNames"])
	}
}

func TestPublishBackupContentValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.PublishBackupContent(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

func TestUnpublishBackupContent(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/dataIntegration/mount-123/unpublish" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		payload := Session{ID: "session-unpublish-1", State: "InProgress"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	session, err := client.UnpublishBackupContent(context.Background(), "mount-123")
	if err != nil {
		t.Fatalf("UnpublishBackupContent returned error: %v", err)
	}
	if session == nil || session.ID != "session-unpublish-1" {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestUnpublishBackupContentValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.UnpublishBackupContent(context.Background(), " "); err == nil {
		t.Fatalf("expected error for empty mount id")
	}
}
