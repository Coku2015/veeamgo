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

func TestCredentialsFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/credentials" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := credentialsResponse{
			Data:       []Credential{},
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
	after := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)
	before := after.Add(24 * time.Hour)

	_, err := client.Credentials(context.Background(), CredentialsFilter{
		Name:                    "svc-*",
		Type:                    "Windows",
		IncludeDefaultAppliance: true,
		CreatedAfter:            &after,
		CreatedBefore:           &before,
		OrderColumn:             "creationTime",
		OrderAscending:          &orderAsc,
		MaxItems:                3,
	})
	if err != nil {
		t.Fatalf("Credentials returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "svc-*")
	assertQueryEquals(t, captured, "typeFilter", "Windows")
	assertQueryEquals(t, captured, "includeDefaultApplianceCreds", "true")
	assertQueryEquals(t, captured, "createdAfterFilter", after.Format(time.RFC3339))
	assertQueryEquals(t, captured, "createdBeforeFilter", before.Format(time.RFC3339))
	assertQueryEquals(t, captured, "orderColumn", "creationTime")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "3")
}

func TestCloudCredentialsFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/cloudCredentials" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := cloudCredentialsResponse{
			Data:       []CloudCredential{},
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

	_, err := client.CloudCredentials(context.Background(), CloudCredentialsFilter{
		Name:        "azure-*",
		Type:        "AzureCompute",
		OrderColumn: "lastModified",
		OrderAsc:    &orderAsc,
		MaxItems:    10,
	})
	if err != nil {
		t.Fatalf("CloudCredentials returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "azure-*")
	assertQueryEquals(t, captured, "typeFilter", "AzureCompute")
	assertQueryEquals(t, captured, "orderColumn", "lastModified")
	assertQueryEquals(t, captured, "orderAsc", "false")
	assertQueryEquals(t, captured, "limit", "10")
}

func TestEncryptionPasswordsFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/encryptionPasswords" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := encryptionPasswordsResponse{
			Data:       []EncryptionPassword{},
			Pagination: paginationResult{},
		}
		payload, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	_, err := client.EncryptionPasswords(context.Background(), EncryptionPasswordsFilter{
		Hint:        "prod",
		OrderColumn: "modificationTime",
		MaxItems:    4,
	})
	if err != nil {
		t.Fatalf("EncryptionPasswords returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "hintFilter", "prod")
	assertQueryEquals(t, captured, "orderColumn", "modificationTime")
	assertQueryEquals(t, captured, "limit", "4")
}

func TestKMSServersFilterEncoding(t *testing.T) {
	var captured url.Values

	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/kmsServers" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		captured = req.URL.Query()
		resp := kmsServersResponse{
			Data:       []KMSServer{},
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

	_, err := client.KMSServers(context.Background(), KMSServersFilter{
		Name:        "vault-*",
		Type:        "KMS",
		OrderColumn: "name",
		OrderAsc:    &orderAsc,
		MaxItems:    2,
	})
	if err != nil {
		t.Fatalf("KMSServers returned error: %v", err)
	}

	if captured == nil {
		t.Fatalf("no query captured")
	}

	assertQueryEquals(t, captured, "nameFilter", "vault-*")
	assertQueryEquals(t, captured, "typeFilter", "KMS")
	assertQueryEquals(t, captured, "orderColumn", "name")
	assertQueryEquals(t, captured, "orderAsc", "true")
	assertQueryEquals(t, captured, "limit", "2")
}
