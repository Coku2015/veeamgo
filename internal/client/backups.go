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

type BackupsFilter struct {
	Name           string
	JobID          string
	JobType        string
	PolicyTag      string
	PlatformID     string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	MaxItems       int
	OrderColumn    string
	OrderAscending *bool
}

type backupsResponse struct {
	Data       []Backup        `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type Backup struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	JobID         string    `json:"jobId"`
	JobType       string    `json:"jobType"`
	PolicyUniqueID string   `json:"policyUniqueId"`
	PlatformName  string    `json:"platformName"`
	PlatformID    string    `json:"platformId"`
	RepositoryID  string    `json:"repositoryId"`
	RepositoryName string   `json:"repositoryName"`
	CreationTime  time.Time `json:"creationTime"`
}

func (c *Client) Backups(ctx context.Context, filter BackupsFilter) ([]Backup, error) {
	results := make([]Backup, 0)
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
		if filter.JobType != "" {
			query.Set("jobTypeFilter", filter.JobType)
		}
		if filter.PolicyTag != "" {
			query.Set("policyTagFilter", filter.PolicyTag)
		}
		if filter.PlatformID != "" {
			query.Set("platformIdFilter", filter.PlatformID)
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

		var resp backupsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/backups", query, &resp); err != nil {
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

func (c *Client) Backup(ctx context.Context, id string) (*Backup, error) {
	path := fmt.Sprintf("/api/v1/backups/%s", id)
	var backup Backup
	if err := c.getJSON(ctx, path, &backup); err != nil {
		return nil, err
	}
	return &backup, nil
}

func (c *Client) BackupDetail(ctx context.Context, id string) (*BackupDetail, error) {
	path := fmt.Sprintf("/api/v1/backups/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode backup payload: %w", err)
	}

	var backup Backup
	if err := json.Unmarshal(raw, &backup); err != nil {
		return nil, fmt.Errorf("decode backup payload: %w", err)
	}

	return &BackupDetail{
		Backup: backup,
		Raw:    payload,
	}, nil
}

type BackupDetail struct {
	Backup Backup
	Raw    map[string]any
}

func (c *Client) BackupByName(ctx context.Context, name string) (*Backup, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("backup name cannot be empty")
	}

	backups, err := c.Backups(ctx, BackupsFilter{Name: clean})
	if err != nil {
		return nil, err
	}

	exact := make([]Backup, 0)
	for _, b := range backups {
		if strings.EqualFold(b.Name, clean) {
			exact = append(exact, b)
		}
	}

	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		if len(backups) == 0 {
			return nil, fmt.Errorf("backup named %q not found", clean)
		}
		suggestions := make([]string, 0, len(backups))
		for _, b := range backups {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", b.Name, b.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("backup named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		ids := make([]string, 0, len(exact))
		for _, b := range exact {
			ids = append(ids, b.ID)
			if len(ids) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple backups named %q found (ids: %s)", clean, strings.Join(ids, ", "))
	}
}

type BackupFilesFilter struct {
	Name          string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	GFSPeroid     string
	OrderColumn   string
	OrderAsc      *bool
	MaxItems      int
}

type backupFilesResponse struct {
	Data       []BackupFile     `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type BackupFile struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	BackupID       string     `json:"backupId"`
	ObjectID       string     `json:"objectId"`
	RestorePointIDs []string  `json:"restorePointIds"`
	DataSize       int64      `json:"dataSize"`
	BackupSize     int64      `json:"backupSize"`
	DedupRatio     int        `json:"dedupRatio"`
	CompressRatio  int        `json:"compressRatio"`
	CreationTime   time.Time  `json:"creationTime"`
	GFSPeroids     []string   `json:"gfsPeriods"`
}

func (c *Client) BackupFiles(ctx context.Context, backupID string, filter BackupFilesFilter) ([]BackupFile, error) {
	results := make([]BackupFile, 0)
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
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.GFSPeroid != "" {
			query.Set("gfsPeriodFilter", filter.GFSPeroid)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAsc != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAsc))
			}
		}

		var resp backupFilesResponse
		path := fmt.Sprintf("/api/v1/backups/%s/backupFiles", backupID)
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
	}

	return results, nil
}

type BackupObject struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	PlatformName       string `json:"platformName"`
	PlatformID         string `json:"platformId"`
	RestorePointsCount int    `json:"restorePointsCount"`
	LastRunFailed      bool   `json:"lastRunFailed"`
}

func (c *Client) BackupObjects(ctx context.Context, backupID string) ([]BackupObject, error) {
	path := fmt.Sprintf("/api/v1/backups/%s/objects", backupID)
	var result struct {
		Data []BackupObject `json:"data"`
	}
	if err := c.getJSON(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
