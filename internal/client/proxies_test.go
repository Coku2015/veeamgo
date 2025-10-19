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

func TestCreateProxy(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST method, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/proxies" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured = body

		payload := Session{
			ID: "session-123",
		}
		resp, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(resp)),
		}, nil
	})

	spec := map[string]any{
		"name":        "proxy01",
		"description": "Created via tests",
		"type":        "ViProxy",
		"server": map[string]any{
			"hostId":        "host-1",
			"transportMode": "auto",
		},
	}

	session, err := client.CreateProxy(context.Background(), spec)
	if err != nil {
		t.Fatalf("CreateProxy returned error: %v", err)
	}
	if session == nil || session.ID != "session-123" {
		t.Fatalf("unexpected session: %+v", session)
	}

	if captured["name"] != "proxy01" {
		t.Fatalf("name not forwarded: %v", captured["name"])
	}
	server, ok := captured["server"].(map[string]any)
	if !ok {
		t.Fatalf("server payload missing")
	}
	if server["hostId"] != "host-1" {
		t.Fatalf("hostId not forwarded: %v", server["hostId"])
	}
	if server["transportMode"] != "auto" {
		t.Fatalf("transportMode not forwarded: %v", server["transportMode"])
	}
}

func TestCreateProxyNilSpec(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be sent")
		return nil, nil
	})

	if _, err := client.CreateProxy(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

func TestDeleteProxy(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/proxies/proxy-1" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(nil)),
		}, nil
	})

	if err := client.DeleteProxy(context.Background(), "proxy-1"); err != nil {
		t.Fatalf("DeleteProxy returned error: %v", err)
	}
}

func TestDeleteProxyEmptyID(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be sent")
		return nil, nil
	})

	if err := client.DeleteProxy(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty id")
	}
}

func TestEnableProxy(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/proxies/proxy-1/enable" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(nil)),
		}, nil
	})

	if err := client.EnableProxy(context.Background(), "proxy-1"); err != nil {
		t.Fatalf("EnableProxy returned error: %v", err)
	}
}

func TestDisableProxy(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/proxies/proxy-1/disable" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(nil)),
		}, nil
	})

	if err := client.DisableProxy(context.Background(), "proxy-1"); err != nil {
		t.Fatalf("DisableProxy returned error: %v", err)
	}
}

func TestUpdateProxy(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/proxies/proxy-1" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		payload := Session{ID: "session-999"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"name":   "proxy01",
		"type":   "ViProxy",
		"server": map[string]any{"hostId": "host-1"},
	}

	session, err := client.UpdateProxy(context.Background(), "proxy-1", spec)
	if err != nil {
		t.Fatalf("UpdateProxy returned error: %v", err)
	}
	if session == nil || session.ID != "session-999" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["name"] != "proxy01" {
		t.Fatalf("payload missing name: %v", captured)
	}
	server, ok := captured["server"].(map[string]any)
	if !ok || server["hostId"] != "host-1" {
		t.Fatalf("payload missing server.hostId: %v", captured["server"])
	}
}

func TestUpdateProxyValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be sent")
		return nil, nil
	})

	if _, err := client.UpdateProxy(context.Background(), "", map[string]any{}); err == nil {
		t.Fatalf("expected error for empty id")
	}
	if _, err := client.UpdateProxy(context.Background(), "proxy-1", nil); err == nil {
		t.Fatalf("expected error for nil spec")
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
