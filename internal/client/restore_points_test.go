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

func TestRestorePointsDeduplicatesEntries(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		now := time.Date(2025, 10, 14, 22, 1, 53, 0, time.UTC)
		points := []RestorePoint{
			{ID: "restore-1", Name: "ubuntu", Type: "Full", CreationTime: now},
			{ID: "restore-1", Name: "ubuntu-copy", Type: "Full", CreationTime: now.Add(time.Minute)},
			{SessionID: "session-1", Name: "vm-no-id", Type: "Incremental", CreationTime: now.Add(2 * time.Minute)},
			{SessionID: "session-1", Name: "vm-no-id-duplicate", Type: "Incremental", CreationTime: now.Add(3 * time.Minute)},
			{Name: "vm-fallback", Type: "Full", CreationTime: now.Add(4 * time.Minute)},
			{Name: "vm-fallback", Type: "Full", CreationTime: now.Add(4 * time.Minute)},
			{ID: "restore-unique", Name: "unique", Type: "Full", CreationTime: now.Add(5 * time.Minute)},
		}

		var raw []json.RawMessage
		for _, rp := range points {
			payload, _ := json.Marshal(rp)
			raw = append(raw, payload)
		}

		resp := restorePointsResponse{
			Data:       raw,
			Pagination: paginationResult{Total: len(raw), Count: len(raw)},
		}

		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	result, err := client.RestorePoints(context.Background(), RestorePointsFilter{})
	if err != nil {
		t.Fatalf("RestorePoints returned error: %v", err)
	}

	if got, want := len(result.RestorePoints), 4; got != want {
		t.Fatalf("expected %d restore points after deduplication, got %d", want, got)
	}

	expectedOrder := []string{"restore-1", "session-1", "", "restore-unique"}
	for idx, expected := range expectedOrder {
		if idx >= len(result.RestorePoints) {
			break
		}
		rp := result.RestorePoints[idx]
		switch expected {
		case "":
			if strings.TrimSpace(rp.ID) != "" || strings.TrimSpace(rp.SessionID) != "" {
				t.Fatalf("expected fallback keyed restore point at index %d, got id=%q session=%q", idx, rp.ID, rp.SessionID)
			}
			if rp.Name != "vm-fallback" {
				t.Fatalf("unexpected fallback restore point name at index %d: %s", idx, rp.Name)
			}
		case "session-1":
			if rp.SessionID != expected {
				t.Fatalf("expected session %q at index %d, got %q", expected, idx, rp.SessionID)
			}
		default:
			if rp.ID != expected {
				t.Fatalf("expected id %q at index %d, got %q", expected, idx, rp.ID)
			}
		}
	}
}
