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

func TestBackupsFilterEncoding(t *testing.T) {
	var capturedQuery url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		capturedQuery = req.URL.Query()
		resp := backupsResponse{
			Data:       []Backup{},
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
	after := time.Date(2025, 1, 10, 11, 0, 0, 0, time.UTC)
	before := after.Add(4 * time.Hour)

	_, err := client.Backups(context.Background(), BackupsFilter{
		Name:           "Daily*",
		JobID:          "job-1",
		JobType:        "VSphereBackup",
		PolicyTag:      "Gold",
		PlatformID:     "00000000-0000-0000-0000-000000000000",
		CreatedAfter:   &after,
		CreatedBefore:  &before,
		OrderColumn:    "creationTime",
		OrderAscending: &orderAsc,
		MaxItems:       5,
	})
	if err != nil {
		t.Fatalf("Backups returned error: %v", err)
	}

	if capturedQuery == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, capturedQuery, "nameFilter", "Daily*")
	assertQueryEquals(t, capturedQuery, "jobIdFilter", "job-1")
	assertQueryEquals(t, capturedQuery, "jobTypeFilter", "VSphereBackup")
	assertQueryEquals(t, capturedQuery, "policyTagFilter", "Gold")
	assertQueryEquals(t, capturedQuery, "platformIdFilter", "00000000-0000-0000-0000-000000000000")
	assertQueryEquals(t, capturedQuery, "createdAfterFilter", after.Format(time.RFC3339))
	assertQueryEquals(t, capturedQuery, "createdBeforeFilter", before.Format(time.RFC3339))
	assertQueryEquals(t, capturedQuery, "orderColumn", "creationTime")
	assertQueryEquals(t, capturedQuery, "orderAsc", "true")
	assertQueryEquals(t, capturedQuery, "limit", "5")
}

func TestBackupFilesFilterEncoding(t *testing.T) {
	var capturedQuery url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		capturedQuery = req.URL.Query()
		resp := backupFilesResponse{
			Data:       []BackupFile{},
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
	after := time.Date(2025, 2, 1, 9, 0, 0, 0, time.UTC)

	_, err := client.BackupFiles(context.Background(), "backup-id", BackupFilesFilter{
		Name:         "VBK*",
		CreatedAfter: &after,
		GFSPeroid:    "Weekly",
		OrderColumn:  "creationTime",
		OrderAsc:     &orderAsc,
		MaxItems:     3,
	})
	if err != nil {
		t.Fatalf("BackupFiles returned error: %v", err)
	}

	if capturedQuery == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, capturedQuery, "nameFilter", "VBK*")
	assertQueryEquals(t, capturedQuery, "createdAfterFilter", after.Format(time.RFC3339))
	assertQueryEquals(t, capturedQuery, "gfsPeriodFilter", "Weekly")
	assertQueryEquals(t, capturedQuery, "orderColumn", "creationTime")
	assertQueryEquals(t, capturedQuery, "orderAsc", "true")
	assertQueryEquals(t, capturedQuery, "limit", "3")
}
