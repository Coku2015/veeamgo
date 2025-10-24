package client

import (
	"context"
	"fmt"
	"strings"
)

// StartVSphereInstantRecovery starts Instant Recovery for a VMware vSphere VM.
func (c *Client) StartVSphereInstantRecovery(ctx context.Context, spec map[string]any) (*Session, error) {
	if spec == nil {
		return nil, fmt.Errorf("instant recovery specification cannot be nil")
	}

	payload := shallowCopy(spec)
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/restore/instantRecovery/vSphere/vm", nil, payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// StartHyperVInstantRecovery starts Instant Recovery for a Microsoft Hyper-V VM.
func (c *Client) StartHyperVInstantRecovery(ctx context.Context, spec map[string]any) (*Session, error) {
	if spec == nil {
		return nil, fmt.Errorf("instant recovery specification cannot be nil")
	}

	payload := shallowCopy(spec)
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/restore/instantRecovery/hyperV/vm", nil, payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// UnmountVSphereInstantRecovery stops publishing a VMware vSphere instant recovery mount.
func (c *Client) UnmountVSphereInstantRecovery(ctx context.Context, mountID string) (*Session, error) {
	clean := strings.TrimSpace(mountID)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/restore/instantRecovery/vSphere/vm/%s/unmount", clean)
	var sess Session
	if err := c.postJSON(ctx, path, nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// UnmountHyperVInstantRecovery stops publishing a Microsoft Hyper-V instant recovery mount.
func (c *Client) UnmountHyperVInstantRecovery(ctx context.Context, mountID string) (*Session, error) {
	clean := strings.TrimSpace(mountID)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/restore/instantRecovery/hyperV/vm/%s/unmount", clean)
	var sess Session
	if err := c.postJSON(ctx, path, nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// MigrateVSphereInstantRecovery triggers quick migration for a vSphere Instant Recovery mount.
func (c *Client) MigrateVSphereInstantRecovery(ctx context.Context, mountID string, spec map[string]any) (*Session, error) {
	clean := strings.TrimSpace(mountID)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}
	if spec == nil {
		return nil, fmt.Errorf("migration specification cannot be nil")
	}

	payload := shallowCopy(spec)
	path := fmt.Sprintf("/api/v1/restore/instantRecovery/vSphere/vm/%s/migrate", clean)
	var sess Session
	if err := c.postJSON(ctx, path, nil, payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// MigrateHyperVInstantRecovery triggers migration for a Hyper-V Instant Recovery mount.
func (c *Client) MigrateHyperVInstantRecovery(ctx context.Context, mountID string) (*Session, error) {
	clean := strings.TrimSpace(mountID)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/restore/instantRecovery/hyperV/vm/%s/migrate", clean)
	if err := c.postJSON(ctx, path, nil, nil, nil); err != nil {
		return nil, err
	}
	return nil, nil
}
