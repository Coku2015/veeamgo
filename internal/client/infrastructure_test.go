package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestCreateManagedServer(t *testing.T) {
	var payload map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/managedServers" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		resp := Session{
			ID:           "session-789",
			Name:         "Add Managed Server",
			SessionType:  "Infrastructure",
			State:        "InProgress",
			CreationTime: time.Date(2025, 10, 17, 11, 30, 0, 0, time.UTC),
		}
		bytesResp, _ := json.Marshal(resp)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(bytesResp)),
		}, nil
	})

	spec := map[string]any{
		"name":          "vc01.example.com",
		"description":   "VMware vCenter",
		"type":          "ViHost",
		"credentialsId": "cred-123",
	}

	session, err := client.CreateManagedServer(context.Background(), spec)
	if err != nil {
		t.Fatalf("CreateManagedServer returned error: %v", err)
	}
	if session == nil {
		t.Fatalf("expected session result")
	}
	if session.ID != "session-789" {
		t.Fatalf("unexpected session ID %s", session.ID)
	}

	if payload == nil {
		t.Fatalf("expected request payload to be captured")
	}
	if got := payload["type"]; got != "ViHost" {
		t.Fatalf("unexpected payload type %v", got)
	}
	if got := payload["credentialsId"]; got != "cred-123" {
		t.Fatalf("unexpected credentialsId %v", got)
	}
}

func TestCreateManagedServerNilSpec(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	session, err := client.CreateManagedServer(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil spec")
	}
	if session != nil {
		t.Fatalf("expected nil session result")
	}
}

func TestDeleteRepository(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/repositories/repo-1" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		if req.URL.RawQuery != "deleteBackups=true" {
			t.Fatalf("unexpected query %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(nil)),
		}, nil
	})

	if err := client.DeleteRepository(context.Background(), "repo-1", true); err != nil {
		t.Fatalf("DeleteRepository returned error: %v", err)
	}
}

func TestDeleteRepositoryValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if err := client.DeleteRepository(context.Background(), "", false); err == nil {
		t.Fatalf("expected error for empty id")
	}
}

func TestDeleteManagedServer(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/managedServers/server-1" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		payload := Session{
			ID:           "session-del-1",
			State:        "InProgress",
			SessionType:  "InfrastructureItemDeletion",
			CreationTime: time.Date(2025, 10, 18, 9, 30, 0, 0, time.UTC),
		}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	session, err := client.DeleteManagedServer(context.Background(), "server-1")
	if err != nil {
		t.Fatalf("DeleteManagedServer returned error: %v", err)
	}
	if session == nil {
		t.Fatalf("expected session")
	}
	if session.ID != "session-del-1" {
		t.Fatalf("unexpected session id %s", session.ID)
	}
}

func TestDeleteManagedServerValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.DeleteManagedServer(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty id")
	}
}

func TestCreateRepository(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/repositories" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}

		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		payload := Session{ID: "session-123"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"name":        "Repo01",
		"description": "Test repository",
		"type":        "WinLocal",
		"isDisabled":  false,
	}

	session, err := client.CreateRepository(context.Background(), spec)
	if err != nil {
		t.Fatalf("CreateRepository returned error: %v", err)
	}
	if session == nil || session.ID != "session-123" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["name"] != "Repo01" {
		t.Fatalf("payload missing name: %v", captured["name"])
	}
}

func TestCreateRepositoryNilSpec(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be executed")
		return nil, nil
	})

	if _, err := client.CreateRepository(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

func TestUpdateRepository(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/backupInfrastructure/repositories/repo-1" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		payload := Session{ID: "session-456"}
		buf, _ := json.Marshal(payload)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(buf)),
		}, nil
	})

	spec := map[string]any{
		"id":          "repo-1",
		"name":        "Repo01",
		"description": "Test repository",
		"type":        "WinLocal",
	}

	session, err := client.UpdateRepository(context.Background(), "repo-1", spec)
	if err != nil {
		t.Fatalf("UpdateRepository returned error: %v", err)
	}
	if session == nil || session.ID != "session-456" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if captured["id"] != "repo-1" {
		t.Fatalf("payload missing id: %v", captured["id"])
	}
}
