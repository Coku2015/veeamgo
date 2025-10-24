package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Coku2015/veeamgo/pkg/apiversion"
)

func TestNegotiateAPIVersionFallback(t *testing.T) {
	attempts := 0
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		attempts++
		version := req.Header.Get(apiversion.HeaderName)
		switch attempts {
		case 1:
			if version != "1.3-rev0" {
				t.Fatalf("expected first attempt to use 1.3-rev0, got %s", version)
			}
			payload := []byte(`{"message":"invalid api version"}`)
			return &http.Response{
				StatusCode: http.StatusNotAcceptable,
				Header:     make(http.Header),
				Body:       ioNopCloser(bytes.NewReader(payload)),
			}, nil
		case 2:
			if version != "1.2-rev1" {
				t.Fatalf("expected fallback to 1.2-rev1, got %s", version)
			}
			info := ServerInfo{BuildVersion: "12.3.1.1139"}
			payload, _ := json.Marshal(info)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       ioNopCloser(bytes.NewReader(payload)),
			}, nil
		default:
			t.Fatalf("unexpected number of attempts: %d", attempts)
			return nil, nil
		}
	})
	client.supportedVersions = []string{"1.3-rev0", "1.2-rev1"}

	result, err := client.NegotiateAPIVersion(context.Background())
	if err != nil {
		t.Fatalf("NegotiateAPIVersion returned error: %v", err)
	}
	if result.SelectedVersion != "1.2-rev1" {
		t.Fatalf("expected selection to be 1.2-rev1, got %s", result.SelectedVersion)
	}
	if client.APIVersion() != "1.2-rev1" {
		t.Fatalf("client api version not updated, got %s", client.APIVersion())
	}
	if !result.ServerVersionKnown || result.ServerSuggestedVersion != "1.2-rev1" {
		t.Fatalf("unexpected server version metadata: %#v", result)
	}
}

func TestNegotiateAPIVersionOverrideFailure(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		payload := []byte(`{"message":"invalid api version"}`)
		return &http.Response{
			StatusCode: http.StatusNotAcceptable,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})
	if err := client.SetAPIVersion("1.2-rev0", true); err != nil {
		t.Fatalf("SetAPIVersion returned error: %v", err)
	}

	if _, err := client.NegotiateAPIVersion(context.Background()); err == nil {
		t.Fatalf("expected negotiation to fail for forced version")
	}
}

func TestSetAPIVersionDefaults(t *testing.T) {
	client := testClientWithResponder(t, func(req *http.Request) (*http.Response, error) {
		info := ServerInfo{BuildVersion: "13.0.0.4967"}
		payload, _ := json.Marshal(info)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(bytes.NewReader(payload)),
		}, nil
	})

	if err := client.SetAPIVersion("", false); err != nil {
		t.Fatalf("SetAPIVersion returned error: %v", err)
	}
	if client.APIVersion() != apiversion.DefaultVersion {
		t.Fatalf("expected default api version, got %s", client.APIVersion())
	}

	if _, err := client.NegotiateAPIVersion(context.Background()); err != nil {
		t.Fatalf("negotiation with default version failed: %v", err)
	}
}
