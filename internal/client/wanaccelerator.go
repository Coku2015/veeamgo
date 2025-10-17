package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// WANAcceleratorFilter controls WAN accelerator listing.
type WANAcceleratorFilter struct {
	Name           string
	OrderColumn    string
	OrderAscending *bool
	Skip           int
	MaxItems       int
}

// WANAccelerator represents a WAN accelerator record.
type WANAccelerator struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Server *WANAcceleratorServer `json:"server,omitempty"`
	Cache  *WANAcceleratorCache  `json:"cache,omitempty"`
	Raw    map[string]any        `json:"-"`
}

// WANAcceleratorServer captures WAN accelerator server settings.
type WANAcceleratorServer struct {
	HostID                   string `json:"hostId"`
	Description              string `json:"description"`
	TrafficPort              int    `json:"trafficPort"`
	StreamsCount             int    `json:"streamsCount"`
	HighBandwidthModeEnabled bool   `json:"highBandwidthModeEnabled"`
}

// WANAcceleratorCache captures cache configuration.
type WANAcceleratorCache struct {
	CacheFolder   string `json:"cacheFolder"`
	CacheSize     int    `json:"cacheSize"`
	CacheSizeUnit string `json:"cacheSizeUnit"`
}

type wanAcceleratorsResponse struct {
	Data       []map[string]any `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// WANAccelerators lists WAN accelerators using the provided filter.
func (c *Client) WANAccelerators(ctx context.Context, filter WANAcceleratorFilter) ([]WANAccelerator, error) {
	accelerators := make([]WANAccelerator, 0)
	skip := filter.Skip
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

		var resp wanAcceleratorsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/wanAccelerators", query, &resp); err != nil {
			return nil, err
		}

		for _, raw := range resp.Data {
			accel, err := decodeWANAccelerator(raw)
			if err != nil {
				return nil, fmt.Errorf("decode wan accelerator: %w", err)
			}
			accelerators = append(accelerators, accel)
			if filter.MaxItems > 0 && len(accelerators) >= filter.MaxItems {
				return accelerators[:filter.MaxItems], nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}

		skip = resp.Pagination.Skip + resp.Pagination.Count
		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(accelerators)
			if remaining <= 0 {
				break
			}
		}
	}

	return accelerators, nil
}

// WANAccelerator fetches a WAN accelerator by ID.
func (c *Client) WANAccelerator(ctx context.Context, id string) (*WANAccelerator, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("wan accelerator id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/backupInfrastructure/wanAccelerators/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	accel, err := decodeWANAccelerator(payload)
	if err != nil {
		return nil, err
	}
	return &accel, nil
}

// WANAcceleratorByName resolves a WAN accelerator by its name, enforcing exact match.
func (c *Client) WANAcceleratorByName(ctx context.Context, name string) (*WANAccelerator, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("wan accelerator name cannot be empty")
	}

	accelerators, err := c.WANAccelerators(ctx, WANAcceleratorFilter{Name: clean})
	if err != nil {
		return nil, err
	}

	exact := make([]WANAccelerator, 0)
	for _, accel := range accelerators {
		if strings.EqualFold(accel.Name, clean) {
			exact = append(exact, accel)
		}
	}

	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		if len(accelerators) == 0 {
			return nil, fmt.Errorf("wan accelerator named %q not found", clean)
		}

		suggestions := make([]string, 0, len(accelerators))
		for _, accel := range accelerators {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", accel.Name, accel.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("wan accelerator named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		candidates := make([]string, 0, len(exact))
		for _, accel := range exact {
			candidates = append(candidates, accel.ID)
			if len(candidates) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple wan accelerators named %q found (ids: %s)", clean, strings.Join(candidates, ", "))
	}
}

func decodeWANAccelerator(raw map[string]any) (WANAccelerator, error) {
	accel := WANAccelerator{
		Raw: raw,
	}

	if v, ok := raw["id"].(string); ok {
		accel.ID = v
	}
	if v, ok := raw["name"].(string); ok {
		accel.Name = v
	}

	if serverRaw, ok := raw["server"].(map[string]any); ok {
		accel.Server = &WANAcceleratorServer{}
		if v, ok := serverRaw["hostId"].(string); ok {
			accel.Server.HostID = v
		}
		if v, ok := serverRaw["description"].(string); ok {
			accel.Server.Description = v
		}
		if v, ok := toInt(serverRaw["trafficPort"]); ok {
			accel.Server.TrafficPort = v
		}
		if v, ok := toInt(serverRaw["streamsCount"]); ok {
			accel.Server.StreamsCount = v
		}
		if v, ok := serverRaw["highBandwidthModeEnabled"].(bool); ok {
			accel.Server.HighBandwidthModeEnabled = v
		}
	}

	if cacheRaw, ok := raw["cache"].(map[string]any); ok {
		accel.Cache = &WANAcceleratorCache{}
		if v, ok := cacheRaw["cacheFolder"].(string); ok {
			accel.Cache.CacheFolder = v
		}
		if v, ok := toInt(cacheRaw["cacheSize"]); ok {
			accel.Cache.CacheSize = v
		}
		if v, ok := cacheRaw["cacheSizeUnit"].(string); ok {
			accel.Cache.CacheSizeUnit = v
		}
	}

	return accel, nil
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}
