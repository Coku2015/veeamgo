package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestProtectionGroupFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/agents/protectionGroups" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := protectionGroupsResponse{
			Data:       []ProtectionGroup{},
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

	_, err := client.ProtectionGroups(context.Background(), ProtectionGroupFilter{
		Name:           "Prod*",
		Type:           "Manual",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       5,
	})
	if err != nil {
		t.Fatalf("ProtectionGroups returned error: %v", err)
	}
	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "Prod*")
	assertQueryEquals(t, captured, "typeFilter", "Manual")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "5")
}

func TestDiscoveredEntityFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/agents/protectionGroups/group/discoveredEntities" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := discoveredEntitiesResponse{
			Data:       []DiscoveredEntity{},
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

	_, err := client.ProtectionGroupDiscoveredEntities(context.Background(), "group", DiscoveredEntityFilter{
		Name:           "srv-*",
		IPAddress:      "10.*",
		State:          "Ready",
		AgentStatus:    "Installed",
		DriverStatus:   "UpToDate",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       8,
	})
	if err != nil {
		t.Fatalf("ProtectionGroupDiscoveredEntities returned error: %v", err)
	}
	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "srv-*")
	assertQueryEquals(t, captured, "ipAddressFilter", "10.*")
	assertQueryEquals(t, captured, "stateFilter", "Ready")
	assertQueryEquals(t, captured, "agentStatusFilter", "Installed")
	assertQueryEquals(t, captured, "driverStatusFilter", "UpToDate")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "8")
}

func TestProtectedComputersFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/agents/protectedComputers" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := protectedComputersResponse{
			Data:       []ProtectedComputer{},
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

	_, err := client.ProtectedComputers(context.Background(), ProtectedComputerFilter{
		Name:           "host-*",
		Type:           "Windows",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       6,
	})
	if err != nil {
		t.Fatalf("ProtectedComputers returned error: %v", err)
	}
	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "host-*")
	assertQueryEquals(t, captured, "typeFilter", "Windows")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "6")
}

func TestLinuxPackageFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/agents/packages/linux" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := linuxPackageResponse{
			Data:       []LinuxPackage{},
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

	_, err := client.LinuxAgentPackages(context.Background(), LinuxPackageFilter{
		Name:           "veeam*",
		Distribution:   "Ubuntu",
		Bitness:        "x64",
		OrderColumn:    "packageName",
		OrderAscending: &orderAsc,
		MaxItems:       3,
	})
	if err != nil {
		t.Fatalf("LinuxAgentPackages returned error: %v", err)
	}
	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "veeam*")
	assertQueryEquals(t, captured, "distributionFilter", "Ubuntu")
	assertQueryEquals(t, captured, "packageBitnessFilter", "x64")
	assertQueryEquals(t, captured, "orderColumn", "packageName")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "3")
}
