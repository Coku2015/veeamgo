package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/veeamgo/veeamgo/internal/session"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testClient(t *testing.T, fn roundTripFunc) *Client {
	t.Helper()

	base, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Transport: fn},
		session:    &session.Session{AccessToken: "token", TokenType: "Bearer"},
	}
}

func TestJobStatesFilterEncoding(t *testing.T) {
	var captured *http.Request

	responsePayload := jobStatesResponse{
		Data:       []JobState{},
		Pagination: paginationResult{Total: 0, Count: 0, Skip: 0, Limit: 0},
	}
	respBytes, err := json.Marshal(responsePayload)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	client := testClient(t, func(req *http.Request) (*http.Response, error) {
		captured = req
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(respBytes)),
		}, nil
	})

	trueVal := true
	from := time.Date(2024, 10, 1, 8, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)

	_, err = client.JobStates(context.Background(), JobStatesFilter{
		Name:          "Job*",
		Types:         []string{"VSphereBackup", "HyperVBackup"},
		Status:        "Running",
		LastResult:    "Success",
		Workload:      "Vmware",
		RepositoryID:  "repo-id",
		HighPriority:  &trueVal,
		LastRunAfter:  &from,
		LastRunBefore: &to,
		AfterJobID:    "after-id",
		AfterJobName:  "Daily Backup",
		OrderBy:       "name",
		OrderAsc:      true,
		MaxItems:      1,
	})
	if err != nil {
		t.Fatalf("JobStates returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("request was not captured")
	}

	if captured.URL.Path != "/api/v1/jobs/states" {
		t.Fatalf("unexpected path %s", captured.URL.Path)
	}

	assertEqual := func(key, expected string) {
		if got := captured.URL.Query().Get(key); got != expected {
			t.Fatalf("expected %s=%s, got %s", key, expected, got)
		}
	}

	assertEqual("nameFilter", "Job*")
	assertEqual("statusFilter", "Running")
	assertEqual("lastResultFilter", "Success")
	assertEqual("workloadFilter", "Vmware")
	assertEqual("repositoryIdFilter", "repo-id")
	assertEqual("isHighPriorityJobFilter", "true")
	assertEqual("afterJobIdFilter", "after-id")
	assertEqual("afterJobNameFilter", "Daily Backup")
	assertEqual("orderColumn", "name")
	assertEqual("orderAsc", "true")
	assertEqual("limit", "1")
	assertEqual("skip", "0")

	if got := captured.URL.Query().Get("lastRunAfterFilter"); got != from.Format(time.RFC3339) {
		t.Fatalf("expected lastRunAfterFilter=%s, got %s", from.Format(time.RFC3339), got)
	}
	if got := captured.URL.Query().Get("lastRunBeforeFilter"); got != to.Format(time.RFC3339) {
		t.Fatalf("expected lastRunBeforeFilter=%s, got %s", to.Format(time.RFC3339), got)
	}

	types := captured.URL.Query()["typeFilter"]
	if len(types) != 2 || types[0] != "VSphereBackup" || types[1] != "HyperVBackup" {
		t.Fatalf("unexpected typeFilter values: %v", types)
	}
}

func TestJobStateByNameErrors(t *testing.T) {
	payload := jobStatesResponse{
		Data: []JobState{
			{ID: "1", Name: "Alpha"},
			{ID: "2", Name: "Bravo"},
		},
		Pagination: paginationResult{Total: 2, Count: 2, Skip: 0, Limit: 2},
	}
	respBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	client := testClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(respBytes)),
		}, nil
	})

	_, err = client.JobStateByName(context.Background(), "Missing")
	if err == nil || !strings.Contains(err.Error(), "did you mean") {
		t.Fatalf("expected suggestion error, got %v", err)
	}
}
