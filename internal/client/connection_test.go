package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestConnectionCertificate(t *testing.T) {
	var captured map[string]any

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/v1/connectionCertificate" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured = body

		resp := ConnectionCertificateModel{
			Fingerprint: "ssh-rsa 3072 AAAAB3NzaC1yc2EAAAADAQABAAABAQC1",
			Certificate: map[string]any{"commonName": "repo01.lab.local"},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	result, err := client.ConnectionCertificate(context.Background(), ConnectionCertificateRequest{
		ServerName:             "repo01.lab.local",
		Type:                   "LinuxHost",
		CredentialsStorageType: "Permanent",
		Port:                   22,
		HandshakeCode:          "123456",
	})
	if err != nil {
		t.Fatalf("ConnectionCertificate returned error: %v", err)
	}
	if result == nil || result.Fingerprint == "" {
		t.Fatalf("expected fingerprint result")
	}
	if captured["serverName"] != "repo01.lab.local" {
		t.Fatalf("unexpected serverName %v", captured["serverName"])
	}
	if captured["type"] != "LinuxHost" {
		t.Fatalf("unexpected type %v", captured["type"])
	}
	if captured["credentialsStorageType"] != "Permanent" {
		t.Fatalf("unexpected credentialsStorageType %v", captured["credentialsStorageType"])
	}
	if captured["port"] != float64(22) {
		t.Fatalf("unexpected port %v", captured["port"])
	}
	if captured["handshakeCode"] != "123456" {
		t.Fatalf("unexpected handshakeCode %v", captured["handshakeCode"])
	}
}

func TestConnectionCertificateValidation(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf("request should not be sent")
		return nil, nil
	})

	if _, err := client.ConnectionCertificate(context.Background(), ConnectionCertificateRequest{}); err == nil {
		t.Fatalf("expected error for missing fields")
	}
}
