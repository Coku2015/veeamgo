package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestWANAcceleratorsFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id":   "d8ecb099-ca8e-4cd3-b12f-10f36508942b",
				"name": "enterprise03.tech.local",
				"server": map[string]any{
					"hostId":                   "b29b8591-bf31-4174-b69b-25ac296a20b2",
					"description":              "Created by TECH\\sheila.d.cory",
					"trafficPort":              6165,
					"streamsCount":             5,
					"highBandwidthModeEnabled": true,
				},
				"cache": map[string]any{
					"cacheSizeUnit": "GB",
					"cacheFolder":   "C:\\VeeamWAN",
					"cacheSize":     100,
				},
			},
			{
				"id":   "bbd849d3-226e-470c-804c-fdd2b09da683",
				"name": "enterprise05.tech.local",
				"server": map[string]any{
					"hostId":                   "612af0b0-ce61-4315-a82c-b056669e48ae",
					"description":              "Created by TECH\\sheila.d.cory",
					"trafficPort":              6165,
					"streamsCount":             5,
					"highBandwidthModeEnabled": false,
				},
				"cache": map[string]any{
					"cacheSizeUnit": "GB",
					"cacheFolder":   "C:\\VeeamWAN",
					"cacheSize":     50,
				},
			},
		},
		"pagination": map[string]any{
			"total": 2,
			"count": 2,
			"skip":  0,
			"limit": 200,
		},
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
	result, err := client.WANAccelerators(context.Background(), WANAcceleratorFilter{
		Name:           "enterprise*",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("WANAccelerators returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("expected captured query")
	}
	assertQueryEquals(t, captured, "nameFilter", "enterprise*")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "1")

	if len(result) != 1 {
		t.Fatalf("expected 1 accelerator, got %d", len(result))
	}
	if result[0].Name != "enterprise03.tech.local" {
		t.Fatalf("unexpected accelerator name: %s", result[0].Name)
	}
	if result[0].Cache == nil || result[0].Cache.CacheSize != 100 {
		t.Fatalf("cache not decoded: %+v", result[0].Cache)
	}
	if result[0].Server == nil || !result[0].Server.HighBandwidthModeEnabled {
		t.Fatalf("server details not decoded: %+v", result[0].Server)
	}
}

func TestWANAcceleratorDetail(t *testing.T) {
	payload := map[string]any{
		"id":   "d8ecb099-ca8e-4cd3-b12f-10f36508942b",
		"name": "enterprise03.tech.local",
		"server": map[string]any{
			"hostId":                   "b29b8591-bf31-4174-b69b-25ac296a20b2",
			"description":              "Created by TECH\\sheila.d.cory",
			"trafficPort":              6165,
			"streamsCount":             5,
			"highBandwidthModeEnabled": true,
		},
		"cache": map[string]any{
			"cacheSizeUnit": "GB",
			"cacheFolder":   "C:\\VeeamWAN",
			"cacheSize":     100,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var path string
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		path = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(body)),
		}, nil
	})

	accel, err := client.WANAccelerator(context.Background(), "d8ecb099-ca8e-4cd3-b12f-10f36508942b")
	if err != nil {
		t.Fatalf("WANAccelerator returned error: %v", err)
	}
	if path != "/api/v1/backupInfrastructure/wanAccelerators/d8ecb099-ca8e-4cd3-b12f-10f36508942b" {
		t.Fatalf("unexpected path %q", path)
	}
	if accel == nil || accel.Server == nil || accel.Server.TrafficPort != 6165 {
		t.Fatalf("server payload not decoded: %+v", accel)
	}
	if accel.Cache == nil || accel.Cache.CacheFolder != "C:\\VeeamWAN" {
		t.Fatalf("cache payload not decoded: %+v", accel)
	}
}
