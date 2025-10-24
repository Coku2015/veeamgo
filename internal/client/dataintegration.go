package client

import (
	"context"
	"fmt"
	"strings"
)

// PublishBackupContent starts a disk publishing session based on the provided specification.
func (c *Client) PublishBackupContent(ctx context.Context, spec map[string]any) (*Session, error) {
	if spec == nil {
		return nil, fmt.Errorf("disk publishing specification cannot be nil")
	}

	payload := shallowCopy(spec)
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/dataIntegration/publish", nil, payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// UnpublishBackupContent stops an active disk publishing session by mount ID.
func (c *Client) UnpublishBackupContent(ctx context.Context, mountID string) (*Session, error) {
	clean := strings.TrimSpace(mountID)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/dataIntegration/%s/unpublish", clean)
	var sess Session
	if err := c.postJSON(ctx, path, nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}
