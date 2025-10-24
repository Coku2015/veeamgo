package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestStartVSphereInstantRecovery(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/vSphere/vm" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		payload := Session{ID: "ir-session-1", State: "InProgress", CreationTime: time.Now()}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"restorePointId": "rp-1",
		"type":           "OriginalLocation",
	}

	session, err := client.StartVSphereInstantRecovery(context.Background(), spec)
	if err != nil {
		t.Fatalf("StartVSphereInstantRecovery returned error: %v", err)
	}
	if session == nil || session.ID != "ir-session-1" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["type"] != "OriginalLocation" {
		t.Fatalf("unexpected type %v", captured["type"])
	}
}

func TestStartVSphereInstantRecoveryValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.StartVSphereInstantRecovery(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

func TestStartHyperVInstantRecovery(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/hyperV/vm" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		payload := Session{ID: "ir-session-hv", State: "InProgress", CreationTime: time.Now()}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"restorePointId": "rp-hv-1",
		"type":           "OriginalLocation",
	}

	session, err := client.StartHyperVInstantRecovery(context.Background(), spec)
	if err != nil {
		t.Fatalf("StartHyperVInstantRecovery returned error: %v", err)
	}
	if session == nil || session.ID != "ir-session-hv" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["restorePointId"] != "rp-hv-1" {
		t.Fatalf("unexpected restorePointId %v", captured["restorePointId"])
	}
}

func TestStartHyperVInstantRecoveryValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.StartHyperVInstantRecovery(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

func TestUnmountVSphereInstantRecovery(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/vSphere/vm/mount-1/unmount" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		payload := Session{ID: "unmount-session", State: "InProgress"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	session, err := client.UnmountVSphereInstantRecovery(context.Background(), "mount-1")
	if err != nil {
		t.Fatalf("UnmountVSphereInstantRecovery returned error: %v", err)
	}
	if session == nil || session.ID != "unmount-session" {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestUnmountHyperVInstantRecovery(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/hyperV/vm/mount-hv/unmount" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		payload := Session{ID: "unmount-hv", State: "InProgress"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	session, err := client.UnmountHyperVInstantRecovery(context.Background(), "mount-hv")
	if err != nil {
		t.Fatalf("UnmountHyperVInstantRecovery returned error: %v", err)
	}
	if session == nil || session.ID != "unmount-hv" {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestUnmountInstantRecoveryValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.UnmountVSphereInstantRecovery(context.Background(), " "); err == nil {
		t.Fatalf("expected error for empty mount id (vsphere)")
	}
	if _, err := client.UnmountHyperVInstantRecovery(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty mount id (hyperv)")
	}
}

func TestMigrateVSphereInstantRecovery(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/vSphere/vm/mount-1/migrate" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		payload := Session{ID: "ir-migrate-vsphere", State: "InProgress"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"destinationHost": map[string]any{
			"name": "esx02.lab.local",
			"type": "HostSystem",
		},
	}

	session, err := client.MigrateVSphereInstantRecovery(context.Background(), "mount-1", spec)
	if err != nil {
		t.Fatalf("MigrateVSphereInstantRecovery returned error: %v", err)
	}
	if session == nil || session.ID != "ir-migrate-vsphere" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["destinationHost"] == nil {
		t.Fatalf("destinationHost missing in payload")
	}
}

func TestMigrateVSphereInstantRecoveryValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.MigrateVSphereInstantRecovery(context.Background(), "mount-1", nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
	if _, err := client.MigrateVSphereInstantRecovery(context.Background(), "", map[string]any{}); err == nil {
		t.Fatalf("expected error for empty mount id")
	}
}

func TestMigrateHyperVInstantRecovery(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/restore/instantRecovery/hyperV/vm/mount-hv/migrate" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(nil)),
		}, nil
	})

	session, err := client.MigrateHyperVInstantRecovery(context.Background(), "mount-hv")
	if err != nil {
		t.Fatalf("MigrateHyperVInstantRecovery returned error: %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil session for Hyper-V migration")
	}

	if _, err := client.MigrateHyperVInstantRecovery(context.Background(), " "); err == nil {
		t.Fatalf("expected error for empty mount id")
	}
}
