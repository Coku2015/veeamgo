package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestProxiesFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id":          "proxy-1",
				"name":        "Proxy 1",
				"description": "Main proxy",
				"type":        "ViProxy",
				"server": map[string]any{
					"hostId":                "host-1",
					"hostName":              "hyperv-01",
					"maxTaskCount":          4,
					"transportMode":         "Direct",
					"failoverToNetwork":     true,
					"hostToProxyEncryption": true,
					"connectedDatastores": map[string]any{
						"autoSelectEnabled": false,
						"datastores": []map[string]any{
							{
								"datastore": map[string]any{
									"name": "Datastore1",
								},
							},
						},
					},
				},
			},
		},
		"pagination": map[string]any{"total": 1, "count": 1, "skip": 0, "limit": 1},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	orderAsc := true
	result, err := client.Proxies(context.Background(), ProxyFilter{
		Name:           "Proxy*",
		Type:           "ViProxy",
		HostID:         "host-1",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("Proxies returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "Proxy*")
	assertQueryEquals(t, captured, "typeFilter", "ViProxy")
	assertQueryEquals(t, captured, "hostIdFilter", "host-1")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Proxies) != 1 {
		t.Fatalf("expected proxy result: %+v", result)
	}

	proxy := result.Proxies[0]
	if proxy.HostID != "host-1" || proxy.TransportMode != "Direct" || !proxy.FailoverToNetwork {
		t.Fatalf("proxy fields not decoded: %+v", proxy)
	}
	if len(proxy.ConnectedDatastores) != 1 || proxy.ConnectedDatastores[0] != "Datastore1" {
		t.Fatalf("datastores not captured: %+v", proxy.ConnectedDatastores)
	}
}

func TestProxyStatesFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id":          "proxy-1",
				"name":        "Proxy 1",
				"description": "Main proxy",
				"type":        "ViProxy",
				"hostId":      "host-1",
				"hostName":    "hyperv-01",
				"isDisabled":  false,
				"isOnline":    true,
				"isOutOfDate": false,
			},
		},
		"pagination": map[string]any{"total": 1, "count": 1, "skip": 0, "limit": 1},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		captured = req.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	orderAsc := false
	result, err := client.ProxyStates(context.Background(), ProxyFilter{
		Name:           "Proxy*",
		Type:           "ViProxy",
		HostID:         "host-1",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("ProxyStates returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "Proxy*")
	assertQueryEquals(t, captured, "typeFilter", "ViProxy")
	assertQueryEquals(t, captured, "hostIdFilter", "host-1")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.States) != 1 {
		t.Fatalf("expected states result: %+v", result)
	}
	if result.States[0].Raw["id"] != "proxy-1" {
		t.Fatalf("raw state missing id")
	}
}

func TestManagedServerVolumesDecode(t *testing.T) {
	respPayload := map[string]any{
		"changedBlockTracking":  true,
		"failoverToVSSProvider": true,
		"volumes": []map[string]any{
			{
				"volume": map[string]any{
					"name":     "Volume 1",
					"platform": "HyperV",
				},
				"volumeSettings": map[string]any{
					"VSSprovider":  "Default",
					"maxSnapshots": 4,
				},
			},
		},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	result, err := client.ManagedServerVolumes(context.Background(), "host-1")
	if err != nil {
		t.Fatalf("ManagedServerVolumes returned error: %v", err)
	}

	if !result.ChangedBlockTracking || len(result.Volumes) != 1 {
		t.Fatalf("unexpected volumes result: %+v", result)
	}
	if result.Volumes[0].VSSProvider != "Default" || result.Volumes[0].MaxSnapshots != 4 {
		t.Fatalf("volume settings missing: %+v", result.Volumes[0])
	}
	if result.Volumes[0].InventoryObject["name"] != "Volume 1" {
		t.Fatalf("inventory object missing: %+v", result.Volumes[0].InventoryObject)
	}
}

func TestOptionalComponentDefaultsDecode(t *testing.T) {
	respPayload := map[string]any{
		"optionalComponents": []map[string]any{
			{
				"displayName":       "Veeam Installer Service",
				"optionalComponent": "VeeamInstallerSvc",
			},
		},
	}
	body, err := json.Marshal(respPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	result, err := client.ManagedServerOptionalComponentDefaults(context.Background())
	if err != nil {
		t.Fatalf("ManagedServerOptionalComponentDefaults returned error: %v", err)
	}

	if len(result.Components) != 1 || result.Components[0].Component != "VeeamInstallerSvc" {
		t.Fatalf("optional components not decoded: %+v", result.Components)
	}
}
