package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/veeamgo/veeamgo/internal/session"
)

func TestSessionsFilterEncoding(t *testing.T) {
	captured := new(struct{ query url.Values })

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured.query = req.URL.Query()
		resp := sessionResponse{
			Data:       []Session{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	trueVal := true
	createdAfter := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	createdBefore := createdAfter.Add(4 * time.Hour)
	endedAfter := createdAfter.Add(5 * time.Hour)
	endedBefore := createdAfter.Add(10 * time.Hour)

	_, err := client.Sessions(context.Background(), SessionsFilter{
		Name:           "Backup*",
		Types:          []string{"Backup", "Replica"},
		States:         []string{"Working", "Stopped"},
		Results:        []string{"Success", "Warning"},
		JobID:          "job-123",
		CreatedAfter:   &createdAfter,
		CreatedBefore:  &createdBefore,
		EndedAfter:     &endedAfter,
		EndedBefore:    &endedBefore,
		OrderColumn:    "creationTime",
		OrderAscending: &trueVal,
		MaxItems:       10,
	})
	if err != nil {
		t.Fatalf("Sessions returned error: %v", err)
	}

	if captured.query == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured.query, "nameFilter", "Backup*")
	assertQueryEquals(t, captured.query, "jobIdFilter", "job-123")
	if got := captured.query["typeFilter"]; len(got) != 2 || got[0] != "Backup" || got[1] != "Replica" {
		t.Fatalf("unexpected typeFilter %v", got)
	}
	if got := captured.query["stateFilter"]; len(got) != 2 || got[0] != "Working" || got[1] != "Stopped" {
		t.Fatalf("unexpected stateFilter %v", got)
	}
	if got := captured.query["resultFilter"]; len(got) != 2 || got[0] != "Success" || got[1] != "Warning" {
		t.Fatalf("unexpected resultFilter %v", got)
	}
	assertQueryEquals(t, captured.query, "createdAfterFilter", createdAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "createdBeforeFilter", createdBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "endedAfterFilter", endedAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "endedBeforeFilter", endedBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "orderColumn", "creationTime")
	assertQueryEquals(t, captured.query, "orderAsc", "true")
	assertQueryEquals(t, captured.query, "limit", "10")
	assertQueryEquals(t, captured.query, "skip", "0")
}

func TestTaskSessionsFilterEncoding(t *testing.T) {
	captured := new(struct{ query url.Values })

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured.query = req.URL.Query()
		resp := taskSessionsResponse{
			Data:       []TaskSessionSummary{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	desc := false
	created := time.Date(2024, 9, 1, 10, 0, 0, 0, time.UTC)
	ended := created.Add(2 * time.Hour)

	_, err := client.TaskSessions(context.Background(), TaskSessionsFilter{
		Name:           "Task*",
		SessionID:      "session-1",
		Type:           "Backup",
		SessionType:    "Job",
		State:          "Working",
		Result:         "Success",
		ScanType:       "MalwareScan",
		ScanResult:     "Clean",
		ScanState:      "Finished",
		CreatedAfter:   &created,
		CreatedBefore:  &ended,
		OrderColumn:    "creationTime",
		OrderAscending: &desc,
		MaxItems:       5,
	})
	if err != nil {
		t.Fatalf("TaskSessions returned error: %v", err)
	}

	if captured.query == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured.query, "nameFilter", "Task*")
	assertQueryEquals(t, captured.query, "sessionIdFilter", "session-1")
	assertQueryEquals(t, captured.query, "typeFilter", "Backup")
	assertQueryEquals(t, captured.query, "sessionTypeFilter", "Job")
	assertQueryEquals(t, captured.query, "stateFilter", "Working")
	assertQueryEquals(t, captured.query, "resultFilter", "Success")
	assertQueryEquals(t, captured.query, "scanTypeFilter", "MalwareScan")
	assertQueryEquals(t, captured.query, "scanResultFilter", "Clean")
	assertQueryEquals(t, captured.query, "scanStateFilter", "Finished")
	assertQueryEquals(t, captured.query, "createdAfterFilter", created.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "createdBeforeFilter", ended.Format(time.RFC3339))
	assertQueryEquals(t, captured.query, "orderColumn", "creationTime")
	assertQueryEquals(t, captured.query, "orderAsc", "false")
	assertQueryEquals(t, captured.query, "limit", "5")
}

// --- helpers ----------------------------------------------------------------

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testClientWithResponder(t *testing.T, fn roundTripperFunc) *Client {
	t.Helper()

	base, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}

	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Transport: fn},
		session:    &session.Session{AccessToken: "token", TokenType: "Bearer"},
	}
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(r *bytes.Reader) nopCloser {
	return nopCloser{Reader: r}
}

func assertQueryEquals(t *testing.T, q url.Values, key, expected string) {
	t.Helper()
	if got := q.Get(key); got != expected {
		t.Fatalf("expected %s=%s, got %s", key, expected, got)
	}
}
