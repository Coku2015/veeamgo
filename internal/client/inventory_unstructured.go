package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// UnstructuredDataServersFilter defines query options for listing unstructured data servers.
type UnstructuredDataServersFilter struct {
	Skip           int
	Limit          int
	Name           string
	OrderColumn    string
	OrderAscending *bool
}

// UnstructuredDataServersResult mirrors the API response for unstructured data servers.
type UnstructuredDataServersResult struct {
	Data       []map[string]any `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// UnstructuredDataServers lists unstructured data servers using optional filters.
func (c *Client) UnstructuredDataServers(ctx context.Context, filter UnstructuredDataServersFilter) (*UnstructuredDataServersResult, error) {
	query := url.Values{}
	if filter.Skip > 0 {
		query.Set("skip", strconv.Itoa(filter.Skip))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
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

	var result UnstructuredDataServersResult
	if err := c.getJSONWithQuery(ctx, "/api/v1/inventory/unstructuredDataServers", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UnstructuredDataServer retrieves the details for a specific unstructured data server.
func (c *Client) UnstructuredDataServer(ctx context.Context, id string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v1/inventory/unstructuredDataServers/%s", url.PathEscape(id))
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
