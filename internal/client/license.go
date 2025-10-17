package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// InstalledLicense summarises the active license.
type InstalledLicense struct {
	Status                              string   `json:"status"`
	Edition                             string   `json:"edition"`
	Type                                string   `json:"type"`
	LicensedTo                          string   `json:"licensedTo"`
	SupportID                           string   `json:"supportId"`
	AutoUpdateEnabled                   bool     `json:"autoUpdateEnabled"`
	FreeAgentInstanceConsumptionEnabled bool     `json:"freeAgentInstanceConsumptionEnabled"`
	CloudConnect                        string   `json:"cloudConnect"`
	IsMultiSection                      bool     `json:"IsMultiSection"`
	ProactiveSupportEnabled             bool     `json:"proactiveSupportEnabled"`
	ExpirationDate                      *APITime `json:"expirationDate,omitempty"`
	SupportExpirationDate               *APITime `json:"supportExpirationDate,omitempty"`
	InstanceSummary                     *LicenseInstanceSummary
	CapacitySummary                     *LicenseCapacitySummary
	SocketSummary                       *LicenseSocketSummary
	Raw                                 map[string]any
}

// LicenseInstanceSummary captures high-level instance allocation metrics.
type LicenseInstanceSummary struct {
	LicensedInstancesNumber              float64  `json:"licensedInstancesNumber"`
	UsedInstancesNumber                  float64  `json:"usedInstancesNumber"`
	NewInstancesNumber                   float64  `json:"newInstancesNumber"`
	RentalInstancesNumber                float64  `json:"rentalInstancesNumber"`
	PromoInstancesNumber                 float64  `json:"promoInstancesNumber,omitempty"`
	LicensedInstancesPromoIncludedNumber float64  `json:"licensedInstancesPromoIncludedNumber,omitempty"`
	PromoExpiresOn                       *APITime `json:"promoExpiresOn,omitempty"`
	Raw                                  map[string]any
}

// LicenseCapacitySummary summarises capacity license consumption.
type LicenseCapacitySummary struct {
	LicensedCapacityTB float64 `json:"licensedCapacityTb"`
	UsedCapacityTB     float64 `json:"usedCapacityTb"`
	Raw                map[string]any
}

// LicenseSocketSummary summarises socket license allocations.
type LicenseSocketSummary struct {
	LicensedSocketsNumber  int `json:"licensedSocketsNumber"`
	UsedSocketsNumber      int `json:"usedSocketsNumber"`
	RemainingSocketsNumber int `json:"remainingSocketsNumber"`
	Raw                    map[string]any
}

// LicenseSocketsFilter controls socket workload listing.
type LicenseSocketsFilter struct {
	Name           string
	HostName       string
	HostID         string
	SocketsNumber  int
	CoresNumber    int
	Type           string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

// SocketLicenseWorkload represents a protected host consuming socket licenses.
type SocketLicenseWorkload struct {
	Name          string         `json:"name"`
	HostName      string         `json:"hostName"`
	HostID        string         `json:"hostId"`
	SocketsNumber int            `json:"socketsNumber"`
	CoresNumber   int            `json:"coresNumber"`
	Type          string         `json:"type"`
	Raw           map[string]any `json:"-"`
}

// LicenseSocketsResult returns socket workloads with pagination metadata.
type LicenseSocketsResult struct {
	Workloads  []SocketLicenseWorkload
	Pagination paginationResult
	Raw        map[string]any
}

// InstanceLicensesFilter controls instance workload listing.
type InstanceLicensesFilter struct {
	Name                string
	HostName            string
	UsedInstancesNumber float64
	Type                string
	InstanceID          string
	OrderColumn         string
	OrderAscending      *bool
	MaxItems            int
}

// InstanceLicenseWorkload represents a workload consuming instance licenses.
type InstanceLicenseWorkload struct {
	Name                string         `json:"name"`
	DisplayName         string         `json:"displayName"`
	HostName            string         `json:"hostName"`
	UsedInstancesNumber float64        `json:"usedInstancesNumber"`
	Type                string         `json:"type"`
	InstanceID          string         `json:"instanceId"`
	PlatformType        string         `json:"platformType"`
	CanBeRevoked        bool           `json:"canBeRevoked"`
	Raw                 map[string]any `json:"-"`
}

// LicenseInstancesResult returns instance workloads.
type LicenseInstancesResult struct {
	Workloads  []InstanceLicenseWorkload
	Pagination paginationResult
	Raw        map[string]any
}

// CapacityLicenseWorkload captures unstructured data consumption.
type CapacityLicenseWorkload struct {
	Name           string         `json:"name"`
	UsedCapacityTB float64        `json:"usedCapacityTb"`
	Type           string         `json:"type"`
	InstanceID     string         `json:"instanceId"`
	Raw            map[string]any `json:"-"`
}

// LicenseCapacityResult lists unstructured data workloads.
type LicenseCapacityResult struct {
	Workloads []CapacityLicenseWorkload
	Raw       map[string]any
}

// GlobalVMExclusion represents a VM-level exclusion entry.
type GlobalVMExclusion struct {
	ID              string         `json:"id"`
	Note            string         `json:"note,omitempty"`
	InventoryObject map[string]any `json:"inventoryObject"`
	Raw             map[string]any `json:"-"`
}

// GlobalVMExclusionsFilter controls exclusion listing.
type GlobalVMExclusionsFilter struct {
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

// GlobalVMExclusionsResult returns VM exclusions with pagination.
type GlobalVMExclusionsResult struct {
	Exclusions []GlobalVMExclusion
	Pagination paginationResult
	Raw        map[string]any
}

// Service represents an auxiliary service entry.
type Service struct {
	Name string         `json:"name"`
	Port int            `json:"port"`
	Raw  map[string]any `json:"-"`
}

// ServicesFilter controls service listing.
type ServicesFilter struct {
	Name           string
	OrderColumn    string
	OrderAscending *bool
	MaxItems       int
}

// ServicesResult returns integration services.
type ServicesResult struct {
	Services   []Service
	Pagination paginationResult
	Raw        map[string]any
}

// TrafficRule models a global network traffic policy.
type TrafficRule struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	SourceIPStart           string         `json:"sourceIPStart"`
	SourceIPEnd             string         `json:"sourceIPEnd"`
	TargetIPStart           string         `json:"targetIPStart"`
	TargetIPEnd             string         `json:"targetIPEnd"`
	EncryptionEnabled       bool           `json:"encryptionEnabled"`
	ThrottlingEnabled       bool           `json:"throttlingEnabled"`
	ThrottlingValue         int            `json:"throttlingValue"`
	ThrottlingUnit          string         `json:"throttlingUnit"`
	ThrottlingWindowEnabled bool           `json:"throttlingWindowEnabled"`
	Raw                     map[string]any `json:"-"`
}

// PreferredNetwork identifies preferred subnets for transfer.
type PreferredNetwork struct {
	IPAddress  string         `json:"ipAddress"`
	SubnetMask string         `json:"subnetMask"`
	CIDR       string         `json:"cidrNotation"`
	Raw        map[string]any `json:"-"`
}

// TrafficRulesResult aggregates traffic settings.
type TrafficRulesResult struct {
	UseMultipleStreamsPerJob bool
	UploadStreamsCount       int
	PreferredNetworksEnabled bool
	PreferredNetworks        []PreferredNetwork
	Rules                    []TrafficRule
	Raw                      map[string]any
}

// GeneralOptions summarises global notification/siem settings.
type GeneralOptions struct {
	NotificationEnabled            bool
	StorageSpaceThresholdEnabled   bool
	DatastoreSpaceThresholdEnabled bool
	SkipVMSpaceThresholdEnabled    bool
	NotifyOnSupportExpiration      bool
	NotifyOnUpdates                bool
	SIEMSNMPEnabled                bool
	SIEMSyslogEnabled              bool
	Raw                            map[string]any
}

// ConfigBackup captures configuration backup policy.
type ConfigBackup struct {
	IsEnabled           bool
	BackupRepositoryID  string
	RestorePointsToKeep int
	EncryptionEnabled   bool
	EncryptionPassword  string
	LastSessionID       string
	LastRunTime         *APITime
	Raw                 map[string]any
}

// LicenseSummary retrieves the main license payload.
func (c *Client) LicenseSummary(ctx context.Context) (*InstalledLicense, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/license", &payload); err != nil {
		return nil, err
	}

	var summary InstalledLicense
	if _, err := decodeObject(payload, &summary); err != nil {
		return nil, fmt.Errorf("decode license summary: %w", err)
	}
	summary.Raw = payload

	if rawInstance, ok := extractMap(payload, "instanceLicenseSummary"); ok {
		var inst LicenseInstanceSummary
		if _, err := decodeObject(rawInstance, &inst); err == nil {
			inst.Raw = rawInstance
			summary.InstanceSummary = &inst
		}
	}
	if rawCapacity, ok := extractMap(payload, "capacityLicenseSummary"); ok {
		var cap LicenseCapacitySummary
		if _, err := decodeObject(rawCapacity, &cap); err == nil {
			cap.Raw = rawCapacity
			summary.CapacitySummary = &cap
		}
	}
	if rawSockets, ok := extractMap(payload, "socketLicenseSummary"); ok {
		var sockets LicenseSocketSummary
		if _, err := decodeObject(rawSockets, &sockets); err == nil {
			sockets.Raw = rawSockets
			summary.SocketSummary = &sockets
		}
	}

	return &summary, nil
}

// LicenseSockets enumerates socket workloads.
func (c *Client) LicenseSockets(ctx context.Context, filter LicenseSocketsFilter) (*LicenseSocketsResult, error) {
	workloads := make([]SocketLicenseWorkload, 0)
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
		if filter.HostName != "" {
			query.Set("hostNameFilter", filter.HostName)
		}
		if filter.HostID != "" {
			query.Set("hostIdFilter", filter.HostID)
		}
		if filter.SocketsNumber > 0 {
			query.Set("socketsNumberFilter", strconv.Itoa(filter.SocketsNumber))
		}
		if filter.CoresNumber > 0 {
			query.Set("coresNumberFilter", strconv.Itoa(filter.CoresNumber))
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
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
		if err := c.getJSONWithQuery(ctx, "/api/v1/license/sockets", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var workload SocketLicenseWorkload
			raw, err := decodeRawMessage(entry, &workload)
			if err != nil {
				return nil, fmt.Errorf("decode socket workload: %w", err)
			}
			workload.Raw = raw
			workloads = append(workloads, workload)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(workloads) >= filter.MaxItems {
				pagination = resp.Pagination
				return &LicenseSocketsResult{
					Workloads:  workloads[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(workloads)
			if remaining <= 0 {
				break
			}
		}
	}

	return &LicenseSocketsResult{
		Workloads:  workloads,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// LicenseInstances enumerates instance-based workloads.
func (c *Client) LicenseInstances(ctx context.Context, filter InstanceLicensesFilter) (*LicenseInstancesResult, error) {
	workloads := make([]InstanceLicenseWorkload, 0)
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
		if filter.HostName != "" {
			query.Set("hostNameFilter", filter.HostName)
		}
		if filter.UsedInstancesNumber > 0 {
			query.Set("usedInstancesNumberFilter", strconv.FormatFloat(filter.UsedInstancesNumber, 'f', -1, 64))
		}
		if filter.Type != "" {
			query.Set("typeFilter", filter.Type)
		}
		if filter.InstanceID != "" {
			query.Set("instanceIdFilter", filter.InstanceID)
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
		if err := c.getJSONWithQuery(ctx, "/api/v1/license/instances", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var workload InstanceLicenseWorkload
			raw, err := decodeRawMessage(entry, &workload)
			if err != nil {
				return nil, fmt.Errorf("decode instance workload: %w", err)
			}
			workload.Raw = raw
			workloads = append(workloads, workload)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(workloads) >= filter.MaxItems {
				pagination = resp.Pagination
				return &LicenseInstancesResult{
					Workloads:  workloads[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(workloads)
			if remaining <= 0 {
				break
			}
		}
	}

	return &LicenseInstancesResult{
		Workloads:  workloads,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// LicenseCapacity lists unstructured data workloads contributing to capacity licenses.
func (c *Client) LicenseCapacity(ctx context.Context) (*LicenseCapacityResult, error) {
	var resp struct {
		Workloads []json.RawMessage `json:"workloads"`
	}
	if err := c.getJSON(ctx, "/api/v1/license/capacity", &resp); err != nil {
		return nil, err
	}

	workloads := make([]CapacityLicenseWorkload, 0, len(resp.Workloads))
	rawItems := make([]map[string]any, 0, len(resp.Workloads))
	for _, entry := range resp.Workloads {
		var workload CapacityLicenseWorkload
		raw, err := decodeRawMessage(entry, &workload)
		if err != nil {
			return nil, fmt.Errorf("decode capacity workload: %w", err)
		}
		workload.Raw = raw
		workloads = append(workloads, workload)
		rawItems = append(rawItems, raw)
	}

	return &LicenseCapacityResult{
		Workloads: workloads,
		Raw: map[string]any{
			"workloads": rawItems,
		},
	}, nil
}

// GlobalVMExclusions lists VM exclusions.
func (c *Client) GlobalVMExclusions(ctx context.Context, filter GlobalVMExclusionsFilter) (*GlobalVMExclusionsResult, error) {
	exclusions := make([]GlobalVMExclusion, 0)
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
		if err := c.getJSONWithQuery(ctx, "/api/v1/globalExclusions/vm", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var exclusion GlobalVMExclusion
			raw, err := decodeRawMessage(entry, &exclusion)
			if err != nil {
				return nil, fmt.Errorf("decode global VM exclusion: %w", err)
			}
			exclusion.Raw = raw
			// Ensure InventoryObject defaults to empty map for consistency.
			if exclusion.InventoryObject == nil {
				if inv, ok := raw["inventoryObject"].(map[string]any); ok {
					exclusion.InventoryObject = inv
				} else {
					exclusion.InventoryObject = map[string]any{}
				}
			}
			exclusions = append(exclusions, exclusion)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(exclusions) >= filter.MaxItems {
				pagination = resp.Pagination
				return &GlobalVMExclusionsResult{
					Exclusions: exclusions[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(exclusions)
			if remaining <= 0 {
				break
			}
		}
	}

	return &GlobalVMExclusionsResult{
		Exclusions: exclusions,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// Services lists registered integration services.
func (c *Client) Services(ctx context.Context, filter ServicesFilter) (*ServicesResult, error) {
	services := make([]Service, 0)
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
		if err := c.getJSONWithQuery(ctx, "/api/v1/services", query, &resp); err != nil {
			return nil, err
		}

		for _, entry := range resp.Data {
			var service Service
			raw, err := decodeRawMessage(entry, &service)
			if err != nil {
				return nil, fmt.Errorf("decode service entry: %w", err)
			}
			service.Raw = raw
			services = append(services, service)
			rawItems = append(rawItems, raw)
			if filter.MaxItems > 0 && len(services) >= filter.MaxItems {
				pagination = resp.Pagination
				return &ServicesResult{
					Services:   services[:filter.MaxItems],
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
			remaining = filter.MaxItems - len(services)
			if remaining <= 0 {
				break
			}
		}
	}

	return &ServicesResult{
		Services:   services,
		Pagination: pagination,
		Raw: map[string]any{
			"data":       rawItems,
			"pagination": pagination,
		},
	}, nil
}

// TrafficRules retrieves global traffic configuration.
func (c *Client) TrafficRules(ctx context.Context) (*TrafficRulesResult, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/trafficRules", &payload); err != nil {
		return nil, err
	}

	result := &TrafficRulesResult{
		Raw: payload,
	}

	if v, ok := payload["useMultipleStreamsPerJob"].(bool); ok {
		result.UseMultipleStreamsPerJob = v
	}
	if v, ok := payload["uploadStreamsCount"].(float64); ok {
		result.UploadStreamsCount = int(v)
	}

	if preferred, ok := extractMap(payload, "preferredNetworks"); ok {
		if enabled, ok := preferred["isEnabled"].(bool); ok {
			result.PreferredNetworksEnabled = enabled
		}
		if networks, ok := preferred["networks"].([]any); ok {
			for _, entry := range networks {
				rawNet, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				var network PreferredNetwork
				if _, err := decodeObject(rawNet, &network); err == nil {
					network.Raw = rawNet
					result.PreferredNetworks = append(result.PreferredNetworks, network)
				}
			}
		}
	}

	if rules, ok := payload["trafficRules"].([]any); ok {
		for _, entry := range rules {
			rawRule, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			var rule TrafficRule
			if _, err := decodeObject(rawRule, &rule); err == nil {
				rule.Raw = rawRule
				// Normalise integer values coming from float decoding.
				if v, ok := rawRule["throttlingValue"].(float64); ok {
					rule.ThrottlingValue = int(v)
				}
				result.Rules = append(result.Rules, rule)
			}
		}
	}

	return result, nil
}

// GeneralOptions fetches global options.
func (c *Client) GeneralOptions(ctx context.Context) (*GeneralOptions, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/generalOptions", &payload); err != nil {
		return nil, err
	}

	opts := &GeneralOptions{
		Raw: payload,
	}
	if v, ok := payload["notificationEnabled"].(bool); ok {
		opts.NotificationEnabled = v
	}

	if notifications, ok := extractMap(payload, "notifications"); ok {
		opts.StorageSpaceThresholdEnabled = boolValue(notifications, "storageSpaceThresholdEnabled")
		opts.DatastoreSpaceThresholdEnabled = boolValue(notifications, "datastoreSpaceThresholdEnabled")
		opts.SkipVMSpaceThresholdEnabled = boolValue(notifications, "skipVMSpaceThresholdEnabled")
		opts.NotifyOnSupportExpiration = boolValue(notifications, "notifyOnSupportExpiration")
		opts.NotifyOnUpdates = boolValue(notifications, "notifyOnUpdates")
	}

	if siem, ok := extractMap(payload, "siemIntegration"); ok {
		opts.SIEMSNMPEnabled = boolValue(siem, "SNMPEventsEnabled")
		opts.SIEMSyslogEnabled = boolValue(siem, "SyslogEventsEnabled")
	}

	return opts, nil
}

// ConfigBackup retrieves configuration backup settings.
func (c *Client) ConfigBackup(ctx context.Context) (*ConfigBackup, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, "/api/v1/configBackup", &payload); err != nil {
		return nil, err
	}

	result := &ConfigBackup{
		Raw: payload,
	}

	if v, ok := payload["isEnabled"].(bool); ok {
		result.IsEnabled = v
	}
	if v, ok := payload["backupRepositoryId"].(string); ok {
		result.BackupRepositoryID = v
	}
	if v, ok := payload["restorePointsToKeep"].(float64); ok {
		result.RestorePointsToKeep = int(v)
	}

	if encryption, ok := extractMap(payload, "encryption"); ok {
		result.EncryptionEnabled = boolValue(encryption, "isEnabled")
		if id, ok := encryption["passwordId"].(string); ok {
			result.EncryptionPassword = id
		}
	}

	if last, ok := extractMap(payload, "lastSuccessfulBackup"); ok {
		if id, ok := last["sessionId"].(string); ok {
			result.LastSessionID = id
		}
		if ts, ok := last["lastSuccessfulTime"].(string); ok && ts != "" {
			var parsed APITime
			if err := parsed.UnmarshalJSON([]byte(strconv.Quote(ts))); err == nil {
				result.LastRunTime = &parsed
			}
		}
	}

	return result, nil
}

// StartConfigBackup triggers an on-demand configuration backup.
func (c *Client) StartConfigBackup(ctx context.Context) (*Session, error) {
	var sess Session
	if err := c.postJSON(ctx, "/api/v1/configBackup/backup", nil, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func extractMap(payload map[string]any, key string) (map[string]any, bool) {
	if value, ok := payload[key]; ok {
		if mapped, ok := value.(map[string]any); ok {
			return mapped, true
		}
	}
	return nil, false
}

func boolValue(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
