package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Coku2015/veeamgo/pkg/apiversion"
)

// NegotiationResult describes the outcome of API version negotiation.
type NegotiationResult struct {
	SelectedVersion          string
	Attempts                 []string
	ServerInfo               *ServerInfo
	ServerSuggestedVersion   string
	ServerVersionKnown       bool
	ServerNewerThanSupported bool
	Forced                   bool
}

// APIVersion returns the currently selected REST revision.
func (c *Client) APIVersion() string {
	version := strings.TrimSpace(c.apiVersion)
	if version == "" {
		return apiversion.DefaultVersion
	}
	return version
}

// SetAPIVersion sets the preferred REST revision. When override is true the
// negotiator will not fall back to older revisions.
func (c *Client) SetAPIVersion(version string, override bool) error {
	version = apiversion.Normalize(version)
	if version == "" {
		c.apiVersion = apiversion.DefaultVersion
		c.overrideVersion = false
		return nil
	}
	c.apiVersion = version
	c.overrideVersion = override
	return nil
}

// NegotiateAPIVersion ensures that the client uses an API revision supported by the server.
func (c *Client) NegotiateAPIVersion(ctx context.Context) (*NegotiationResult, error) {
    c.negotiating = true
    defer func() {
        c.negotiating = false
    }()

 candidates := c.candidateVersions()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no api version candidates available")
	}

	result := &NegotiationResult{
		Forced: c.overrideVersion,
	}

	original := c.apiVersion
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		result.Attempts = append(result.Attempts, candidate)
		if err := c.SetAPIVersion(candidate, c.overrideVersion); err != nil {
			c.apiVersion = original
			return nil, err
		}

		info, err := c.ServerInfo(ctx)
		if err == nil {
			result.SelectedVersion = candidate
			result.ServerInfo = info
			if info != nil {
				suggested, known, newer := apiversion.SuggestByBuild(info.BuildVersion)
				result.ServerSuggestedVersion = suggested
				result.ServerVersionKnown = known
				result.ServerNewerThanSupported = newer
			}
			return result, nil
		}

		var apiErr *APIError
		if c.overrideVersion {
			c.apiVersion = original
			return nil, fmt.Errorf("api version %s is not supported by the server: %w", candidate, err)
		}
		if errors.As(err, &apiErr) && isNegotiationStatus(apiErr.StatusCode) {
			continue
		}

		c.apiVersion = original
		return nil, err
	}

	c.apiVersion = original
	return nil, fmt.Errorf("server does not support any known api versions (attempted: %s)", strings.Join(result.Attempts, ", "))
}

func (c *Client) candidateVersions() []string {
	primary := c.APIVersion()
	if c.overrideVersion {
		return []string{primary}
	}

	versions := append([]string{primary}, c.availableSupportedVersions()...)
	return apiversion.MergeCandidates(versions...)
}

func (c *Client) availableSupportedVersions() []string {
	if len(c.supportedVersions) == 0 {
		return apiversion.Supported()
	}
	return append([]string(nil), c.supportedVersions...)
}

func isNegotiationStatus(code int) bool {
	switch code {
	case http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusNotAcceptable:
		return true
	default:
		return false
	}
}
