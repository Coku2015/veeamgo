package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestAuthorizationEventsFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id":             "event-1",
				"name":           "Pending approval",
				"description":    "Role change",
				"state":          "Pending",
				"createdBy":      "svc-security",
				"creationTime":   "2025-01-01T10:00:00Z",
				"expirationTime": "2025-01-02T10:00:00Z",
			},
		},
		"pagination": map[string]any{
			"total": 1,
			"count": 1,
			"skip":  0,
			"limit": 1,
		},
	}

	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	falseVal := false
	createdAfter := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	createdBefore := createdAfter.Add(6 * time.Hour)
	processedAfter := createdAfter.Add(2 * time.Hour)
	processedBefore := createdAfter.Add(4 * time.Hour)
	expireAfter := createdAfter.Add(24 * time.Hour)
	expireBefore := createdAfter.Add(48 * time.Hour)

	result, err := client.AuthorizationEvents(context.Background(), AuthorizationEventsFilter{
		Name:            "Role*",
		State:           "Pending",
		CreatedBy:       "svc-security",
		ProcessedBy:     "auditor",
		CreatedAfter:    &createdAfter,
		CreatedBefore:   &createdBefore,
		ProcessedAfter:  &processedAfter,
		ProcessedBefore: &processedBefore,
		ExpireAfter:     &expireAfter,
		ExpireBefore:    &expireBefore,
		OrderColumn:     "creationTime",
		OrderAscending:  &falseVal,
		MaxItems:        5,
	})
	if err != nil {
		t.Fatalf("AuthorizationEvents returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "Role*")
	assertQueryEquals(t, captured, "stateFilter", "Pending")
	assertQueryEquals(t, captured, "createdByFilter", "svc-security")
	assertQueryEquals(t, captured, "processedByFilter", "auditor")
	assertQueryEquals(t, captured, "createdAfterFilter", createdAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "createdBeforeFilter", createdBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "processedAfterFilter", processedAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "processedBeforeFilter", processedBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "expireAfterFilter", expireAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "expireBeforeFilter", expireBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "creationTime")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "5")
	assertQueryEquals(t, captured, "skip", "0")

	if result == nil || len(result.Events) != 1 {
		t.Fatalf("expected one authorization event, got %+v", result)
	}
	event := result.Events[0]
	if event.ID != "event-1" || event.Raw["id"] != "event-1" {
		t.Fatalf("unexpected event data: %+v", event)
	}
	if _, ok := result.Raw["data"]; !ok {
		t.Fatalf("raw result missing data field")
	}
}

func TestAuthorizationEventDetailRaw(t *testing.T) {
	payload := map[string]any{
		"id":            "event-42",
		"name":          "Scoped approval",
		"description":   "Approve scope",
		"state":         "Approved",
		"createdBy":     "svc",
		"creationTime":  "2025-01-05T12:00:00Z",
		"processedBy":   "auditor",
		"processedTime": "2025-01-05T12:05:00Z",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	detail, err := client.AuthorizationEventDetail(context.Background(), "event-42")
	if err != nil {
		t.Fatalf("AuthorizationEventDetail returned error: %v", err)
	}
	if detail.Event.ID != "event-42" {
		t.Fatalf("unexpected event id: %s", detail.Event.ID)
	}
	if detail.Raw["processedBy"] != "auditor" {
		t.Fatalf("expected raw field processedBy=auditor, got %v", detail.Raw["processedBy"])
	}
}

func TestSuspiciousActivityEventsFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id":               "mal-1",
				"type":             "YaraScan",
				"state":            "Created",
				"source":           "InternalVeeamDetector",
				"severity":         "Suspicious",
				"details":          "Rule hit",
				"engine":           "Yara",
				"createdBy":        "sensor",
				"creationTimeUtc":  "2025-01-02T08:00:00Z",
				"detectionTimeUtc": "2025-01-02T08:05:00Z",
				"machine": map[string]any{
					"displayName":    "srv01",
					"uuid":           "uuid-1",
					"backupObjectId": "obj-1",
				},
			},
		},
		"pagination": map[string]any{
			"total": 1,
			"count": 1,
			"skip":  0,
			"limit": 1,
		},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	orderAsc := true
	detectedAfter := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	detectedBefore := detectedAfter.Add(12 * time.Hour)

	result, err := client.SuspiciousActivityEvents(context.Background(), SuspiciousActivityFilter{
		Type:           "YaraScan",
		State:          "Created",
		Source:         "InternalVeeamDetector",
		Severity:       "Suspicious",
		CreatedBy:      "sensor",
		Engine:         "Yara",
		MachineName:    "srv01",
		BackupObjectID: "obj-1",
		DetectedAfter:  &detectedAfter,
		DetectedBefore: &detectedBefore,
		OrderColumn:    "detectionTimeUtc",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("SuspiciousActivityEvents returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "typeFilter", "YaraScan")
	assertQueryEquals(t, captured, "stateFilter", "Created")
	assertQueryEquals(t, captured, "sourceFilter", "InternalVeeamDetector")
	assertQueryEquals(t, captured, "severityFilter", "Suspicious")
	assertQueryEquals(t, captured, "createdByFilter", "sensor")
	assertQueryEquals(t, captured, "engineFilter", "Yara")
	assertQueryEquals(t, captured, "machineNameFilter", "srv01")
	assertQueryEquals(t, captured, "backupObjectIdFilter", "obj-1")
	assertQueryEquals(t, captured, "detectedAfterTimeUtcFilter", detectedAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "detectedBeforeTimeUtcFilter", detectedBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "detectionTimeUtc")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Events) != 1 {
		t.Fatalf("expected single event in result: %+v", result)
	}
	if result.Events[0].Raw["id"] != "mal-1" {
		t.Fatalf("raw data not preserved")
	}
}

func TestSecurityAnalyzerBestPracticesRaw(t *testing.T) {
	respPayload := map[string]any{
		"items": []map[string]any{
			{
				"id":           "bp-1",
				"bestPractice": "Enable MFA",
				"status":       "Warning",
				"note":         "Pending",
			},
		},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	result, err := client.SecurityAnalyzerBestPractices(context.Background())
	if err != nil {
		t.Fatalf("SecurityAnalyzerBestPractices returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one best practice, got %d", len(result.Items))
	}
	if result.Items[0].Raw["note"] != "Pending" {
		t.Fatalf("expected raw note field preserved")
	}
	if items, ok := result.Raw["items"].([]map[string]any); !ok || len(items) != 1 {
		t.Fatalf("raw result missing items slice")
	}
}

func TestYaraRulesRaw(t *testing.T) {
	respPayload := map[string]any{
		"data": []map[string]any{
			{"fileName": "ransomware.yar"},
		},
		"pagination": map[string]any{
			"total": 1,
			"count": 1,
			"skip":  0,
			"limit": 100,
		},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	result, err := client.YaraRules(context.Background())
	if err != nil {
		t.Fatalf("YaraRules returned error: %v", err)
	}
	if len(result.Rules) != 1 || result.Rules[0].FileName != "ransomware.yar" {
		t.Fatalf("unexpected rules result: %+v", result)
	}
	if data, ok := result.Raw["data"].([]map[string]any); !ok || len(data) != 1 {
		t.Fatalf("raw result missing data slice")
	}
}
