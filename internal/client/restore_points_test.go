package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRestorePointsFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		resp := restorePointsResponse{
			Data:       []json.RawMessage{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	orderAsc := true
	createdAfter := time.Date(2025, 3, 1, 8, 0, 0, 0, time.UTC)
	createdBefore := createdAfter.Add(24 * time.Hour)

	_, err := client.RestorePoints(context.Background(), RestorePointsFilter{
		Name:           "vm*",
		BackupID:       "backup-id",
		BackupObjectID: "object-id",
		PlatformName:   "VMware",
		PlatformID:     "platform-id",
		MalwareStatus:  "Suspicious",
		CreatedAfter:   &createdAfter,
		CreatedBefore:  &createdBefore,
		OrderColumn:    "creationTime",
		OrderAscending: &orderAsc,
		MaxItems:       7,
	})
	if err != nil {
		t.Fatalf("RestorePoints returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "vm*")
	assertQueryEquals(t, captured, "backupIdFilter", "backup-id")
	assertQueryEquals(t, captured, "backupObjectIdFilter", "object-id")
	assertQueryEquals(t, captured, "platformNameFilter", "VMware")
	assertQueryEquals(t, captured, "platformIdFilter", "platform-id")
	assertQueryEquals(t, captured, "malwareStatusFilter", "Suspicious")
	assertQueryEquals(t, captured, "createdAfterFilter", createdAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "createdBeforeFilter", createdBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "creationTime")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "7")
}

func TestRestorePointByNameSuggestions(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		var raw []json.RawMessage
		for _, rp := range []RestorePoint{{ID: "1", Name: "VM1"}, {ID: "2", Name: "VM2"}} {
			payload, _ := json.Marshal(rp)
			raw = append(raw, payload)
		}
		resp := restorePointsResponse{
			Data:       raw,
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	_, err := client.RestorePointByName(context.Background(), "Missing", RestorePointsFilter{})
	if err == nil || !strings.Contains(err.Error(), "did you mean") {
		t.Fatalf("expected suggestion error, got %v", err)
	}
}
