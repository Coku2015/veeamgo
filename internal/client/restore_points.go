package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RestorePointsFilter struct {
	Name           string
	BackupID       string
	BackupObjectID string
	PlatformName   string
	PlatformID     string
	MalwareStatus  string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	MaxItems       int
	OrderColumn    string
	OrderAscending *bool
}

type restorePointsResponse struct {
	Data       []json.RawMessage `json:"data"`
	Pagination paginationResult  `json:"pagination"`
}

type RestorePoint struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	PlatformName  string    `json:"platformName"`
	PlatformID    string    `json:"platformId"`
	CreationTime  time.Time `json:"creationTime"`
	BackupID      string    `json:"backupId"`
	Type          string    `json:"type"`
	SessionID     string    `json:"sessionId"`
	AllowedOps    []string  `json:"allowedOperations"`
	MalwareStatus string    `json:"malwareStatus"`
	BackupFileID  string    `json:"backupFileId"`
	GuestOSFamily string    `json:"guestOsFamily"`
	OriginalSize  int64     `json:"originalSize"`
}

type RestorePointsResult struct {
	RestorePoints []RestorePoint
	Pagination    paginationResult
	Raw           map[string]any
}

func (c *Client) RestorePoints(ctx context.Context, filter RestorePointsFilter) (*RestorePointsResult, error) {
	results := make([]RestorePoint, 0)
	rawItems := make([]map[string]any, 0)
	var pagination paginationResult
	skip := 0
	remaining := filter.MaxItems

	seen := make(map[string]struct{})

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
		if filter.BackupID != "" {
			query.Set("backupIdFilter", filter.BackupID)
		}
		if filter.BackupObjectID != "" {
			query.Set("backupObjectIdFilter", filter.BackupObjectID)
		}
		if filter.PlatformName != "" {
			query.Set("platformNameFilter", filter.PlatformName)
		}
		if filter.PlatformID != "" {
			query.Set("platformIdFilter", filter.PlatformID)
		}
		if filter.MalwareStatus != "" {
			query.Set("malwareStatusFilter", filter.MalwareStatus)
		}
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp restorePointsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/restorePoints", query, &resp); err != nil {
			return nil, err
		}
		pagination = resp.Pagination
		for _, entry := range resp.Data {
			var rp RestorePoint
			raw, err := decodeRawMessage(entry, &rp)
			if err != nil {
				return nil, fmt.Errorf("decode restore point payload: %w", err)
			}
			key := strings.TrimSpace(rp.ID)
			if key == "" {
				key = strings.TrimSpace(rp.SessionID)
			}
			if key == "" {
				key = fmt.Sprintf("%s|%s|%s", strings.TrimSpace(rp.Name), strings.TrimSpace(rp.Type), rp.CreationTime.UTC().Format(time.RFC3339Nano))
			}
			normalized := strings.ToLower(key)
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			results = append(results, rp)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
				return &RestorePointsResult{
					RestorePoints: results[:filter.MaxItems],
					Pagination:    resp.Pagination,
					Raw: map[string]any{
						"data":       rawItems[:filter.MaxItems],
						"pagination": resp.Pagination,
					},
				}, nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count
	}

	return &RestorePointsResult{
		RestorePoints: results,
		Pagination:    pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

func (c *Client) RestorePoint(ctx context.Context, id string) (*RestorePoint, error) {
	path := fmt.Sprintf("/api/v1/restorePoints/%s", id)
	var rp RestorePoint
	if err := c.getJSON(ctx, path, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}

func (c *Client) RestorePointDetail(ctx context.Context, id string) (*RestorePointDetail, error) {
	path := fmt.Sprintf("/api/v1/restorePoints/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode restore point payload: %w", err)
	}
	var rp RestorePoint
	if err := json.Unmarshal(raw, &rp); err != nil {
		return nil, fmt.Errorf("decode restore point payload: %w", err)
	}
	return &RestorePointDetail{
		RestorePoint: rp,
		Raw:          payload,
	}, nil
}

type RestorePointDetail struct {
	RestorePoint RestorePoint
	Raw          map[string]any
}

func (c *Client) RestorePointByName(ctx context.Context, name string, filter RestorePointsFilter) (*RestorePoint, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("restore point name cannot be empty")
	}
	filter.Name = clean
	result, err := c.RestorePoints(ctx, filter)
	if err != nil {
		return nil, err
	}
	rps := result.RestorePoints

	exact := make([]RestorePoint, 0)
	for _, rp := range rps {
		if strings.EqualFold(rp.Name, clean) {
			exact = append(exact, rp)
		}
	}

	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		if len(rps) == 0 {
			return nil, fmt.Errorf("restore point named %q not found", clean)
		}
		suggestions := make([]string, 0, len(rps))
		for _, rp := range rps {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", rp.Name, rp.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("restore point named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		ids := make([]string, 0, len(exact))
		for _, rp := range exact {
			ids = append(ids, rp.ID)
			if len(ids) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple restore points named %q found (ids: %s)", clean, strings.Join(ids, ", "))
	}
}

type RestorePointDisksResponse struct {
	Data       []RestorePointDisk `json:"data"`
	Pagination paginationResult   `json:"pagination"`
}

type RestorePointDisk struct {
	UID      string `json:"uid"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Capacity int64  `json:"capacity"`
	State    string `json:"state"`
	Platform string `json:"platform"`
}

func (c *Client) RestorePointDisks(ctx context.Context, id string) ([]RestorePointDisk, error) {
	path := fmt.Sprintf("/api/v1/restorePoints/%s/disks", id)
	var resp RestorePointDisksResponse
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
