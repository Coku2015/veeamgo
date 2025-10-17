package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type SessionsFilter struct {
	Name           string
	Types          []string
	States         []string
	Results        []string
	JobID          string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	EndedAfter     *time.Time
	EndedBefore    *time.Time
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type SessionLogsFilter struct {
	Status string
}

type sessionResponse struct {
	Data       []Session        `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

type SessionLogResult struct {
	TotalRecords int               `json:"totalRecords"`
	Records      []SessionLogEntry `json:"records"`
}

type SessionLogEntry struct {
	ID             int        `json:"id"`
	Status         string     `json:"status"`
	StartTime      *time.Time `json:"startTime,omitempty"`
	UpdateTime     *time.Time `json:"updateTime,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	AdditionalInfo string     `json:"additionalInfo"`
}

func (c *Client) Sessions(ctx context.Context, filter SessionsFilter) ([]Session, error) {
	results := make([]Session, 0)
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
		for _, t := range filter.Types {
			if t != "" {
				query.Add("typeFilter", t)
			}
		}
		for _, s := range filter.States {
			if s != "" {
				query.Add("stateFilter", s)
			}
		}
		for _, r := range filter.Results {
			if r != "" {
				query.Add("resultFilter", r)
			}
		}
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.EndedAfter != nil && !filter.EndedAfter.IsZero() {
			query.Set("endedAfterFilter", filter.EndedAfter.Format(time.RFC3339))
		}
		if filter.EndedBefore != nil && !filter.EndedBefore.IsZero() {
			query.Set("endedBeforeFilter", filter.EndedBefore.Format(time.RFC3339))
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp sessionResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/sessions", query, &resp); err != nil {
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

type SessionDetail struct {
	Session Session
	Raw     map[string]any
}

func (c *Client) SessionDetail(ctx context.Context, id string) (*SessionDetail, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	rawBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode session payload: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(rawBytes, &sess); err != nil {
		return nil, fmt.Errorf("decode session payload: %w", err)
	}

	return &SessionDetail{
		Session: sess,
		Raw:     payload,
	}, nil
}

func (c *Client) SessionLogs(ctx context.Context, id string, filter SessionLogsFilter) ([]SessionLogEntry, error) {
	query := url.Values{}
	if filter.Status != "" {
		query.Set("statusFilter", filter.Status)
	}

	var result SessionLogResult
	path := fmt.Sprintf("/api/v1/sessions/%s/logs", id)
	if err := c.getJSONWithQuery(ctx, path, query, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

type TaskSessionsFilter struct {
	Name           string
	SessionID      string
	Type           string
	SessionType    string
	State          string
	Result         string
	ScanType       string
	ScanResult     string
	ScanState      string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	EndedAfter     *time.Time
	EndedBefore    *time.Time
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

type taskSessionsResponse struct {
	Data       []TaskSessionSummary `json:"data"`
	Pagination paginationResult     `json:"pagination"`
}

type TaskSessionSummary struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	SessionID    string         `json:"sessionId"`
	SessionType  string         `json:"sessionType"`
	State        string         `json:"state"`
	Result       *SessionResult `json:"result,omitempty"`
	Progress     *ProgressInfo  `json:"progress,omitempty"`
	CreationTime APITime        `json:"creationTime"`
	EndTime      *APITime       `json:"endTime,omitempty"`
}

type TaskSessionDetail struct {
	Summary TaskSessionSummary
	Raw     map[string]any
}

func (c *Client) TaskSessions(ctx context.Context, filter TaskSessionsFilter) ([]TaskSessionSummary, error) {
	results := make([]TaskSessionSummary, 0)
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
		if filter.SessionID != "" {
			query.Set("sessionIdFilter", filter.SessionID)
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.SessionType != "" {
			query.Set("sessionTypeFilter", filter.SessionType)
		}
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}
		if filter.Result != "" {
			query.Set("resultFilter", filter.Result)
		}
		if filter.ScanType != "" {
			query.Set("scanTypeFilter", filter.ScanType)
		}
		if filter.ScanResult != "" {
			query.Set("scanResultFilter", filter.ScanResult)
		}
		if filter.ScanState != "" {
			query.Set("scanStateFilter", filter.ScanState)
		}
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.EndedAfter != nil && !filter.EndedAfter.IsZero() {
			query.Set("endedAfterFilter", filter.EndedAfter.Format(time.RFC3339))
		}
		if filter.EndedBefore != nil && !filter.EndedBefore.IsZero() {
			query.Set("endedBeforeFilter", filter.EndedBefore.Format(time.RFC3339))
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
			if filter.OrderAscending != nil {
				query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
			}
		}

		var resp taskSessionsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/taskSessions", query, &resp); err != nil {
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

func (c *Client) TaskSessionDetail(ctx context.Context, id string) (*TaskSessionDetail, error) {
	path := fmt.Sprintf("/api/v1/taskSessions/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	rawBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode task session payload: %w", err)
	}

	var summary TaskSessionSummary
	if err := json.Unmarshal(rawBytes, &summary); err != nil {
		return nil, fmt.Errorf("decode task session payload: %w", err)
	}

	return &TaskSessionDetail{
		Summary: summary,
		Raw:     payload,
	}, nil
}

func (c *Client) TaskSessionLogs(ctx context.Context, id string, filter SessionLogsFilter) ([]SessionLogEntry, error) {
	query := url.Values{}
	if filter.Status != "" {
		query.Set("statusFilter", filter.Status)
	}

	var result SessionLogResult
	path := fmt.Sprintf("/api/v1/taskSessions/%s/logs", id)
	if err := c.getJSONWithQuery(ctx, path, query, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}
