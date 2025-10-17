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

func TestLicenseSocketsFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"name":          "host-1",
				"hostName":      "host-1",
				"hostId":        "11111111-2222-3333-4444-555555555555",
				"socketsNumber": 4,
				"coresNumber":   32,
				"type":          "Vmware",
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
	result, err := client.LicenseSockets(context.Background(), LicenseSocketsFilter{
		Name:           "host-*",
		HostName:       "host-1",
		HostID:         "11111111-2222-3333-4444-555555555555",
		SocketsNumber:  4,
		CoresNumber:    32,
		Type:           "Vmware",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("LicenseSockets returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "host-*")
	assertQueryEquals(t, captured, "hostNameFilter", "host-1")
	assertQueryEquals(t, captured, "hostIdFilter", "11111111-2222-3333-4444-555555555555")
	assertQueryEquals(t, captured, "socketsNumberFilter", "4")
	assertQueryEquals(t, captured, "coresNumberFilter", "32")
	assertQueryEquals(t, captured, "typeFilter", "Vmware")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Workloads) != 1 {
		t.Fatalf("expected workload in result: %+v", result)
	}
	if result.Raw == nil || len(result.Raw) == 0 {
		t.Fatalf("expected raw payload")
	}
}

func TestLicenseInstancesFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"name":                "job-1",
				"displayName":         "job-1",
				"hostName":            "host-1",
				"usedInstancesNumber": 2,
				"type":                "Vmware",
				"instanceId":          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				"platformType":        "Vmware",
				"canBeRevoked":        true,
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
	result, err := client.LicenseInstances(context.Background(), InstanceLicensesFilter{
		Name:                "job-*",
		HostName:            "host-1",
		UsedInstancesNumber: 2,
		Type:                "Vmware",
		InstanceID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		OrderColumn:         "usedInstancesNumber",
		OrderAscending:      &orderAsc,
		MaxItems:            1,
	})
	if err != nil {
		t.Fatalf("LicenseInstances returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "job-*")
	assertQueryEquals(t, captured, "hostNameFilter", "host-1")
	assertQueryEquals(t, captured, "usedInstancesNumberFilter", "2")
	assertQueryEquals(t, captured, "typeFilter", "Vmware")
	assertQueryEquals(t, captured, "instanceIdFilter", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	assertQueryEquals(t, captured, "orderColumn", "usedInstancesNumber")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Workloads) != 1 {
		t.Fatalf("expected workload in result: %+v", result)
	}
	if result.Workloads[0].Raw["instanceId"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("raw workload not preserved")
	}
}

func TestGlobalVMExclusionsFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{
				"id": "ex-1",
				"inventoryObject": map[string]any{
					"name":     "vm-1",
					"platform": "VSphere",
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
	result, err := client.GlobalVMExclusions(context.Background(), GlobalVMExclusionsFilter{
		OrderColumn:    "id",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("GlobalVMExclusions returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "orderColumn", "id")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Exclusions) != 1 {
		t.Fatalf("expected exclusions in result: %+v", result)
	}
	if result.Exclusions[0].InventoryObject["name"] != "vm-1" {
		t.Fatalf("inventory object not preserved")
	}
}

func TestServicesFilterEncoding(t *testing.T) {
	var captured url.Values

	respPayload := map[string]any{
		"data": []map[string]any{
			{"name": "Enterprise Manager", "port": 9398},
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
	result, err := client.Services(context.Background(), ServicesFilter{
		Name:           "Enterprise*",
		OrderColumn:    "name",
		OrderAscending: &orderAsc,
		MaxItems:       1,
	})
	if err != nil {
		t.Fatalf("Services returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}
	assertQueryEquals(t, captured, "nameFilter", "Enterprise*")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "1")

	if result == nil || len(result.Services) != 1 {
		t.Fatalf("expected services in result: %+v", result)
	}
	if result.Services[0].Raw["port"] != float64(9398) {
		t.Fatalf("raw port not preserved")
	}
}

func TestLicenseSummaryDecode(t *testing.T) {
	respPayload := map[string]any{
		"status":         "Active",
		"edition":        "Enterprise",
		"type":           "Perpetual",
		"licensedTo":     "Veeam",
		"supportId":      "123456789",
		"expirationDate": "2025-12-31T23:59:59Z",
		"instanceLicenseSummary": map[string]any{
			"licensedInstancesNumber": 100,
			"usedInstancesNumber":     90,
			"newInstancesNumber":      5,
			"rentalInstancesNumber":   0,
		},
		"capacityLicenseSummary": map[string]any{
			"licensedCapacityTb": 50,
			"usedCapacityTb":     30,
		},
		"socketLicenseSummary": map[string]any{
			"licensedSocketsNumber":  20,
			"usedSocketsNumber":      16,
			"remainingSocketsNumber": 4,
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

	summary, err := client.LicenseSummary(context.Background())
	if err != nil {
		t.Fatalf("LicenseSummary returned error: %v", err)
	}

	if summary.InstanceSummary == nil || summary.InstanceSummary.LicensedInstancesNumber != 100 {
		t.Fatalf("instance summary not decoded: %+v", summary.InstanceSummary)
	}
	if summary.CapacitySummary == nil || summary.CapacitySummary.UsedCapacityTB != 30 {
		t.Fatalf("capacity summary not decoded: %+v", summary.CapacitySummary)
	}
	if summary.SocketSummary == nil || summary.SocketSummary.RemainingSocketsNumber != 4 {
		t.Fatalf("socket summary not decoded: %+v", summary.SocketSummary)
	}
	if summary.Raw["edition"] != "Enterprise" {
		t.Fatalf("raw payload missing field: %+v", summary.Raw)
	}
}

func TestGeneralOptionsDecode(t *testing.T) {
	respPayload := map[string]any{
		"notificationEnabled": true,
		"notifications": map[string]any{
			"storageSpaceThresholdEnabled":   true,
			"datastoreSpaceThresholdEnabled": false,
			"skipVMSpaceThresholdEnabled":    true,
			"notifyOnSupportExpiration":      true,
			"notifyOnUpdates":                false,
		},
		"siemIntegration": map[string]any{
			"SNMPEventsEnabled":   true,
			"SyslogEventsEnabled": false,
		},
	}
	body, err := json.Marshal(respPayload)
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

	opts, err := client.GeneralOptions(context.Background())
	if err != nil {
		t.Fatalf("GeneralOptions returned error: %v", err)
	}
	if path != "/api/v1/generalOptions" {
		t.Fatalf("unexpected path %q", path)
	}

	if !opts.NotificationEnabled {
		t.Fatalf("notification flag not decoded: %+v", opts)
	}
	if !opts.StorageSpaceThresholdEnabled || opts.DatastoreSpaceThresholdEnabled {
		t.Fatalf("storage/datastore thresholds unexpected: %+v", opts)
	}
	if !opts.SkipVMSpaceThresholdEnabled {
		t.Fatalf("skip VM threshold not decoded: %+v", opts)
	}
	if !opts.NotifyOnSupportExpiration || opts.NotifyOnUpdates {
		t.Fatalf("notification toggles unexpected: %+v", opts)
	}
	if !opts.SIEMSNMPEnabled || opts.SIEMSyslogEnabled {
		t.Fatalf("siem flags unexpected: %+v", opts)
	}
	if opts.Raw == nil || len(opts.Raw) == 0 {
		t.Fatalf("raw payload missing")
	}
}

func TestConfigBackupDecode(t *testing.T) {
	respPayload := map[string]any{
		"isEnabled":           true,
		"backupRepositoryId":  "repo-1",
		"restorePointsToKeep": 14,
		"encryption": map[string]any{
			"isEnabled":  true,
			"passwordId": "pass-1",
		},
		"lastSuccessfulBackup": map[string]any{
			"sessionId":          "session-1",
			"lastSuccessfulTime": "2025-02-01T10:00:00Z",
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

	cfg, err := client.ConfigBackup(context.Background())
	if err != nil {
		t.Fatalf("ConfigBackup returned error: %v", err)
	}

	if !cfg.EncryptionEnabled || cfg.EncryptionPassword != "pass-1" {
		t.Fatalf("encryption details missing: %+v", cfg)
	}
	if cfg.LastSessionID != "session-1" {
		t.Fatalf("last session id missing: %+v", cfg)
	}
	if cfg.LastRunTime == nil || !cfg.LastRunTime.Time.Equal(time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("last run time missing: %+v", cfg.LastRunTime)
	}
}
