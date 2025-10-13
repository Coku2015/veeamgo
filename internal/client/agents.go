package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type ProtectionGroupFilter struct {
	Name           string
	Type           string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type protectionGroupsResponse struct {
	Data       []ProtectionGroup `json:"data"`
	Pagination paginationResult  `json:"pagination"`
}

type ProtectionGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Disabled    bool   `json:"isDisabled"`
}

func (c *Client) ProtectionGroups(ctx context.Context, filter ProtectionGroupFilter) ([]ProtectionGroup, error) {
	results := make([]ProtectionGroup, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp protectionGroupsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/agents/protectionGroups", query, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

type ProtectionGroupDetail struct {
	Group ProtectionGroup
	Raw   map[string]any
}

func (c *Client) ProtectionGroupDetail(ctx context.Context, id string) (*ProtectionGroupDetail, error) {
	path := fmt.Sprintf("/api/v1/agents/protectionGroups/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode protection group payload: %w", err)
	}
	var group ProtectionGroup
	if err := json.Unmarshal(raw, &group); err != nil {
		return nil, fmt.Errorf("decode protection group payload: %w", err)
	}
	return &ProtectionGroupDetail{
		Group: group,
		Raw:   payload,
	}, nil
}

type DiscoveredEntityFilter struct {
	Name           string
	IPAddress      string
	State          string
	AgentStatus    string
	DriverStatus   string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type discoveredEntitiesResponse struct {
	Data       []DiscoveredEntity `json:"data"`
	Pagination paginationResult   `json:"pagination"`
}

type DiscoveredEntity struct {
	ID                      string     `json:"id"`
	Name                    string     `json:"name"`
	Type                    string     `json:"type"`
	State                   string     `json:"state"`
	AgentStatus             string     `json:"agentStatus"`
	AgentVersion            string     `json:"agentVersion"`
	DriverStatus            string     `json:"driverStatus"`
	DriverVersion           string     `json:"driverVersion"`
	RebootRequired          bool       `json:"rebootRequired"`
	IPAddresses             []string   `json:"ipAddresses"`
	LastConnected           *time.Time `json:"lastConnected"`
	OperatingSystem         string     `json:"operatingSystem"`
	OperatingSystemPlatform string     `json:"operatingSystemPlatform"`
	OperatingSystemVersion  string     `json:"operatingSystemVersion"`
}

func (c *Client) ProtectionGroupDiscoveredEntities(ctx context.Context, groupID string, filter DiscoveredEntityFilter) ([]DiscoveredEntity, error) {
	results := make([]DiscoveredEntity, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.IPAddress != "" {
			query.Set("ipAddressFilter", filter.IPAddress)
		}
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}
		if filter.AgentStatus != "" {
			query.Set("agentStatusFilter", filter.AgentStatus)
		}
		if filter.DriverStatus != "" {
			query.Set("driverStatusFilter", filter.DriverStatus)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		path := fmt.Sprintf("/api/v1/agents/protectionGroups/%s/discoveredEntities", groupID)
		var resp discoveredEntitiesResponse
		if err := c.getJSONWithQuery(ctx, path, query, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

type DiscoveredEntityDetail struct {
	Entity DiscoveredEntity
	Raw    map[string]any
}

func (c *Client) ProtectionGroupDiscoveredEntity(ctx context.Context, groupID, entityID string) (*DiscoveredEntityDetail, error) {
	path := fmt.Sprintf("/api/v1/agents/protectionGroups/%s/discoveredEntities/%s", groupID, entityID)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode discovered entity payload: %w", err)
	}
	var entity DiscoveredEntity
	if err := json.Unmarshal(raw, &entity); err != nil {
		return nil, fmt.Errorf("decode discovered entity payload: %w", err)
	}
	return &DiscoveredEntityDetail{
		Entity: entity,
		Raw:    payload,
	}, nil
}

type ProtectedComputerFilter struct {
	Name           string
	Type           string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type protectedComputersResponse struct {
	Data       []ProtectedComputer `json:"data"`
	Pagination paginationResult    `json:"pagination"`
}

type ProtectedComputer struct {
	ID               string                   `json:"id"`
	Name             string                   `json:"name"`
	Type             string                   `json:"type"`
	ProtectionGroups []ProtectedComputerGroup `json:"protectionGroups"`
}

type ProtectedComputerGroup struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CredentialsID string `json:"credentialsId"`
}

func (c *Client) ProtectedComputers(ctx context.Context, filter ProtectedComputerFilter) ([]ProtectedComputer, error) {
	results := make([]ProtectedComputer, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp protectedComputersResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/agents/protectedComputers", query, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

type ProtectedComputerDetail struct {
	Computer ProtectedComputer
	Raw      map[string]any
}

func (c *Client) ProtectedComputerByID(ctx context.Context, id string) (*ProtectedComputerDetail, error) {
	path := fmt.Sprintf("/api/v1/agents/protectedComputers/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode protected computer payload: %w", err)
	}
	var computer ProtectedComputer
	if err := json.Unmarshal(raw, &computer); err != nil {
		return nil, fmt.Errorf("decode protected computer payload: %w", err)
	}
	return &ProtectedComputerDetail{
		Computer: computer,
		Raw:      payload,
	}, nil
}

type LinuxPackageFilter struct {
	Name           string
	Distribution   string
	Bitness        string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type linuxPackageResponse struct {
	Data       []LinuxPackage   `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type LinuxPackage struct {
	Name         string `json:"packageName"`
	Distribution string `json:"distributionName"`
	Bitness      string `json:"packageBitness"`
}

func (c *Client) LinuxAgentPackages(ctx context.Context, filter LinuxPackageFilter) ([]LinuxPackage, error) {
	results := make([]LinuxPackage, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.Distribution != "" {
			query.Set("distributionFilter", filter.Distribution)
		}
		if filter.Bitness != "" {
			query.Set("packageBitnessFilter", filter.Bitness)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp linuxPackageResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/agents/packages/linux", query, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

type UnixPackageFilter struct {
	Name           string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type unixPackageResponse struct {
	Data       []UnixPackage    `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type UnixPackage struct {
	Name string `json:"packageName"`
}

func (c *Client) UnixAgentPackages(ctx context.Context, filter UnixPackageFilter) ([]UnixPackage, error) {
	results := make([]UnixPackage, 0)
	skip := 0
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp unixPackageResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/agents/packages/unix", query, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}
