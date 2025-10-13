package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultPageSize = 200

type paginationResult struct {
	Total int `json:"total"`
	Count int `json:"count"`
	Skip  int `json:"skip"`
	Limit int `json:"limit"`
}

type ManagedServersFilter struct {
	Name     string
	Types    []string
	MaxItems int
}

type ManagedServer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
}

type managedServersResponse struct {
	Data       []ManagedServer  `json:"data"`
	Pagination paginationResult `json:"pagination"`
}

func (c *Client) ManagedServers(ctx context.Context, filter ManagedServersFilter) ([]ManagedServer, error) {
	results := make([]ManagedServer, 0)
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

		var resp managedServersResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/managedServers", query, &resp); err != nil {
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

func (c *Client) ManagedServer(ctx context.Context, id string) (*ManagedServer, error) {
	path := fmt.Sprintf("/api/v1/backupInfrastructure/managedServers/%s", id)
	var payload ManagedServer
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (c *Client) RescanAllManagedServers(ctx context.Context) (*Session, error) {
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/backupInfrastructure/managedServers/rescan", nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *Client) RescanManagedServer(ctx context.Context, id string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/backupInfrastructure/managedServers/%s/rescan", id)
	var sess Session
	if err := c.postJSON(ctx, path, nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

type RepositoryStatesFilter struct {
	Name     string
	Types    []string
	ID       string
	MaxItems int
}

type RepositoryState struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Description   string  `json:"description"`
	HostID        string  `json:"hostId"`
	HostName      string  `json:"hostName"`
	Path          string  `json:"path"`
	CapacityGB    float64 `json:"capacityGB"`
	FreeGB        float64 `json:"freeGB"`
	UsedSpaceGB   float64 `json:"usedSpaceGB"`
	IsOnline      bool    `json:"isOnline"`
	IsOutOfDate   bool    `json:"isOutOfDate"`
	ScaleOutState any     `json:"scaleOutRepositoryDetails,omitempty"`
}

type repositoryStatesResponse struct {
	Data       []RepositoryState `json:"data"`
	Pagination paginationResult  `json:"pagination"`
}

func (c *Client) RepositoryStates(ctx context.Context, filter RepositoryStatesFilter) ([]RepositoryState, error) {
	results := make([]RepositoryState, 0)
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
		if filter.ID != "" {
			query.Set("idFilter", filter.ID)
		}
		for _, t := range filter.Types {
			if t != "" {
				query.Add("typeFilter", t)
			}
		}

		var resp repositoryStatesResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/repositories/states", query, &resp); err != nil {
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

func (c *Client) RepositoryState(ctx context.Context, id string) (*RepositoryState, error) {
	states, err := c.RepositoryStates(ctx, RepositoryStatesFilter{ID: id, MaxItems: 1})
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, fmt.Errorf("repository %s not found", id)
	}
	return &states[0], nil
}

func (c *Client) RepositoryStateByName(ctx context.Context, name string) (*RepositoryState, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	states, err := c.RepositoryStates(ctx, RepositoryStatesFilter{Name: clean})
	if err != nil {
		return nil, err
	}

	exactMatches := make([]RepositoryState, 0)
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
			return nil, fmt.Errorf("repository named %q not found", clean)
		}

		suggestions := make([]string, 0, len(states))
		for _, st := range states {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", st.Name, st.ID))
			if len(suggestions) == 5 {
				break
			}
		}

		return nil, fmt.Errorf("repository named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		dups := make([]string, 0, len(exactMatches))
		for _, st := range exactMatches {
			dups = append(dups, st.ID)
			if len(dups) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple repositories named %q found (ids: %s)", clean, strings.Join(dups, ", "))
	}
}

func (c *Client) Repository(ctx context.Context, id string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v1/backupInfrastructure/repositories/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) RescanRepositories(ctx context.Context, ids []string) (*Session, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no repository IDs provided")
	}
	payload := map[string]any{
		"repositoryIds": ids,
	}
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/backupInfrastructure/repositories/rescan", nil, payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

type Session struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	SessionType       string         `json:"sessionType"`
	JobID             string         `json:"jobId"`
	State             string         `json:"state"`
	ProgressPercent   int            `json:"progressPercent"`
	Result            *SessionResult `json:"result,omitempty"`
	CreationTime      time.Time      `json:"creationTime"`
	EndTime           *time.Time     `json:"endTime,omitempty"`
	ResourceID        string         `json:"resourceId"`
	ResourceReference string         `json:"resourceReference"`
	ParentSessionID   string         `json:"parentSessionId"`
	USN               int64          `json:"usn"`
	PlatformName      string         `json:"platformName"`
	PlatformID        string         `json:"platformId"`
	InitiatedBy       string         `json:"initiatedBy"`
}

type SessionResult struct {
	Result     string `json:"result"`
	Message    string `json:"message,omitempty"`
	IsCanceled bool   `json:"isCanceled,omitempty"`
}

func (c *Client) Session(ctx context.Context, id string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s", id)
	var sess Session
	if err := c.getJSON(ctx, path, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *Client) WaitForSession(ctx context.Context, id string, pollInterval time.Duration) (*Session, error) {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		session, err := c.Session(ctx, id)
		if err != nil {
			return nil, err
		}
		if !session.inProgress() {
			return session, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Session) inProgress() bool {
	switch s.State {
	case "Starting", "Working", "Stopping", "Pausing", "Resuming", "WaitingTape", "WaitingRepository", "WaitingSlot", "Postprocessing":
		return true
	default:
		return false
	}
}
