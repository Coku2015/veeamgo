package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestScaleOutRepositoriesFilterEncoding(t *testing.T) {
	var captured struct {
		path  string
		query url.Values
	}

	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":          "sobr-1",
				"name":        "Primary SOBR",
				"description": "Primary repository",
				"uniqueId":    "unique-1",
				"performanceTier": map[string]any{
					"performanceExtents": []map[string]any{
						{
							"id":     "extent-1",
							"name":   "Repo1",
							"status": []string{"Normal", "Maintenance"},
						},
					},
					"advancedSettings": map[string]any{
						"perVmBackup":           true,
						"fullWhenExtentOffline": false,
					},
				},
				"placementPolicy": map[string]any{
					"type":                         "DataLocality",
					"enforceStrictPlacementPolicy": true,
				},
				"capacityTier": map[string]any{
					"isEnabled":         true,
					"copyPolicyEnabled": true,
					"movePolicyEnabled": false,
					"extents": []map[string]any{
						{"id": "cap-1"},
					},
				},
				"archiveTier": map[string]any{
					"isEnabled":         true,
					"extentId":          "arch-1",
					"archivePeriodDays": 30,
				},
			},
		},
		"pagination": map[string]any{
			"total": 6,
			"count": 1,
			"skip":  5,
			"limit": 5,
		},
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured.path = req.URL.Path
		captured.query = req.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	orderAsc := true
	results, err := client.ScaleOutRepositories(context.Background(), ScaleOutRepositoriesFilter{
		Name:           "Prod*",
		OrderColumn:    "Name",
		OrderAscending: &orderAsc,
		Skip:           5,
		MaxItems:       5,
	})
	if err != nil {
		t.Fatalf("ScaleOutRepositories returned error: %v", err)
	}

	if captured.path != "/api/v1/backupInfrastructure/scaleOutRepositories" {
		t.Fatalf("unexpected path %s", captured.path)
	}
	assertQueryEquals(t, captured.query, "nameFilter", "Prod*")
	assertQueryEquals(t, captured.query, "orderColumn", "Name")
	assertQueryEquals(t, captured.query, "orderAsc", "true")
	assertQueryEquals(t, captured.query, "skip", "5")
	assertQueryEquals(t, captured.query, "limit", "5")

	if len(results) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(results))
	}
	repo := results[0]
	if repo.ID != "sobr-1" || repo.UniqueID != "unique-1" {
		t.Fatalf("unexpected repository payload: %+v", repo)
	}
	if repo.PlacementPolicy == nil || repo.PlacementPolicy.Type != "DataLocality" || !repo.PlacementPolicy.EnforceStrictPlacementPolicy {
		t.Fatalf("placement policy not decoded: %+v", repo.PlacementPolicy)
	}
	if repo.CapacityTier == nil || !repo.CapacityTier.IsEnabled || !repo.CapacityTier.CopyPolicyEnabled {
		t.Fatalf("capacity tier not decoded: %+v", repo.CapacityTier)
	}
	if repo.ArchiveTier == nil || repo.ArchiveTier.ArchivePeriodDays != 30 {
		t.Fatalf("archive tier not decoded: %+v", repo.ArchiveTier)
	}
	if repo.PerformanceTier.AdvancedSettings == nil || !repo.PerformanceTier.AdvancedSettings.PerVMBackup {
		t.Fatalf("advanced settings not decoded: %+v", repo.PerformanceTier.AdvancedSettings)
	}
	if len(repo.PerformanceTier.PerformanceExtents) != 1 {
		t.Fatalf("performance extents missing: %+v", repo.PerformanceTier.PerformanceExtents)
	}
	if statuses := repo.PerformanceTier.PerformanceExtents[0].Status; len(statuses) != 2 || statuses[0] != "Normal" {
		t.Fatalf("extent status missing: %+v", statuses)
	}
}

func TestScaleOutRepositoryByName(t *testing.T) {
	response := map[string]any{
		"data": []map[string]any{
			{"id": "sobr-1", "name": "Primary SOBR", "description": "", "uniqueId": "u-1", "performanceTier": map[string]any{"performanceExtents": []map[string]any{}}},
			{"id": "sobr-2", "name": "Archive SOBR", "description": "", "uniqueId": "u-2", "performanceTier": map[string]any{"performanceExtents": []map[string]any{}}},
		},
		"pagination": map[string]any{"total": 2, "count": 2, "skip": 0, "limit": 2},
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/backupInfrastructure/scaleOutRepositories" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	repo, err := client.ScaleOutRepositoryByName(context.Background(), "primary sobr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.ID != "sobr-1" {
		t.Fatalf("expected sobr-1, got %+v", repo)
	}

	_, err = client.ScaleOutRepositoryByName(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "did you mean") {
		t.Fatalf("expected suggestions error, got %v", err)
	}

	response["data"] = []map[string]any{
		{"id": "sobr-3", "name": "Duplicate", "performanceTier": map[string]any{"performanceExtents": []map[string]any{}}},
		{"id": "sobr-4", "name": "duplicate", "performanceTier": map[string]any{"performanceExtents": []map[string]any{}}},
	}
	payload, err = json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	repo = nil
	_, err = client.ScaleOutRepositoryByName(context.Background(), "Duplicate")
	if err == nil || !strings.Contains(err.Error(), "multiple scale-out repositories") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestScaleOutRepositoryRequiresID(t *testing.T) {
	client := &Client{}
	if _, err := client.ScaleOutRepository(context.Background(), ""); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := client.ScaleOutRepository(context.Background(), "   "); err == nil {
		t.Fatalf("expected validation error on whitespace")
	}
}
