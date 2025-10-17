package client

import (
	"context"
	"fmt"
	"strings"
)

// ConnectionCertificateRequest represents the payload for retrieving a host fingerprint/certificate.
type ConnectionCertificateRequest struct {
	ServerName             string `json:"serverName"`
	Type                   string `json:"type"`
	CredentialsStorageType string `json:"credentialsStorageType,omitempty"`
	Port                   int    `json:"port,omitempty"`
	HandshakeCode          string `json:"handshakeCode,omitempty"`
}

// ConnectionCertificateModel contains the SSH fingerprint or TLS certificate for a host.
type ConnectionCertificateModel struct {
	Fingerprint string         `json:"fingerprint"`
	Certificate map[string]any `json:"certificate,omitempty"`
}

// ConnectionCertificate fetches the SSH fingerprint or TLS certificate for the given host.
func (c *Client) ConnectionCertificate(ctx context.Context, req ConnectionCertificateRequest) (*ConnectionCertificateModel, error) {
	if strings.TrimSpace(req.ServerName) == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if strings.TrimSpace(req.Type) == "" {
		return nil, fmt.Errorf("server type is required")
	}

	payload := map[string]any{
		"serverName": req.ServerName,
		"type":       req.Type,
	}
	if strings.TrimSpace(req.CredentialsStorageType) != "" {
		payload["credentialsStorageType"] = req.CredentialsStorageType
	}
	if req.Port > 0 {
		payload["port"] = req.Port
	}
	if strings.TrimSpace(req.HandshakeCode) != "" {
		payload["handshakeCode"] = req.HandshakeCode
	}

	var result ConnectionCertificateModel
	if err := c.postJSON(ctx, "/api/v1/connectionCertificate", nil, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
