package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func TestInventoryServersPayload(t *testing.T) {
	var captured struct {
		body []byte
		path string
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured.path = req.URL.Path
		body, _ := io.ReadAll(req.Body)
		captured.body = body
		resp := InventoryResult{
			Data:       []map[string]any{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	_, err := client.InventoryServers(context.Background(), InventoryRequest{
		Pagination:    &InventoryPagination{Skip: 5, Limit: 10},
		HierarchyType: "HostsAndClusters",
		Sorting: map[string]any{
			"property":  "name",
			"direction": "descending",
		},
	})
	if err != nil {
		t.Fatalf("InventoryServers returned error: %v", err)
	}

	if captured.path != "/api/v1/inventory" {
		t.Fatalf("expected path /api/v1/inventory, got %s", captured.path)
	}

	var payload InventoryRequest
	if err := json.Unmarshal(captured.body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if payload.Pagination == nil || payload.Pagination.Skip != 5 || payload.Pagination.Limit != 10 {
		t.Fatalf("unexpected pagination %+v", payload.Pagination)
	}
	if payload.HierarchyType != "HostsAndClusters" {
		t.Fatalf("unexpected hierarchy %s", payload.HierarchyType)
	}
	if payload.Sorting["property"] != "name" || payload.Sorting["direction"] != "descending" {
		t.Fatalf("unexpected sorting %+v", payload.Sorting)
	}
}

func TestInventoryObjectsPath(t *testing.T) {
	var capturedPath string
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		capturedPath = req.URL.Path
		resp := InventoryResult{
			Data:       []map[string]any{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	_, err := client.InventoryObjects(context.Background(), "host.example.com", InventoryRequest{})
	if err != nil {
		t.Fatalf("InventoryObjects returned error: %v", err)
	}
	if capturedPath != "/api/v1/inventory/host.example.com" {
		t.Fatalf("unexpected path %s", capturedPath)
	}
}

func TestUnstructuredDataServersQuery(t *testing.T) {
	var captured struct {
		path  string
		query url.Values
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured.path = req.URL.Path
		captured.query = req.URL.Query()
		resp := UnstructuredDataServersResult{
			Data:       []map[string]any{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	asc := true
	_, err := client.UnstructuredDataServers(context.Background(), UnstructuredDataServersFilter{
		Skip:           10,
		Limit:          50,
		Name:           "*fs*",
		OrderColumn:    "Name",
		OrderAscending: &asc,
	})
	if err != nil {
		t.Fatalf("UnstructuredDataServers returned error: %v", err)
	}

	if captured.path != "/api/v1/inventory/unstructuredDataServers" {
		t.Fatalf("unexpected path %s", captured.path)
	}
	assertQueryEquals(t, captured.query, "skip", "10")
	assertQueryEquals(t, captured.query, "limit", "50")
	assertQueryEquals(t, captured.query, "nameFilter", "*fs*")
	assertQueryEquals(t, captured.query, "orderColumn", "Name")
	assertQueryEquals(t, captured.query, "orderAsc", "true")
}

func TestUnstructuredDataServerPath(t *testing.T) {
	var capturedPath string
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		capturedPath = req.URL.Path
		payload, _ := json.Marshal(map[string]any{})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	if _, err := client.UnstructuredDataServer(context.Background(), "server-id"); err != nil {
		t.Fatalf("UnstructuredDataServer returned error: %v", err)
	}

	if capturedPath != "/api/v1/inventory/unstructuredDataServers/server-id" {
		t.Fatalf("unexpected path %s", capturedPath)
	}
}
