package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type ReplicasFilter struct {
	Name           string
	JobID          string
	PolicyTag      string
	PlatformID     string
	State          string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	MaxItems       int
	OrderColumn    string
	OrderAscending *bool
}

type replicasResponse struct {
	Data       []Replica        `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type Replica struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	JobID          string    `json:"jobId"`
	PolicyUniqueID string    `json:"policyUniqueId"`
	PlatformName   string    `json:"platformName"`
	PlatformID     string    `json:"platformId"`
	CreationTime   time.Time `json:"creationTime"`
	State          string    `json:"state"`
}

func (c *Client) Replicas(ctx context.Context, filter ReplicasFilter) ([]Replica, error) {
	results := make([]Replica, 0)
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
		if filter.JobID != "" {
			query.Set("jobIdFilter", filter.JobID)
		}
		if filter.PolicyTag != "" {
			query.Set("policyTagFilter", filter.PolicyTag)
		}
		if filter.PlatformID != "" {
			query.Set("platformIdFilter", filter.PlatformID)
		}
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
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

		var resp replicasResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/replicas", query, &resp); err != nil {
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
	}

	return results, nil
}

func (c *Client) Replica(ctx context.Context, id string) (*Replica, error) {
	path := fmt.Sprintf("/api/v1/replicas/%s", id)
	var replica Replica
	if err := c.getJSON(ctx, path, &replica); err != nil {
		return nil, err
	}
	return &replica, nil
}

type ReplicaDetail struct {
	Replica Replica
	Raw     map[string]any
}

func (c *Client) ReplicaDetail(ctx context.Context, id string) (*ReplicaDetail, error) {
	path := fmt.Sprintf("/api/v1/replicas/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode replica payload: %w", err)
	}

	var replica Replica
	if err := json.Unmarshal(raw, &replica); err != nil {
		return nil, fmt.Errorf("decode replica payload: %w", err)
	}

	return &ReplicaDetail{
		Replica: replica,
		Raw:     payload,
	}, nil
}

type ReplicaPointsFilter struct {
	Name           string
	ReplicaID      string
	PlatformName   string
	PlatformID     string
	MalwareStatus  string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	MaxItems       int
	OrderColumn    string
	OrderAscending *bool
}

type replicaPointsResponse struct {
	Data       []json.RawMessage `json:"data"`
	Pagination paginationResult  `json:"pagination"`
}

type ReplicaPoint struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	PlatformName      string    `json:"platformName"`
	PlatformID        string    `json:"platformId"`
	CreationTime      time.Time `json:"creationTime"`
	ReplicaID         string    `json:"replicaId"`
	AllowedOperations []string  `json:"allowedOperations"`
	State             string    `json:"state"`
	MalwareStatus     string    `json:"malwareStatus"`
}

type ReplicaPointsResult struct {
	ReplicaPoints []ReplicaPoint
	Pagination    paginationResult
	Raw           map[string]any
}

func (c *Client) ReplicaPoints(ctx context.Context, filter ReplicaPointsFilter) (*ReplicaPointsResult, error) {
	results := make([]ReplicaPoint, 0)
	rawItems := make([]map[string]any, 0)
	skip := 0
	remaining := filter.MaxItems
	var pagination paginationResult

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
		if filter.ReplicaID != "" {
			query.Set("replicaIdFilter", filter.ReplicaID)
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

		var resp replicaPointsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/replicaPoints", query, &resp); err != nil {
			return nil, err
		}
		pagination = resp.Pagination

		for _, entry := range resp.Data {
			var point ReplicaPoint
			raw, err := decodeRawMessage(entry, &point)
			if err != nil {
				return nil, fmt.Errorf("decode replica point payload: %w", err)
			}
			results = append(results, point)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
				return &ReplicaPointsResult{
					ReplicaPoints: results[:filter.MaxItems],
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

	return &ReplicaPointsResult{
		ReplicaPoints: results,
		Pagination:    pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

func (c *Client) ReplicaPoint(ctx context.Context, id string) (*ReplicaPoint, error) {
	path := fmt.Sprintf("/api/v1/replicaPoints/%s", id)
	var point ReplicaPoint
	if err := c.getJSON(ctx, path, &point); err != nil {
		return nil, err
	}
	return &point, nil
}

type ReplicaPointDetail struct {
	Point ReplicaPoint
	Raw   map[string]any
}

func (c *Client) ReplicaPointDetail(ctx context.Context, id string) (*ReplicaPointDetail, error) {
	path := fmt.Sprintf("/api/v1/replicaPoints/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode replica point payload: %w", err)
	}

	var point ReplicaPoint
	if err := json.Unmarshal(raw, &point); err != nil {
		return nil, fmt.Errorf("decode replica point payload: %w", err)
	}

	return &ReplicaPointDetail{
		Point: point,
		Raw:   payload,
	}, nil
}
