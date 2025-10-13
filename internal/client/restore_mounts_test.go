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

func TestInstantRecoveryMountsFilterEncodingVSphere(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/api/v1/restore/instantRecovery/vSphere/vm") {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := instantVMRecoveryMountsResponse{
			Data:       []instantVMRecoveryMount{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	_, err := client.InstantRecoveryMounts(context.Background(), InstantRecoveryMountFilter{
		Platforms: []InstantRecoveryPlatform{InstantRecoveryPlatformVSphere},
		State:     "Ready",
		Name:      "Prod*",
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("InstantRecoveryMounts returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "stateFilter", "Ready")
	assertQueryEquals(t, captured, "vmNameFilter", "Prod*")
	assertQueryEquals(t, captured, "limit", "5")
	assertQueryEquals(t, captured, "skip", "0")
}

func TestInstantRecoveryMountsAzureNameFilter(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/api/v1/restore/instantRecovery/azure/vm") {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		resp := azureInstantVMRecoveryMountsResponse{
			Data: []azureInstantVMRecoveryMount{
				{
					ID:    "1",
					State: "Running",
					Spec: azureInstantVMRecoverySpec{
						RestorePointID: "rp-1",
						Name:           azureComputeNameModel{Name: "ProdVM01"},
						Subscription:   azureComputeSubscriptionModel{SubscriptionID: "sub-1", Location: "eastus"},
						ResourceGroup:  azureComputeResourceGroupModel{ResourceGroup: "rg-prod"},
						Network:        azureComputeNetworkModel{Network: "vnet-prod"},
					},
				},
				{
					ID:    "2",
					State: "Running",
					Spec: azureInstantVMRecoverySpec{
						RestorePointID: "rp-2",
						Name:           azureComputeNameModel{Name: "DevVM01"},
						Subscription:   azureComputeSubscriptionModel{SubscriptionID: "sub-1", Location: "eastus"},
						ResourceGroup:  azureComputeResourceGroupModel{ResourceGroup: "rg-dev"},
						Network:        azureComputeNetworkModel{Network: "vnet-dev"},
					},
				},
			},
			Pagination: paginationResult{
				Total: 2,
				Count: 2,
				Skip:  0,
				Limit: 200,
			},
		}

		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	results, err := client.InstantRecoveryMounts(context.Background(), InstantRecoveryMountFilter{
		Platforms: []InstantRecoveryPlatform{InstantRecoveryPlatformAzure},
		Name:      "Prod*",
	})
	if err != nil {
		t.Fatalf("InstantRecoveryMounts returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(results))
	}
	if results[0].ID != "1" {
		t.Fatalf("expected mount ID 1, got %s", results[0].ID)
	}
}
