package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// InstantRecoveryPlatform identifies the source endpoint a mount originates from.
type InstantRecoveryPlatform string

const (
	InstantRecoveryPlatformVSphere         InstantRecoveryPlatform = "vsphere"
	InstantRecoveryPlatformHyperV          InstantRecoveryPlatform = "hyperv"
	InstantRecoveryPlatformAzure           InstantRecoveryPlatform = "azure"
	InstantRecoveryPlatformVSphereFCD      InstantRecoveryPlatform = "vsphere-fcd"
	InstantRecoveryPlatformDataIntegration InstantRecoveryPlatform = "data-integration"
)

// InstantRecoveryMountFilter narrows the mount listing.
type InstantRecoveryMountFilter struct {
	Platforms []InstantRecoveryPlatform
	State     string
	Name      string
	Limit     int
}

// InstantRecoveryMountSummary contains a normalised view of a mount.
type InstantRecoveryMountSummary struct {
	ID              string                        `json:"id"`
	Platform        InstantRecoveryPlatform       `json:"platform"`
	Object          string                        `json:"object"`
	Job             string                        `json:"jobName,omitempty"`
	Backup          string                        `json:"backupName,omitempty"`
	Host            string                        `json:"host,omitempty"`
	State           string                        `json:"state"`
	Mode            string                        `json:"mode,omitempty"`
	SessionID       string                        `json:"sessionId,omitempty"`
	RestorePointID  string                        `json:"restorePointId,omitempty"`
	RestorePoint    time.Time                     `json:"restorePointDate"`
	Error           string                        `json:"errorMessage,omitempty"`
	Azure           *AzureInstantRecoveryMetadata `json:"azure,omitempty"`
	DataIntegration *DataIntegrationMountMetadata `json:"dataIntegration,omitempty"`
	FCD             *FCDInstantRecoveryMetadata   `json:"fcd,omitempty"`
	Raw             map[string]any                `json:"-"`
}

// AzureInstantRecoveryMetadata carries Azure specific fields.
type AzureInstantRecoveryMetadata struct {
	Name           string `json:"name"`
	Subscription   string `json:"subscriptionId"`
	Location       string `json:"location"`
	ResourceGroup  string `json:"resourceGroup"`
	Network        string `json:"network"`
	AssignPublicIP bool   `json:"assignPublicIp"`
	VerifyBoot     bool   `json:"verifyVmBoot"`
}

// DataIntegrationMountMetadata captures data integration publishing details.
type DataIntegrationMountMetadata struct {
	Mode       string   `json:"mode"`
	ServerIPs  []string `json:"serverIps"`
	ServerPort int      `json:"serverPort"`
}

// FCDInstantRecoveryMetadata summarises mounted disks.
type FCDInstantRecoveryMetadata struct {
	Destination string   `json:"destinationCluster"`
	DiskCount   int      `json:"diskCount"`
	Disks       []string `json:"disks"`
}

// InstantRecoveryMountDetail augments the summary with raw payload.
type InstantRecoveryMountDetail struct {
	Platform InstantRecoveryPlatform
	Summary  InstantRecoveryMountSummary
	Raw      map[string]any
}

// AzureInstantRecoverySwitchoverSettings models switchover configuration.
type AzureInstantRecoverySwitchoverSettings struct {
	Type         string     `json:"type"`
	ScheduleTime *time.Time `json:"scheduleTime,omitempty"`
	VerifyVMBoot bool       `json:"verifyVMBoot,omitempty"`
	PowerOnVM    bool       `json:"powerOnVM,omitempty"`
}

// InstantRecoveryMounts retrieves mount summaries across the selected platforms.
func (c *Client) InstantRecoveryMounts(ctx context.Context, filter InstantRecoveryMountFilter) ([]InstantRecoveryMountSummary, error) {
	platforms := filter.Platforms
	if len(platforms) == 0 {
		platforms = []InstantRecoveryPlatform{
			InstantRecoveryPlatformVSphere,
			InstantRecoveryPlatformHyperV,
			InstantRecoveryPlatformAzure,
			InstantRecoveryPlatformVSphereFCD,
			InstantRecoveryPlatformDataIntegration,
		}
	}

	results := make([]InstantRecoveryMountSummary, 0)
	remaining := filter.Limit

	for _, platform := range platforms {
		if filter.Limit > 0 && remaining <= 0 {
			break
		}

		platformLimit := 0
		if filter.Limit > 0 {
			platformLimit = remaining
		}

		var (
			summaries []InstantRecoveryMountSummary
			err       error
		)

		switch platform {
		case InstantRecoveryPlatformVSphere:
			summaries, err = c.fetchInstantVMMounts(ctx, "/api/v1/restore/instantRecovery/vSphere/vm", platform, filter, platformLimit)
		case InstantRecoveryPlatformHyperV:
			summaries, err = c.fetchInstantVMMounts(ctx, "/api/v1/restore/instantRecovery/hyperV/vm", platform, filter, platformLimit)
		case InstantRecoveryPlatformAzure:
			summaries, err = c.fetchAzureInstantRecoveryMounts(ctx, filter, platformLimit)
		case InstantRecoveryPlatformVSphereFCD:
			summaries, err = c.fetchFCDInstantRecoveryMounts(ctx, filter, platformLimit)
		case InstantRecoveryPlatformDataIntegration:
			summaries, err = c.fetchDataIntegrationMounts(ctx, filter, platformLimit)
		default:
			err = fmt.Errorf("unsupported mount platform %q", platform)
		}

		if err != nil {
			return nil, err
		}

		results = append(results, summaries...)
		if filter.Limit > 0 {
			remaining -= len(summaries)
		}
	}

	if filter.Limit > 0 && len(results) > filter.Limit {
		return results[:filter.Limit], nil
	}
	return results, nil
}

// InstantRecoveryMountDetail retrieves detailed information for a specific mount ID.
func (c *Client) InstantRecoveryMountDetail(ctx context.Context, id string) (*InstantRecoveryMountDetail, error) {
	if mount, raw, err := c.fetchInstantVMMountDetail(ctx, "/api/v1/restore/instantRecovery/vSphere/vm/"+id, InstantRecoveryPlatformVSphere); err == nil {
		return &InstantRecoveryMountDetail{
			Platform: InstantRecoveryPlatformVSphere,
			Summary:  mount.toSummary(InstantRecoveryPlatformVSphere),
			Raw:      raw,
		}, nil
	} else if !isNotFound(err) {
		return nil, err
	}

	if mount, raw, err := c.fetchInstantVMMountDetail(ctx, "/api/v1/restore/instantRecovery/hyperV/vm/"+id, InstantRecoveryPlatformHyperV); err == nil {
		return &InstantRecoveryMountDetail{
			Platform: InstantRecoveryPlatformHyperV,
			Summary:  mount.toSummary(InstantRecoveryPlatformHyperV),
			Raw:      raw,
		}, nil
	} else if !isNotFound(err) {
		return nil, err
	}

	if mount, raw, err := c.fetchAzureInstantRecoveryMountDetail(ctx, id); err == nil {
		return &InstantRecoveryMountDetail{
			Platform: InstantRecoveryPlatformAzure,
			Summary:  mount.toSummary(),
			Raw:      raw,
		}, nil
	} else if !isNotFound(err) {
		return nil, err
	}

	if mount, raw, err := c.fetchFCDInstantRecoveryMountDetail(ctx, id); err == nil {
		return &InstantRecoveryMountDetail{
			Platform: InstantRecoveryPlatformVSphereFCD,
			Summary:  mount.toSummary(),
			Raw:      raw,
		}, nil
	} else if !isNotFound(err) {
		return nil, err
	}

	if mount, raw, err := c.fetchDataIntegrationMountDetail(ctx, id); err == nil {
		return &InstantRecoveryMountDetail{
			Platform: InstantRecoveryPlatformDataIntegration,
			Summary:  mount.toSummary(),
			Raw:      raw,
		}, nil
	} else if !isNotFound(err) {
		return nil, err
	}

	return nil, fmt.Errorf("instant recovery mount %q not found", id)
}

// AzureInstantRecoveryMountSessions fetches historical sessions for an Azure mount.
func (c *Client) AzureInstantRecoveryMountSessions(ctx context.Context, mountID string) ([]Session, error) {
	path := fmt.Sprintf("/api/v1/restore/instantRecovery/azure/vm/%s/sessions", mountID)
	var resp sessionResponse
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// AzureInstantRecoverySwitchoverSettings retrieves switchover configuration for an Azure mount.
func (c *Client) AzureInstantRecoverySwitchoverSettings(ctx context.Context, mountID string) (*AzureInstantRecoverySwitchoverSettings, error) {
	path := fmt.Sprintf("/api/v1/restore/instantRecovery/azure/vm/%s/switchoverSettings", mountID)
	var settings AzureInstantRecoverySwitchoverSettings
	if err := c.getJSON(ctx, path, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (c *Client) fetchInstantVMMounts(ctx context.Context, endpoint string, platform InstantRecoveryPlatform, filter InstantRecoveryMountFilter, limit int) ([]InstantRecoveryMountSummary, error) {
	results := make([]InstantRecoveryMountSummary, 0)
	skip := 0
	remaining := limit

	for {
		pageLimit := defaultPageSize
		if remaining > 0 && remaining < pageLimit {
			pageLimit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(pageLimit))
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}
		if filter.Name != "" {
			query.Set("vmNameFilter", filter.Name)
		}

		var resp instantVMRecoveryMountsResponse
		if err := c.getJSONWithQuery(ctx, endpoint, query, &resp); err != nil {
			return nil, err
		}

		for _, mount := range resp.Data {
			raw, err := structToMap(mount)
			if err != nil {
				return nil, fmt.Errorf("decode instant VM mount: %w", err)
			}
			summary := mount.toSummary(platform)
			summary.Raw = raw
			results = append(results, summary)
			if limit > 0 && len(results) >= limit {
				return results[:limit], nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if limit > 0 {
			remaining = limit - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

func (c *Client) fetchInstantVMMountDetail(ctx context.Context, path string, platform InstantRecoveryPlatform) (*instantVMRecoveryMount, map[string]any, error) {
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("encode mount payload: %w", err)
	}

	var mount instantVMRecoveryMount
	if err := json.Unmarshal(raw, &mount); err != nil {
		return nil, nil, fmt.Errorf("decode mount payload: %w", err)
	}
	return &mount, payload, nil
}

func (c *Client) fetchAzureInstantRecoveryMounts(ctx context.Context, filter InstantRecoveryMountFilter, limit int) ([]InstantRecoveryMountSummary, error) {
	results := make([]InstantRecoveryMountSummary, 0)
	skip := 0

	for {
		pageLimit := defaultPageSize
		if limit > 0 {
			remaining := limit - len(results)
			if remaining <= 0 {
				break
			}
			if remaining < pageLimit {
				pageLimit = remaining
			}
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(pageLimit))
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}

		var resp azureInstantVMRecoveryMountsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/restore/instantRecovery/azure/vm", query, &resp); err != nil {
			return nil, err
		}

		for _, mount := range resp.Data {
			raw, err := structToMap(mount)
			if err != nil {
				return nil, fmt.Errorf("decode azure instant recovery mount: %w", err)
			}
			summary := mount.toSummary()
			if filter.Name != "" && !matchesPattern(summary.Object, filter.Name) {
				continue
			}
			summary.Raw = raw
			results = append(results, summary)
			if limit > 0 && len(results) >= limit {
				return results[:limit], nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count
	}

	return results, nil
}

func (c *Client) fetchAzureInstantRecoveryMountDetail(ctx context.Context, id string) (*azureInstantVMRecoveryMount, map[string]any, error) {
	path := fmt.Sprintf("/api/v1/restore/instantRecovery/azure/vm/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("encode azure mount payload: %w", err)
	}

	var mount azureInstantVMRecoveryMount
	if err := json.Unmarshal(raw, &mount); err != nil {
		return nil, nil, fmt.Errorf("decode azure mount payload: %w", err)
	}
	return &mount, payload, nil
}

func (c *Client) fetchFCDInstantRecoveryMounts(ctx context.Context, filter InstantRecoveryMountFilter, limit int) ([]InstantRecoveryMountSummary, error) {
	results := make([]InstantRecoveryMountSummary, 0)
	skip := 0
	remaining := limit

	for {
		pageLimit := defaultPageSize
		if remaining > 0 && remaining < pageLimit {
			pageLimit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(pageLimit))
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}

		var resp vmwareFcdInstantRecoveryMountsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/restore/instantRecovery/vSphere/fcd", query, &resp); err != nil {
			return nil, err
		}

		for _, mount := range resp.Data {
			raw, err := structToMap(mount)
			if err != nil {
				return nil, fmt.Errorf("decode fcd instant recovery mount: %w", err)
			}
			summary := mount.toSummary()
			summary.Raw = raw
			results = append(results, summary)
			if limit > 0 && len(results) >= limit {
				return results[:limit], nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if limit > 0 {
			remaining = limit - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

func (c *Client) fetchFCDInstantRecoveryMountDetail(ctx context.Context, id string) (*vmwareFcdInstantRecoveryMount, map[string]any, error) {
	path := fmt.Sprintf("/api/v1/restore/instantRecovery/vSphere/fcd/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("encode fcd mount payload: %w", err)
	}
	var mount vmwareFcdInstantRecoveryMount
	if err := json.Unmarshal(raw, &mount); err != nil {
		return nil, nil, fmt.Errorf("decode fcd mount payload: %w", err)
	}
	return &mount, payload, nil
}

func (c *Client) fetchDataIntegrationMounts(ctx context.Context, filter InstantRecoveryMountFilter, limit int) ([]InstantRecoveryMountSummary, error) {
	results := make([]InstantRecoveryMountSummary, 0)
	skip := 0
	remaining := limit

	for {
		pageLimit := defaultPageSize
		if remaining > 0 && remaining < pageLimit {
			pageLimit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(pageLimit))
		if filter.State != "" {
			query.Set("stateFilter", filter.State)
		}

		var resp backupContentMountsResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/dataIntegration", query, &resp); err != nil {
			return nil, err
		}

		for _, mount := range resp.Data {
			raw, err := structToMap(mount)
			if err != nil {
				return nil, fmt.Errorf("decode data integration mount: %w", err)
			}
			summary := mount.toSummary()
			summary.Raw = raw
			results = append(results, summary)
			if limit > 0 && len(results) >= limit {
				return results[:limit], nil
			}
		}

		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count

		if limit > 0 {
			remaining = limit - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

func (c *Client) fetchDataIntegrationMountDetail(ctx context.Context, id string) (*backupContentMount, map[string]any, error) {
	path := fmt.Sprintf("/api/v1/dataIntegration/%s", id)
	payload := make(map[string]any)
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("encode data integration payload: %w", err)
	}
	var mount backupContentMount
	if err := json.Unmarshal(raw, &mount); err != nil {
		return nil, nil, fmt.Errorf("decode data integration payload: %w", err)
	}
	return &mount, payload, nil
}

type instantVMRecoveryMountsResponse struct {
	Data       []instantVMRecoveryMount `json:"data"`
	Pagination paginationResult         `json:"pagination"`
}

type instantVMRecoveryMount struct {
	ID               string                `json:"id"`
	SessionID        string                `json:"sessionId"`
	State            string                `json:"state"`
	Spec             instantVMRecoverySpec `json:"spec"`
	VMName           string                `json:"vmName"`
	JobName          string                `json:"jobName"`
	RestorePointDate time.Time             `json:"restorePointDate"`
	HostName         string                `json:"hostName"`
	ErrorMessage     string                `json:"errorMessage"`
}

type instantVMRecoverySpec struct {
	RestorePointID string `json:"restorePointId"`
	Type           string `json:"type"`
}

func (m instantVMRecoveryMount) toSummary(platform InstantRecoveryPlatform) InstantRecoveryMountSummary {
	return InstantRecoveryMountSummary{
		ID:             m.ID,
		Platform:       platform,
		Object:         m.VMName,
		Job:            m.JobName,
		Host:           m.HostName,
		State:          m.State,
		Mode:           m.Spec.Type,
		SessionID:      m.SessionID,
		RestorePointID: m.Spec.RestorePointID,
		RestorePoint:   m.RestorePointDate,
		Error:          m.ErrorMessage,
	}
}

type azureInstantVMRecoveryMountsResponse struct {
	Data       []azureInstantVMRecoveryMount `json:"data"`
	Pagination paginationResult              `json:"pagination"`
}

type azureInstantVMRecoveryMount struct {
	ID           string                     `json:"id"`
	State        string                     `json:"state"`
	Spec         azureInstantVMRecoverySpec `json:"spec"`
	ErrorMessage string                     `json:"errorMessage"`
}

type azureInstantVMRecoverySpec struct {
	RestorePointID string                         `json:"restorePointId"`
	Subscription   azureComputeSubscriptionModel  `json:"subscription"`
	Name           azureComputeNameModel          `json:"name"`
	ResourceGroup  azureComputeResourceGroupModel `json:"resourceGroup"`
	Network        azureComputeNetworkModel       `json:"network"`
	VerifyVMBoot   bool                           `json:"verifyVMBoot,omitempty"`
}

type azureComputeSubscriptionModel struct {
	SubscriptionID string `json:"subscriptionId"`
	Location       string `json:"location"`
}

type azureComputeNameModel struct {
	Name string `json:"name"`
}

type azureComputeResourceGroupModel struct {
	ResourceGroup    string `json:"resourceGroup"`
	NewResourceGroup string `json:"newResourceGroupName"`
}

type azureComputeNetworkModel struct {
	Network              string `json:"network"`
	Subnet               string `json:"subnet"`
	NetworkSecurityGroup string `json:"networkSecurityGroup"`
	AssignPublicIP       bool   `json:"assignPublicIp"`
}

func (m azureInstantVMRecoveryMount) toSummary() InstantRecoveryMountSummary {
	metadata := &AzureInstantRecoveryMetadata{
		Name:           m.Spec.Name.Name,
		Subscription:   m.Spec.Subscription.SubscriptionID,
		Location:       m.Spec.Subscription.Location,
		ResourceGroup:  m.Spec.ResourceGroup.ResourceGroup,
		Network:        m.Spec.Network.Network,
		AssignPublicIP: m.Spec.Network.AssignPublicIP,
		VerifyBoot:     m.Spec.VerifyVMBoot,
	}
	return InstantRecoveryMountSummary{
		ID:             m.ID,
		Platform:       InstantRecoveryPlatformAzure,
		Object:         m.Spec.Name.Name,
		State:          m.State,
		RestorePointID: m.Spec.RestorePointID,
		Error:          m.ErrorMessage,
		Azure:          metadata,
	}
}

type vmwareFcdInstantRecoveryMountsResponse struct {
	Data       []vmwareFcdInstantRecoveryMount `json:"data"`
	Pagination paginationResult                `json:"pagination"`
}

type vmwareFcdInstantRecoveryMount struct {
	ID           string                             `json:"id"`
	SessionID    string                             `json:"sessionId"`
	State        string                             `json:"state"`
	Spec         vmwareFcdInstantRecoverySpec       `json:"spec"`
	ErrorMessage string                             `json:"errorMessage"`
	MountedDisks []vmwareFcdInstantRecoveryDiskInfo `json:"mountedDisks"`
}

type vmwareFcdInstantRecoverySpec struct {
	RestorePointID     string               `json:"restorePointId"`
	DestinationCluster inventoryObjectModel `json:"destinationCluster"`
}

type inventoryObjectModel struct {
	Name string `json:"name"`
}

type vmwareFcdInstantRecoveryDiskInfo struct {
	NameInBackup      string `json:"nameInBackup"`
	MountedDiskName   string `json:"mountedDiskName"`
	RegisteredFcdName string `json:"registeredFcdName"`
	ObjectID          string `json:"objectId"`
}

func (m vmwareFcdInstantRecoveryMount) toSummary() InstantRecoveryMountSummary {
	disks := make([]string, 0, len(m.MountedDisks))
	for _, disk := range m.MountedDisks {
		disks = append(disks, disk.MountedDiskName)
	}
	return InstantRecoveryMountSummary{
		ID:             m.ID,
		Platform:       InstantRecoveryPlatformVSphereFCD,
		Object:         m.Spec.DestinationCluster.Name,
		State:          m.State,
		SessionID:      m.SessionID,
		RestorePointID: m.Spec.RestorePointID,
		Error:          m.ErrorMessage,
		FCD: &FCDInstantRecoveryMetadata{
			Destination: m.Spec.DestinationCluster.Name,
			DiskCount:   len(m.MountedDisks),
			Disks:       disks,
		},
	}
}

type backupContentMountsResponse struct {
	Data       []backupContentMount `json:"data"`
	Pagination paginationResult     `json:"pagination"`
}

type backupContentMount struct {
	ID               string                       `json:"id"`
	InitiatorName    string                       `json:"initiatorName"`
	BackupID         string                       `json:"backupId"`
	BackupName       string                       `json:"backupName"`
	RestorePointID   string                       `json:"restorePointId"`
	RestorePointName string                       `json:"restorePointName"`
	MountState       string                       `json:"mountState"`
	Info             backupContentPublicationInfo `json:"info"`
}

type backupContentPublicationInfo struct {
	Mode       string                             `json:"mode"`
	ServerPort int                                `json:"serverPort"`
	ServerIPs  []string                           `json:"serverIps"`
	Disks      []backupContentDiskPublicationInfo `json:"disks"`
}

type backupContentDiskPublicationInfo struct {
	Name string `json:"name"`
}

func (m backupContentMount) toSummary() InstantRecoveryMountSummary {
	return InstantRecoveryMountSummary{
		ID:             m.ID,
		Platform:       InstantRecoveryPlatformDataIntegration,
		Object:         m.RestorePointName,
		Backup:         m.BackupName,
		Host:           m.InitiatorName,
		State:          m.MountState,
		RestorePointID: m.RestorePointID,
		DataIntegration: &DataIntegrationMountMetadata{
			Mode:       m.Info.Mode,
			ServerIPs:  append([]string{}, m.Info.ServerIPs...),
			ServerPort: m.Info.ServerPort,
		},
	}
}

// DataIntegrationMounts lists published disk mounts (Data Integration API).
func (c *Client) DataIntegrationMounts(ctx context.Context, limit int) ([]InstantRecoveryMountSummary, error) {
	filter := InstantRecoveryMountFilter{
		Platforms: []InstantRecoveryPlatform{InstantRecoveryPlatformDataIntegration},
		Limit:     limit,
	}
	return c.fetchDataIntegrationMounts(ctx, filter, limit)
}

// DataIntegrationMount retrieves a specific published disk mount by ID.
func (c *Client) DataIntegrationMount(ctx context.Context, id string) (*InstantRecoveryMountDetail, error) {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return nil, fmt.Errorf("mount id cannot be empty")
	}

	mount, raw, err := c.fetchDataIntegrationMountDetail(ctx, clean)
	if err != nil {
		return nil, err
	}
	summary := mount.toSummary()
	summary.Raw = raw
	return &InstantRecoveryMountDetail{
		Platform: InstantRecoveryPlatformDataIntegration,
		Summary:  summary,
		Raw:      raw,
	}, nil
}

func structToMap(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var mapped map[string]any
	if err := json.Unmarshal(raw, &mapped); err != nil {
		return nil, err
	}
	return mapped, nil
}

func matchesPattern(value, pattern string) bool {
	if pattern == "" {
		return true
	}
	ok, err := path.Match(strings.ToLower(pattern), strings.ToLower(value))
	if err != nil {
		return strings.EqualFold(value, pattern)
	}
	return ok
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status 404")
}
