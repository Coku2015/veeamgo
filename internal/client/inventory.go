package client

import (
	"context"
	"fmt"
	"net/url"
)

type InventoryPagination struct {
	Skip  int `json:"skip,omitempty"`
	Limit int `json:"limit,omitempty"`
}

type InventoryRequest struct {
	Pagination    *InventoryPagination `json:"pagination,omitempty"`
	Filter        map[string]any       `json:"filter,omitempty"`
	Sorting       map[string]any       `json:"sorting,omitempty"`
	HierarchyType string               `json:"hierarchyType,omitempty"`
}

type InventoryResult struct {
	Data          []map[string]any `json:"data"`
	Pagination    paginationResult `json:"pagination"`
	HierarchyType string           `json:"hierarchyType"`
}

func (c *Client) InventoryServers(ctx context.Context, req InventoryRequest) (*InventoryResult, error) {
	payload := normalizeInventoryRequest(req)
	var result InventoryResult
	if err := c.postJSON(ctx, "/api/v1/inventory", nil, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) InventoryObjects(ctx context.Context, hostname string, req InventoryRequest) (*InventoryResult, error) {
	payload := normalizeInventoryRequest(req)
	var result InventoryResult
	path := fmt.Sprintf("/api/v1/inventory/%s", url.PathEscape(hostname))
	if err := c.postJSON(ctx, path, nil, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) InventoryProtectionGroups(ctx context.Context, req InventoryRequest) (*InventoryResult, error) {
	payload := normalizeInventoryRequest(req)
	var result InventoryResult
	if err := c.postJSON(ctx, "/api/v1/inventory/physical", nil, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) InventoryProtectionGroupItems(ctx context.Context, protectionGroupID string, req InventoryRequest) (*InventoryResult, error) {
	payload := normalizeInventoryRequest(req)
	var result InventoryResult
	path := fmt.Sprintf("/api/v1/inventory/physical/%s", url.PathEscape(protectionGroupID))
	if err := c.postJSON(ctx, path, nil, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func normalizeInventoryRequest(req InventoryRequest) InventoryRequest {
	payload := req
	if payload.Pagination != nil && payload.Pagination.Limit == 0 && payload.Pagination.Skip == 0 {
		payload.Pagination = nil
	}
	return payload
}
