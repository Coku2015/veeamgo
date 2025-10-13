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

func TestReplicasFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		resp := replicasResponse{
			Data:       []Replica{},
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
	createdAfter := time.Date(2025, 4, 2, 10, 0, 0, 0, time.UTC)
	createdBefore := createdAfter.Add(2 * time.Hour)

	_, err := client.Replicas(context.Background(), ReplicasFilter{
		Name:           "Replica*",
		JobID:          "job-id",
		PolicyTag:      "Bronze",
		PlatformID:     "platform-id",
		State:          "Ready",
		CreatedAfter:   &createdAfter,
		CreatedBefore:  &createdBefore,
		OrderColumn:    "creationTime",
		OrderAscending: &orderAsc,
		MaxItems:       5,
	})
	if err != nil {
		t.Fatalf("Replicas returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "Replica*")
	assertQueryEquals(t, captured, "jobIdFilter", "job-id")
	assertQueryEquals(t, captured, "policyTagFilter", "Bronze")
	assertQueryEquals(t, captured, "platformIdFilter", "platform-id")
	assertQueryEquals(t, captured, "stateFilter", "Ready")
	assertQueryEquals(t, captured, "createdAfterFilter", createdAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "createdBeforeFilter", createdBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "creationTime")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "5")
}

func TestReplicaPointsFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		resp := replicaPointsResponse{
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

	orderAsc := false
	createdAfter := time.Date(2025, 5, 6, 9, 0, 0, 0, time.UTC)
	createdBefore := createdAfter.Add(6 * time.Hour)

	_, err := client.ReplicaPoints(context.Background(), ReplicaPointsFilter{
		Name:           "RestorePoint*",
		ReplicaID:      "replica-id",
		PlatformName:   "VMware",
		PlatformID:     "platform-id",
		MalwareStatus:  "Suspicious",
		CreatedAfter:   &createdAfter,
		CreatedBefore:  &createdBefore,
		OrderColumn:    "creationTime",
		OrderAscending: &orderAsc,
		MaxItems:       3,
	})
	if err != nil {
		t.Fatalf("ReplicaPoints returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "RestorePoint*")
	assertQueryEquals(t, captured, "replicaIdFilter", "replica-id")
	assertQueryEquals(t, captured, "platformNameFilter", "VMware")
	assertQueryEquals(t, captured, "platformIdFilter", "platform-id")
	assertQueryEquals(t, captured, "malwareStatusFilter", "Suspicious")
	assertQueryEquals(t, captured, "createdAfterFilter", createdAfter.Format(time.RFC3339))
	assertQueryEquals(t, captured, "createdBeforeFilter", createdBefore.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "creationTime")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "3")
}
