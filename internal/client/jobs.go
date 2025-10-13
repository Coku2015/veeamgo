package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Job represents a lightweight job model returned from /api/v1/jobs.
type Job struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsDisabled bool   `json:"isDisabled"`
}

// JobsFilter provides optional filters for listing jobs.
type JobsFilter struct {
	Name     string
	Types    []string
	OrderBy  string
	OrderAsc bool
	MaxItems int
}

type jobsResponse struct {
	Data       []Job            `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// Jobs lists jobs with client-side pagination handling.
func (c *Client) Jobs(ctx context.Context, filter JobsFilter) ([]Job, error) {
	results := make([]Job, 0)
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
		for _, t := range filter.Types {
			if t != "" {
				query.Add("typeFilter", t)
			}
		}
		if filter.OrderBy != "" {
			query.Set("orderColumn", filter.OrderBy)
			query.Set("orderAsc", strconv.FormatBool(filter.OrderAsc))
		}

		var resp jobsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/jobs", query, &resp); err != nil {
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

// Job fetches detailed job configuration by ID.
func (c *Client) Job(ctx context.Context, id string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v1/jobs/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// JobState captures data returned from /api/v1/jobs/states.
type JobState struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Type            string        `json:"type"`
	Description     string        `json:"description"`
	Status          string        `json:"status"`
	LastRun         *time.Time    `json:"lastRun"`
	LastResult      string        `json:"lastResult"`
	NextRun         *time.Time    `json:"nextRun"`
	NextRunPolicy   string        `json:"nextRunPolicy"`
	Workload        string        `json:"workload"`
	RepositoryID    string        `json:"repositoryId"`
	RepositoryName  string        `json:"repositoryName"`
	ObjectsCount    int           `json:"objectsCount"`
	SessionID       string        `json:"sessionId"`
	HighPriority    bool          `json:"highPriority"`
	ProgressPercent int           `json:"progressPercent"`
	SessionProgress *ProgressInfo `json:"sessionProgress"`
	RunAfterJob     *RunAfterInfo `json:"runAfterJob"`
	BackupCopyMode  string        `json:"backupCopyMode"`
	IsStorageSnap   bool          `json:"isStorageSnapshot"`
}

// ProgressInfo describes the current session progress metrics.
type ProgressInfo struct {
	Duration        string `json:"duration"`
	ProcessingRate  string `json:"processingRate"`
	Bottleneck      string `json:"bottleneck"`
	ProcessedSize   *int64 `json:"processedSize"`
	ReadSize        *int64 `json:"readSize"`
	TransferredSize *int64 `json:"transferredSize"`
	ProgressPercent *int   `json:"progressPercent"`
}

// RunAfterInfo contains chaining metadata for a job.
type RunAfterInfo struct {
	JobName string `json:"jobName"`
	JobID   string `json:"jobId"`
}

// JobStatesFilter provides optional filters for /api/v1/jobs/states.
type JobStatesFilter struct {
	ID            string
	Name          string
	Types         []string
	Status        string
	LastResult    string
	Workload      string
	RepositoryID  string
	HighPriority  *bool
	LastRunAfter  *time.Time
	LastRunBefore *time.Time
	AfterJobID    string
	AfterJobName  string
	OrderBy       string
	OrderAsc      bool
	MaxItems      int
}

type jobStatesResponse struct {
	Data       []JobState       `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

// JobStates returns job state projections with pagination handling.
func (c *Client) JobStates(ctx context.Context, filter JobStatesFilter) ([]JobState, error) {
	results := make([]JobState, 0)
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
		if filter.ID != "" {
			query.Set("idFilter", filter.ID)
		}
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		for _, t := range filter.Types {
			if t != "" {
				query.Add("typeFilter", t)
			}
		}
		if filter.Status != "" {
			query.Set("statusFilter", filter.Status)
		}
		if filter.LastResult != "" {
			query.Set("lastResultFilter", filter.LastResult)
		}
		if filter.Workload != "" {
			query.Set("workloadFilter", filter.Workload)
		}
		if filter.RepositoryID != "" {
			query.Set("repositoryIdFilter", filter.RepositoryID)
		}
		if filter.HighPriority != nil {
			query.Set("isHighPriorityJobFilter", strconv.FormatBool(*filter.HighPriority))
		}
		if filter.LastRunAfter != nil && !filter.LastRunAfter.IsZero() {
			query.Set("lastRunAfterFilter", filter.LastRunAfter.Format(time.RFC3339))
		}
		if filter.LastRunBefore != nil && !filter.LastRunBefore.IsZero() {
			query.Set("lastRunBeforeFilter", filter.LastRunBefore.Format(time.RFC3339))
		}
		if filter.AfterJobID != "" {
			query.Set("afterJobIdFilter", filter.AfterJobID)
		}
		if filter.AfterJobName != "" {
			query.Set("afterJobNameFilter", filter.AfterJobName)
		}
		if filter.OrderBy != "" {
			query.Set("orderColumn", filter.OrderBy)
			query.Set("orderAsc", strconv.FormatBool(filter.OrderAsc))
		}

		var resp jobStatesResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/jobs/states", query, &resp); err != nil {
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

// JobStateByName resolves a job state by name, offering helpful errors on duplicates or missing entries.
func (c *Client) JobStateByName(ctx context.Context, name string) (*JobState, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("job name cannot be empty")
	}

	states, err := c.JobStates(ctx, JobStatesFilter{Name: clean})
	if err != nil {
		return nil, err
	}

	exactMatches := make([]JobState, 0)
	for _, st := range states {
		if strings.EqualFold(st.Name, clean) {
			exactMatches = append(exactMatches, st)
		}
	}

	switch len(exactMatches) {
	case 1:
		return &exactMatches[0], nil
	case 0:
		if len(states) == 0 {
			return nil, fmt.Errorf("job named %q not found", clean)
		}

		suggestions := make([]string, 0, len(states))
		for _, st := range states {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", st.Name, st.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("job named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		ids := make([]string, 0, len(exactMatches))
		for _, st := range exactMatches {
			ids = append(ids, st.ID)
			if len(ids) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple jobs named %q found (ids: %s)", clean, strings.Join(ids, ", "))
	}
}
