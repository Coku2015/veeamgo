package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ProxyFilter controls proxy listing.
type ProxyFilter struct {
	Name           string
	Type           string
	HostID         string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

// Proxy summarises a backup proxy.
type Proxy struct {
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	Description           string         `json:"description"`
	Type                  string         `json:"type"`
	HostID                string         `json:"hostId,omitempty"`
	HostName              string         `json:"hostName,omitempty"`
	MaxTaskCount          int            `json:"maxTaskCount,omitempty"`
	TransportMode         string         `json:"transportMode,omitempty"`
	FailoverToNetwork     bool           `json:"failoverToNetwork,omitempty"`
	HostToProxyEncryption bool           `json:"hostToProxyEncryption,omitempty"`
	AutoSelectDatastores  bool           `json:"autoSelectDatastores,omitempty"`
	ConnectedDatastores   []string       `json:"connectedDatastores,omitempty"`
	Raw                   map[string]any `json:"-"`
}

// ProxiesResult contains proxy records and raw payload.
type ProxiesResult struct {
	Proxies    []Proxy
	Pagination paginationResult
	Raw        map[string]any
}

// ProxyDetail augments a proxy with raw payload.
type ProxyDetail struct {
	Proxy Proxy
	Raw   map[string]any
}

// ProxyState captures health/status for a proxy.
type ProxyState struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	HostID      string         `json:"hostId"`
	HostName    string         `json:"hostName"`
	IsDisabled  bool           `json:"isDisabled"`
	IsOnline    bool           `json:"isOnline"`
	IsOutOfDate bool           `json:"isOutOfDate"`
	Raw         map[string]any `json:"-"`
}

// ProxyStatesResult returns proxy state summaries.
type ProxyStatesResult struct {
	States     []ProxyState
	Pagination paginationResult
	Raw        map[string]any
}

// ServerVolume captures Hyper-V volume metadata for CBT configuration.
type ServerVolume struct {
	InventoryObject map[string]any `json:"inventoryObject"`
	VSSProvider     string         `json:"vssProvider"`
	MaxSnapshots    int            `json:"maxSnapshots"`
	Raw             map[string]any `json:"-"`
}

// ServerVolumesResult enumerates Hyper-V host volumes.
type ServerVolumesResult struct {
	ChangedBlockTracking  bool
	FailoverToVSSProvider bool
	Volumes               []ServerVolume
	Raw                   map[string]any
}

// OptionalComponent represents installable server component defaults.
type OptionalComponent struct {
	DisplayName string         `json:"displayName"`
	Component   string         `json:"optionalComponent"`
	Raw         map[string]any `json:"-"`
}

// OptionalComponentDefaults lists default optional components.
type OptionalComponentDefaults struct {
	Components []OptionalComponent
	Raw        map[string]any
}

// Proxies returns backup proxy inventory with optional filters.
func (c *Client) Proxies(ctx context.Context, filter ProxyFilter) (*ProxiesResult, error) {
	proxies := make([]Proxy, 0)
	rawItems := make([]map[string]any, 0)
	skip := 0
	remaining := filter.MaxItems
	pagination := paginationResult{}

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
		if filter.HostID != "" {
			query.Set("hostIdFilter", filter.HostID)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp struct {
			Data       []json.RawMessage `json:"data"`
			Pagination paginationResult  `json:"pagination"`
		}
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/proxies", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			payload := struct {
				ID          string         `json:"id"`
				Name        string         `json:"name"`
				Description string         `json:"description"`
				Type        string         `json:"type"`
				Server      map[string]any `json:"server"`
			}{}
			raw, err := decodeRawMessage(entry, &payload)
			if err != nil {
				return nil, fmt.Errorf("decode proxy payload: %w", err)
			}

			proxy := Proxy{
				ID:          payload.ID,
				Name:        payload.Name,
				Description: payload.Description,
				Type:        payload.Type,
				Raw:         raw,
			}
			populateProxyServerFields(&proxy, payload.Server)
			proxies = append(proxies, proxy)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(proxies) >= filter.MaxItems {
				pagination = resp.Pagination
				return &ProxiesResult{
					Proxies:    proxies[:filter.MaxItems],
					Pagination: pagination,
					Raw: map[string]any{
						"data":       rawItems[:filter.MaxItems],
						"pagination": pagination,
					},
				}, nil
			}
		}

		pagination = resp.Pagination
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}

		skip = resp.Pagination.Skip + resp.Pagination.Count
		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(proxies)
			if remaining <= 0 {
				break
			}
		}
	}

	return &ProxiesResult{
		Proxies:    proxies,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// ProxyDetail fetches detailed proxy payload.
func (c *Client) ProxyDetail(ctx context.Context, id string) (*ProxyDetail, error) {
	payload := make(map[string]any)
	path := fmt.Sprintf("/api/v1/backupInfrastructure/proxies/%s", id)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	proxy := Proxy{
		Raw: payload,
	}
	if v, ok := payload["id"].(string); ok {
		proxy.ID = v
	}
	if v, ok := payload["name"].(string); ok {
		proxy.Name = v
	}
	if v, ok := payload["description"].(string); ok {
		proxy.Description = v
	}
	if v, ok := payload["type"].(string); ok {
		proxy.Type = v
	}
	if server, ok := extractMap(payload, "server"); ok {
		populateProxyServerFields(&proxy, server)
	}

	return &ProxyDetail{
		Proxy: proxy,
		Raw:   payload,
	}, nil
}

// ProxyStates lists runtime states for proxies.
func (c *Client) ProxyStates(ctx context.Context, filter ProxyFilter) (*ProxyStatesResult, error) {
	states := make([]ProxyState, 0)
	rawItems := make([]map[string]any, 0)
	skip := 0
	remaining := filter.MaxItems
	pagination := paginationResult{}

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
		if filter.HostID != "" {
			query.Set("hostIdFilter", filter.HostID)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp struct {
			Data       []json.RawMessage `json:"data"`
			Pagination paginationResult  `json:"pagination"`
		}
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/proxies/states", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			payload := ProxyState{}
			raw, err := decodeRawMessage(entry, &payload)
			if err != nil {
				return nil, fmt.Errorf("decode proxy state: %w", err)
			}
			payload.Raw = raw
			states = append(states, payload)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(states) >= filter.MaxItems {
				pagination = resp.Pagination
				return &ProxyStatesResult{
					States:     states[:filter.MaxItems],
					Pagination: pagination,
					Raw: map[string]any{
						"data":       rawItems[:filter.MaxItems],
						"pagination": pagination,
					},
				}, nil
			}
		}

		pagination = resp.Pagination
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count
		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(states)
			if remaining <= 0 {
				break
			}
		}
	}

	return &ProxyStatesResult{
		States:     states,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// ManagedServerVolumes retrieves Hyper-V volume configuration for CBT.
func (c *Client) ManagedServerVolumes(ctx context.Context, serverID string) (*ServerVolumesResult, error) {
	payload := make(map[string]any)
	path := fmt.Sprintf("/api/v1/backupInfrastructure/managedServers/%s/volumes", serverID)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	result := &ServerVolumesResult{
		Raw: payload,
	}
	if v, ok := payload["changedBlockTracking"].(bool); ok {
		result.ChangedBlockTracking = v
	}
	if v, ok := payload["failoverToVSSProvider"].(bool); ok {
		result.FailoverToVSSProvider = v
	}
	if volumes, ok := payload["volumes"].([]any); ok {
		for _, entry := range volumes {
			rawVol, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			volume := ServerVolume{
				Raw: rawVol,
			}
			if inv, ok := rawVol["volume"].(map[string]any); ok {
				volume.InventoryObject = inv
			}
			if settings, ok := rawVol["volumeSettings"].(map[string]any); ok {
				if v, ok := settings["VSSprovider"].(string); ok {
					volume.VSSProvider = v
				}
				if v, ok := settings["maxSnapshots"].(float64); ok {
					volume.MaxSnapshots = int(v)
				}
			}
			result.Volumes = append(result.Volumes, volume)
		}
	}

	return result, nil
}

// ManagedServerOptionalComponentDefaults returns default optional components.
func (c *Client) ManagedServerOptionalComponentDefaults(ctx context.Context) (*OptionalComponentDefaults, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/backupInfrastructure/managedServers/optionalComponents/defaults", &payload); err != nil {
		return nil, err
	}

	result := &OptionalComponentDefaults{
		Raw: payload,
	}
	if components, ok := payload["optionalComponents"].([]any); ok {
		for _, entry := range components {
			rawComp, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			var comp OptionalComponent
			if _, err := decodeObject(rawComp, &comp); err == nil {
				comp.Raw = rawComp
				result.Components = append(result.Components, comp)
			}
		}
	}

	return result, nil
}

func populateProxyServerFields(proxy *Proxy, server map[string]any) {
	if server == nil {
		return
	}
	if v, ok := server["hostId"].(string); ok {
		proxy.HostID = v
	}
	if v, ok := server["hostName"].(string); ok {
		proxy.HostName = v
	}
	if v, ok := server["maxTaskCount"].(float64); ok {
		proxy.MaxTaskCount = int(v)
	}
	if v, ok := server["transportMode"].(string); ok {
		proxy.TransportMode = v
	}
	if v, ok := server["failoverToNetwork"].(bool); ok {
		proxy.FailoverToNetwork = v
	}
	if v, ok := server["hostToProxyEncryption"].(bool); ok {
		proxy.HostToProxyEncryption = v
	}
	if conn, ok := server["connectedDatastores"].(map[string]any); ok {
		if v, ok := conn["autoSelectEnabled"].(bool); ok {
			proxy.AutoSelectDatastores = v
		}
		if datastores, ok := conn["datastores"].([]any); ok {
			for _, entry := range datastores {
				dsMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				if datastore, ok := dsMap["datastore"].(map[string]any); ok {
					if name, ok := datastore["name"].(string); ok && name != "" {
						proxy.ConnectedDatastores = append(proxy.ConnectedDatastores, name)
					}
				}
			}
		}
	}
}

// matchesPattern proxies path match, reused from restore mounts.
