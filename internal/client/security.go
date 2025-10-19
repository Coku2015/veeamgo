package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// SecurityAnalyzerRun represents the most recent analyzer execution.
type SecurityAnalyzerRun struct {
	Session Session
	Raw     map[string]any
}

// SecurityAnalyzerSchedule captures the scheduled run configuration.
type SecurityAnalyzerSchedule struct {
	Settings SecurityAnalyzerScheduleSettings
	Raw      map[string]any
}

// SecurityAnalyzerScheduleSettings mirrors the REST schedule schema.
type SecurityAnalyzerScheduleSettings struct {
	DailyScanEnabled         bool                               `json:"dailyScanEnabled"`
	DailyScanLocalTime       string                             `json:"dailyScanLocalTime"`
	SendScanResults          bool                               `json:"sendScanResults"`
	Recipients               string                             `json:"recipients"`
	NotificationType         string                             `json:"notificationType"`
	CustomNotificationConfig *SecurityAnalyzerEmailNotification `json:"customNotificationSettings,omitempty"`
}

// SecurityAnalyzerEmailNotification represents the custom email notification payload.
type SecurityAnalyzerEmailNotification struct {
	Subject         string `json:"subject"`
	NotifyOnSuccess bool   `json:"notifyOnSuccess"`
	NotifyOnWarning bool   `json:"notifyOnWarning"`
	NotifyOnError   bool   `json:"notifyOnError"`
}

// SecurityBestPractice reflects a best-practice entry with raw payload.
type SecurityBestPractice struct {
	ID           string         `json:"id"`
	BestPractice string         `json:"bestPractice"`
	Status       string         `json:"status"`
	Note         string         `json:"note,omitempty"`
	Raw          map[string]any `json:"-"`
}

// SecurityAnalyzerBestPracticesResult aggregates best-practice records.
type SecurityAnalyzerBestPracticesResult struct {
	Items []SecurityBestPractice
	Raw   map[string]any
}

// AuthorizationEventsFilter captures list query options.
type AuthorizationEventsFilter struct {
	Name            string
	State           string
	CreatedBy       string
	ProcessedBy     string
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	ProcessedAfter  *time.Time
	ProcessedBefore *time.Time
	ExpireAfter     *time.Time
	ExpireBefore    *time.Time
	OrderColumn     string
	OrderAscending  *bool
	MaxItems        int
}

// AuthorizationEvent models an authorization/audit event.
type AuthorizationEvent struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	State          string         `json:"state"`
	CreatedBy      string         `json:"createdBy"`
	ProcessedBy    string         `json:"processedBy"`
	CreationTime   APITime        `json:"creationTime"`
	ProcessedTime  *APITime       `json:"processedTime,omitempty"`
	ExpirationTime *APITime       `json:"expirationTime,omitempty"`
	Raw            map[string]any `json:"-"`
}

// AuthorizationEventsResult returns both typed and raw responses.
type AuthorizationEventsResult struct {
	Events     []AuthorizationEvent
	Pagination paginationResult
	Raw        map[string]any
}

// AuthorizationEventDetail carries a single event with raw JSON.
type AuthorizationEventDetail struct {
	Event AuthorizationEvent
	Raw   map[string]any
}

// SuspiciousActivityFilter captures malware event filters.
type SuspiciousActivityFilter struct {
	Type           string
	State          string
	Source         string
	Severity       string
	CreatedBy      string
	Engine         string
	MachineName    string
	BackupObjectID string
	DetectedAfter  *time.Time
	DetectedBefore *time.Time
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

// SuspiciousActivityEvent models a malware detection event.
type SuspiciousActivityEvent struct {
	ID            string                     `json:"id"`
	Type          string                     `json:"type"`
	State         string                     `json:"state"`
	Source        string                     `json:"source"`
	Severity      string                     `json:"severity"`
	Details       string                     `json:"details"`
	Engine        string                     `json:"engine"`
	CreatedBy     string                     `json:"createdBy"`
	CreationTime  APITime                    `json:"creationTimeUtc"`
	DetectionTime APITime                    `json:"detectionTimeUtc"`
	Machine       *SuspiciousActivityMachine `json:"machine,omitempty"`
	Raw           map[string]any             `json:"-"`
}

// SuspiciousActivityMachine stores affected workload details.
type SuspiciousActivityMachine struct {
	DisplayName    string `json:"displayName"`
	UUID           string `json:"uuid"`
	BackupObjectID string `json:"backupObjectId"`
	RestorePointID string `json:"restorePointId"`
}

// SuspiciousActivityEventsResult aggregates malware events.
type SuspiciousActivityEventsResult struct {
	Events     []SuspiciousActivityEvent
	Pagination paginationResult
	Raw        map[string]any
}

// SuspiciousActivityEventDetail exposes a single malware event.
type SuspiciousActivityEventDetail struct {
	Event SuspiciousActivityEvent
	Raw   map[string]any
}

// YaraRule represents a custom YARA rule entry.
type YaraRule struct {
	FileName string         `json:"fileName"`
	Raw      map[string]any `json:"-"`
}

// YaraRulesResult contains available YARA rules.
type YaraRulesResult struct {
	Rules      []YaraRule
	Pagination paginationResult
	Raw        map[string]any
}

// SecurityAnalyzerLastRun retrieves the most recent analyzer session.
func (c *Client) SecurityAnalyzerLastRun(ctx context.Context) (*SecurityAnalyzerRun, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/securityAnalyzer/lastRun", &payload); err != nil {
		return nil, err
	}

	var session Session
	if _, err := decodeObject(payload, &session); err != nil {
		return nil, fmt.Errorf("decode security analyzer session: %w", err)
	}

	return &SecurityAnalyzerRun{
		Session: session,
		Raw:     payload,
	}, nil
}

// SecurityAnalyzerSchedule retrieves the scheduler configuration.
func (c *Client) SecurityAnalyzerSchedule(ctx context.Context) (*SecurityAnalyzerSchedule, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/securityAnalyzer/schedule", &payload); err != nil {
		return nil, err
	}

	var settings SecurityAnalyzerScheduleSettings
	if _, err := decodeObject(payload, &settings); err != nil {
		return nil, fmt.Errorf("decode security analyzer schedule: %w", err)
	}

	return &SecurityAnalyzerSchedule{
		Settings: settings,
		Raw:      payload,
	}, nil
}

// SecurityAnalyzerBestPractices returns the compliance checklist.
func (c *Client) SecurityAnalyzerBestPractices(ctx context.Context) (*SecurityAnalyzerBestPracticesResult, error) {
	var resp struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := c.getJSON(ctx, "/api/v1/securityAnalyzer/bestPractices", &resp); err != nil {
		return nil, err
	}

	rawItems := make([]map[string]any, 0, len(resp.Items))
	practices := make([]SecurityBestPractice, 0, len(resp.Items))

	for _, entry := range resp.Items {
		var practice SecurityBestPractice
		raw, err := decodeRawMessage(entry, &practice)
		if err != nil {
			return nil, fmt.Errorf("decode best practice: %w", err)
		}
		practice.Raw = raw
		practices = append(practices, practice)
		rawItems = append(rawItems, raw)
	}

	return &SecurityAnalyzerBestPracticesResult{
		Items: practices,
		Raw: map[string]any{
			"items": rawItems,
		},
	}, nil
}

// StartSecurityAnalyzer triggers a new Security & Compliance Analyzer run.
func (c *Client) StartSecurityAnalyzer(ctx context.Context) (*Session, error) {
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/securityAnalyzer/start", nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// AuthorizationEvents retrieves authorization events with optional filters.
func (c *Client) AuthorizationEvents(ctx context.Context, filter AuthorizationEventsFilter) (*AuthorizationEventsResult, error) {
	results := make([]AuthorizationEvent, 0)
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
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}
		if filter.CreatedBy != "" {
			query.Set("createdByFilter", filter.CreatedBy)
		}
		if filter.ProcessedBy != "" {
			query.Set("processedByFilter", filter.ProcessedBy)
		}
		if filter.CreatedAfter != nil && !filter.CreatedAfter.IsZero() {
			query.Set("createdAfterFilter", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil && !filter.CreatedBefore.IsZero() {
			query.Set("createdBeforeFilter", filter.CreatedBefore.Format(time.RFC3339))
		}
		if filter.ProcessedAfter != nil && !filter.ProcessedAfter.IsZero() {
			query.Set("processedAfterFilter", filter.ProcessedAfter.Format(time.RFC3339))
		}
		if filter.ProcessedBefore != nil && !filter.ProcessedBefore.IsZero() {
			query.Set("processedBeforeFilter", filter.ProcessedBefore.Format(time.RFC3339))
		}
		if filter.ExpireAfter != nil && !filter.ExpireAfter.IsZero() {
			query.Set("expireAfterFilter", filter.ExpireAfter.Format(time.RFC3339))
		}
		if filter.ExpireBefore != nil && !filter.ExpireBefore.IsZero() {
			query.Set("expireBeforeFilter", filter.ExpireBefore.Format(time.RFC3339))
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

		if err := c.getJSONWithQuery(ctx, "/api/v1/authorization/events", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var event AuthorizationEvent
			raw, err := decodeRawMessage(entry, &event)
			if err != nil {
				return nil, fmt.Errorf("decode authorization event: %w", err)
			}
			event.Raw = raw
			results = append(results, event)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
				pagination = resp.Pagination
				return &AuthorizationEventsResult{
					Events:     results[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return &AuthorizationEventsResult{
		Events:     results,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// AuthorizationEventDetail fetches a single authorization event.
func (c *Client) AuthorizationEventDetail(ctx context.Context, id string) (*AuthorizationEventDetail, error) {
	payload := make(map[string]any)
	path := fmt.Sprintf("/api/v1/authorization/events/%s", id)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	var event AuthorizationEvent
	if _, err := decodeObject(payload, &event); err != nil {
		return nil, fmt.Errorf("decode authorization event detail: %w", err)
	}
	event.Raw = payload

	return &AuthorizationEventDetail{
		Event: event,
		Raw:   payload,
	}, nil
}

// SuspiciousActivityEvents lists malware detection events.
func (c *Client) SuspiciousActivityEvents(ctx context.Context, filter SuspiciousActivityFilter) (*SuspiciousActivityEventsResult, error) {
	events := make([]SuspiciousActivityEvent, 0)
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
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}
		if filter.Source != "" {
			query.Set("sourceFilter", filter.Source)
		}
		if filter.Severity != "" {
			query.Set("severityFilter", filter.Severity)
		}
		if filter.CreatedBy != "" {
			query.Set("createdByFilter", filter.CreatedBy)
		}
		if filter.Engine != "" {
			query.Set("engineFilter", filter.Engine)
		}
		if filter.MachineName != "" {
			query.Set("machineNameFilter", filter.MachineName)
		}
		if filter.BackupObjectID != "" {
			query.Set("backupObjectIdFilter", filter.BackupObjectID)
		}
		if filter.DetectedAfter != nil && !filter.DetectedAfter.IsZero() {
			query.Set("detectedAfterTimeUtcFilter", filter.DetectedAfter.Format(time.RFC3339))
		}
		if filter.DetectedBefore != nil && !filter.DetectedBefore.IsZero() {
			query.Set("detectedBeforeTimeUtcFilter", filter.DetectedBefore.Format(time.RFC3339))
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

		if err := c.getJSONWithQuery(ctx, "/api/v1/malwareDetection/events", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var event SuspiciousActivityEvent
			raw, err := decodeRawMessage(entry, &event)
			if err != nil {
				return nil, fmt.Errorf("decode suspicious activity event: %w", err)
			}
			event.Raw = raw
			events = append(events, event)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(events) >= filter.MaxItems {
				pagination = resp.Pagination
				return &SuspiciousActivityEventsResult{
					Events:     events[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(events)
			if remaining <= 0 {
				break
			}
		}
	}

	return &SuspiciousActivityEventsResult{
		Events:     events,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// SuspiciousActivityEventDetail retrieves an individual malware event.
func (c *Client) SuspiciousActivityEventDetail(ctx context.Context, id string) (*SuspiciousActivityEventDetail, error) {
	payload := make(map[string]any)
	path := fmt.Sprintf("/api/v1/malwareDetection/events/%s", id)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, err
	}

	var event SuspiciousActivityEvent
	if _, err := decodeObject(payload, &event); err != nil {
		return nil, fmt.Errorf("decode suspicious activity event detail: %w", err)
	}
	event.Raw = payload

	return &SuspiciousActivityEventDetail{
		Event: event,
		Raw:   payload,
	}, nil
}

// YaraRules lists uploaded YARA rule files.
func (c *Client) YaraRules(ctx context.Context) (*YaraRulesResult, error) {
	var resp struct {
		Data       []json.RawMessage `json:"data"`
		Pagination paginationResult  `json:"pagination"`
	}
	if err := c.getJSON(ctx, "/api/v1/malwareDetection/yaraRules", &resp); err != nil {
		return nil, err
	}

	rules := make([]YaraRule, 0, len(resp.Data))
	rawItems := make([]map[string]any, 0, len(resp.Data))

	for _, entry := range resp.Data {
		var rule YaraRule
		raw, err := decodeRawMessage(entry, &rule)
		if err != nil {
			return nil, fmt.Errorf("decode yara rule: %w", err)
		}
		rule.Raw = raw
		rules = append(rules, rule)
		rawItems = append(rawItems, raw)
	}

	return &YaraRulesResult{
		Rules:      rules,
		Pagination: resp.Pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": resp.Pagination,
		},
	}, nil
}
