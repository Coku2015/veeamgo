package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
