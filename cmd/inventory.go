package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

// --- Verb-first wiring helpers ---

func inventoryGetRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdInventoryUse,
		Short: helptext.InventoryGetRootShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(inventoryVirtualInfraListCmd())
	cmd.AddCommand(inventoryUnstructuredListCmd())
	cmd.AddCommand(inventoryProtectionGroupListCmd())
	return cmd
}

func inventoryDescribeRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdInventoryUse,
		Short: helptext.InventoryDescribeRootShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(inventoryVirtualInfraDescribeCmd())
	cmd.AddCommand(inventoryUnstructuredDescribeCmd())
	cmd.AddCommand(inventoryProtectionGroupDescribeCmd())
	return cmd
}

// --- Virtual infrastructure (vSphere, Hyper-V, Cloud Director) ---

func inventoryVirtualInfraListCmd() *cobra.Command {
	opts := inventoryVirtualListOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "virtualinfra",
		Short: helptext.InventoryVirtualListShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req := client.InventoryRequest{}
			if opts.limit > 0 {
				req.Pagination = &client.InventoryPagination{Limit: opts.limit}
			}
			if opts.sort != "" {
				req.Sorting = map[string]any{
					"property":  opts.sort,
					"direction": sortDirection(opts.desc),
				}
			}

			result, err := httpClient.InventoryServers(ctx, req)
			if err != nil {
				return err
			}

			platformFilters := normalizePlatformFilters(opts.platforms)
			typeFilters := normalizeSlice(opts.types)
			nameFilter := strings.TrimSpace(strings.ToLower(opts.name))
			hostFilter := strings.TrimSpace(strings.ToLower(opts.host))

			rows := make([]inventoryVirtualRow, 0, len(result.Data))
			for _, entry := range result.Data {
				row := inventoryVirtualRow{
					Name: stringFromMap(entry, "name"),
					Type: stringFromMap(entry, "type"),
					Host: stringFromMap(entry, "hostName"),
					Size: stringFromMap(entry, "size"),
				}
				row.Platform = friendlyPlatform(stringFromMap(entry, "platform"))

				if len(platformFilters) > 0 && !platformFilters[row.platformKey()] {
					continue
				}
				if len(typeFilters) > 0 && !typeFilters[strings.ToLower(row.Type)] {
					continue
				}
				if nameFilter != "" && !containsInsensitive(row.Name, nameFilter) {
					continue
				}
				if hostFilter != "" && !containsInsensitive(row.Host, hostFilter) {
					continue
				}

				rows = append(rows, row)
			}

			return output.Print(outputFormat(), rows)
		},
	}

	cmd.Flags().StringSliceVar(&opts.platforms, "platform", nil, "Filter by platform (vsphere|hyperv|clouddirector)")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "Filter by inventory type (e.g. VCenterServer, Scvmm)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by server name (case insensitive substring)")
	cmd.Flags().StringVar(&opts.host, "host", "", "Filter by host name")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to return (default 200)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by field (e.g. name)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")

	cmd.AddCommand(inventoryVirtualInfraObjectsCmd())
	return cmd
}

func inventoryVirtualInfraObjectsCmd() *cobra.Command {
	opts := inventoryVirtualObjectsOptions{
		limit:     200,
		hierarchy: "HostsAndClusters",
	}

	cmd := &cobra.Command{
		Use:   "objects",
		Short: helptext.InventoryVirtualObjectsShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.name) == "" {
				return requireFlag("--name", "provide the server or host name to inspect")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req := client.InventoryRequest{}
			if opts.limit > 0 {
				req.Pagination = &client.InventoryPagination{Limit: opts.limit}
			}
			if opts.hierarchy != "" {
				req.HierarchyType = opts.hierarchy
			}
			if opts.sort != "" {
				req.Sorting = map[string]any{
					"property":  opts.sort,
					"direction": sortDirection(opts.desc),
				}
			}

			result, err := httpClient.InventoryObjects(ctx, opts.name, req)
			if err != nil {
				return err
			}

			typeFilters := normalizeSlice(opts.objectTypes)
			nameFilter := strings.TrimSpace(strings.ToLower(opts.objectName))

			rows := make([]inventoryVirtualObjectRow, 0, len(result.Data))
			for _, entry := range result.Data {
				obj := mapFromMap(entry, "inventoryObject")
				if len(obj) == 0 {
					obj = entry
				}
				row := inventoryVirtualObjectRow{
					Name:     stringFromMap(obj, "name"),
					Type:     stringFromMap(obj, "type"),
					Host:     stringFromMap(obj, "hostName"),
					ObjectID: stringFromMap(obj, "objectId"),
					Size:     stringFromMap(entry, "size"),
				}
				if len(typeFilters) > 0 && !typeFilters[strings.ToLower(row.Type)] {
					continue
				}
				if nameFilter != "" && !containsInsensitive(row.Name, nameFilter) {
					continue
				}
				rows = append(rows, row)
			}

			return output.Print(outputFormat(), rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Server or host name to browse")
	cmd.Flags().StringVar(&opts.hierarchy, "hierarchy", "", "Hierarchy type (e.g. HostsAndClusters, VmsAndTemplates)")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum child objects to return (default 200)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort child objects by field (e.g. name)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort child objects descending")
	cmd.Flags().StringSliceVar(&opts.objectTypes, "object-type", nil, "Filter objects by type (e.g. Datacenter, VirtualMachine)")
	cmd.Flags().StringVar(&opts.objectName, "object-name", "", "Filter objects by name (case insensitive substring)")

	return cmd
}

func inventoryVirtualInfraDescribeCmd() *cobra.Command {
	opts := inventoryVirtualDescribeOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "virtualinfra",
		Short: helptext.InventoryVirtualDescribeShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.name) == "" {
				return requireFlag("--name", "provide the server or host name to describe")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req := client.InventoryRequest{}
			if opts.limit > 0 {
				req.Pagination = &client.InventoryPagination{Limit: opts.limit}
			}
			if opts.sort != "" {
				req.Sorting = map[string]any{
					"property":  opts.sort,
					"direction": sortDirection(opts.desc),
				}
			}

			result, err := httpClient.InventoryServers(ctx, req)
			if err != nil {
				return err
			}

			var detail *inventoryVirtualDescribeDetail
			for _, entry := range result.Data {
				if !strings.EqualFold(stringFromMap(entry, "name"), opts.name) {
					continue
				}
				detail = &inventoryVirtualDescribeDetail{
					Name:     stringFromMap(entry, "name"),
					Type:     stringFromMap(entry, "type"),
					Host:     stringFromMap(entry, "hostName"),
					ObjectID: stringFromMap(entry, "objectId"),
					URN:      stringFromMap(entry, "urn"),
					Platform: friendlyPlatform(stringFromMap(entry, "platform")),
					Size:     stringFromMap(entry, "size"),
				}
				break
			}

			if detail == nil {
				return fmt.Errorf("inventory server %q not found", opts.name)
			}
			return output.Print(outputFormat(), detail)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Server or host name to describe")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to inspect (default 200)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort records by field (e.g. name)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort records descending")

	return cmd
}

func inventoryUnstructuredListCmd() *cobra.Command {
	opts := inventoryUnstructuredListOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "unstructured",
		Short: helptext.InventoryUnstructuredListShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.UnstructuredDataServersFilter{
				Skip:  0,
				Limit: opts.limit,
				Name:  opts.name,
			}
			if opts.sort != "" {
				filter.OrderColumn = normalizeUnstructuredOrderColumn(opts.sort)
				asc := !opts.desc
				filter.OrderAscending = &asc
			}

			result, err := httpClient.UnstructuredDataServers(ctx, filter)
			if err != nil {
				return err
			}

			typeFilters := normalizeSlice(opts.types)
			nameFilter := strings.TrimSpace(strings.ToLower(opts.contains))

			rows := make([]inventoryUnstructuredRow, 0, len(result.Data))
			for _, entry := range result.Data {
				rowType := stringFromMap(entry, "type")
				if len(typeFilters) > 0 && !typeFilters[strings.ToLower(rowType)] {
					continue
				}
				name := bestUnstructuredName(entry)
				if nameFilter != "" && !containsInsensitive(name, nameFilter) {
					continue
				}

				rows = append(rows, inventoryUnstructuredRow{
					Name:     name,
					Type:     rowType,
					Location: unstructuredLocation(entry),
					ID:       stringFromMap(entry, "id"),
				})
			}

			return output.Print(outputFormat(), rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Filter servers by name pattern (supports * wildcards)")
	cmd.Flags().StringVar(&opts.contains, "contains", "", "Filter servers whose name contains the provided value (case insensitive)")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "Filter servers by type (FileServer, SMBShare, NFSShare, NASFiler, S3Compatible, AmazonS3, AzureBlob)")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to return (default 200)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (Name or Description)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")

	return cmd
}

func inventoryUnstructuredDescribeCmd() *cobra.Command {
	opts := inventoryUnstructuredDescribeOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "unstructured",
		Short: helptext.InventoryUnstructuredDescribeShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.id) == "" && strings.TrimSpace(opts.name) == "" {
				return requireFlag("--name or --id", "provide the server name or ID to describe")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			var detail map[string]any

			if opts.id != "" {
				detail, err = httpClient.UnstructuredDataServer(ctx, opts.id)
				if err != nil {
					return err
				}
			} else {
				filter := client.UnstructuredDataServersFilter{
					Limit: opts.limit,
					Name:  opts.name,
				}
				result, err := httpClient.UnstructuredDataServers(ctx, filter)
				if err != nil {
					return err
				}

				var match map[string]any
				for _, entry := range result.Data {
					if strings.EqualFold(bestUnstructuredName(entry), opts.name) {
						match = entry
						break
					}
				}
				if match == nil {
					return fmt.Errorf("unstructured data server %q not found", opts.name)
				}

				id := stringFromMap(match, "id")
				if id == "" {
					detail = match
				} else {
					detail, err = httpClient.UnstructuredDataServer(ctx, id)
					if err != nil {
						return err
					}
				}
			}

			row := inventoryUnstructuredDetailRow{
				Name:                bestUnstructuredName(detail),
				Type:                stringFromMap(detail, "type"),
				Location:            unstructuredLocation(detail),
				ID:                  stringFromMap(detail, "id"),
				CredentialsRequired: yesNoString(boolFromMap(detail, "accessCredentialsRequired")),
				CredentialsID:       unstructuredCredentialsID(detail),
				CacheRepositoryID:   unstructuredCacheRepository(detail),
			}

			return output.Print(outputFormat(), row)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Server name to describe")
	cmd.Flags().StringVar(&opts.id, "id", "", "Server ID to describe")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to inspect when resolving by name (default 200)")

	return cmd
}

func inventoryProtectionGroupListCmd() *cobra.Command {
	opts := inventoryProtectionGroupListOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "protectiongroup",
		Short: helptext.InventoryProtectionGroupListShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.ProtectionGroupFilter{
				Name:     opts.name,
				Type:     opts.ptype,
				MaxItems: opts.limit,
			}
			if opts.sort != "" {
				filter.OrderColumn = opts.sort
				asc := !opts.desc
				filter.OrderAscending = &asc
			}

			groups, err := httpClient.ProtectionGroups(ctx, filter)
			if err != nil {
				return err
			}

			rows := make([]inventoryProtectionGroupRow, 0, len(groups))
			for _, group := range groups {
				rows = append(rows, inventoryProtectionGroupRow{
					Name:        group.Name,
					Type:        group.Type,
					Description: group.Description,
				})
			}

			return output.Print(outputFormat(), rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Filter protection groups by name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.ptype, "type", "", "Filter protection groups by type")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to return (default 200)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. name)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")

	cmd.AddCommand(inventoryProtectionGroupAgentsCmd())
	return cmd
}

func inventoryProtectionGroupAgentsCmd() *cobra.Command {
	opts := inventoryProtectionGroupAgentsOptions{limit: 200}

	cmd := &cobra.Command{
		Use:   "agents",
		Short: helptext.InventoryProtectionGroupAgentsShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.groupName) == "" {
				return requireFlag("--name", "provide the protection group name")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			group, err := findProtectionGroupByName(ctx, httpClient, opts.groupName)
			if err != nil {
				return err
			}

			filter := client.DiscoveredEntityFilter{MaxItems: opts.limit}
			entities, err := httpClient.ProtectionGroupDiscoveredEntities(ctx, group.ID, filter)
			if err != nil {
				return err
			}

			rows := make([]inventoryProtectionGroupAgentRow, 0, len(entities))
			for _, entity := range entities {
				ip := ""
				if len(entity.IPAddresses) > 0 {
					ip = entity.IPAddresses[0]
				}
				rows = append(rows, inventoryProtectionGroupAgentRow{
					Group:           group.Name,
					Server:          entity.Name,
					IP:              ip,
					LastSeen:        formatTimePtr(entity.LastConnected),
					HasBackupAgent:  yesNoString(isAgentInstalled(entity.AgentStatus)),
					HasAppPlugin:    yesNoString(isAgentInstalled(entity.DriverStatus)),
					OperatingSystem: entity.OperatingSystem,
				})
			}

			return output.Print(outputFormat(), rows)
		},
	}

	cmd.Flags().StringVar(&opts.groupName, "name", "", "Protection group name (required)")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum agents to return (default 200)")

	return cmd
}

func inventoryProtectionGroupDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protectiongroup",
		Short: helptext.InventoryProtectionGroupDescribeShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(inventoryProtectionGroupDescribeAgentCmd())
	return cmd
}

func inventoryProtectionGroupDescribeAgentCmd() *cobra.Command {
	opts := inventoryProtectionGroupDescribeAgentOptions{}

	cmd := &cobra.Command{
		Use:   "agent",
		Short: helptext.InventoryProtectionGroupAgentDescribeShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.agentName) == "" {
				return requireFlag("--name", "provide the agent name")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			group, entity, err := resolveAgentEntity(ctx, httpClient, opts.groupName, opts.agentName)
			if err != nil {
				return err
			}

			detail, err := httpClient.ProtectionGroupDiscoveredEntity(ctx, group.ID, entity.ID)
			if err != nil {
				return err
			}

			if !strings.EqualFold(detail.Entity.Type, "Computer") {
				return fmt.Errorf("agent %q is of type %q; only Computer entities are supported", detail.Entity.Name, detail.Entity.Type)
			}

			row := inventoryProtectionGroupAgentDetail{
				Group:                  group.Name,
				Name:                   detail.Entity.Name,
				Type:                   detail.Entity.Type,
				State:                  detail.Entity.State,
				AgentStatus:            detail.Entity.AgentStatus,
				AgentVersion:           detail.Entity.AgentVersion,
				DriverStatus:           detail.Entity.DriverStatus,
				DriverVersion:          detail.Entity.DriverVersion,
				RebootRequired:         yesNoString(detail.Entity.RebootRequired),
				IPAddresses:            strings.Join(detail.Entity.IPAddresses, ", "),
				LastSeen:               formatTimePtr(detail.Entity.LastConnected),
				OperatingSystem:        detail.Entity.OperatingSystem,
				OperatingSystemVersion: detail.Entity.OperatingSystemVersion,
				Platform:               detail.Entity.OperatingSystemPlatform,
			}

			return output.Print(outputFormat(), row)
		},
	}

	cmd.Flags().StringVar(&opts.agentName, "name", "", "Agent (entity) name to describe")
	cmd.Flags().StringVar(&opts.groupName, "group", "", "Restrict search to a specific protection group")

	return cmd
}

// --- Option & row structures ---
type inventoryVirtualListOptions struct {
	platforms []string
	types     []string
	name      string
	host      string
	limit     int
	sort      string
	desc      bool
}

type inventoryVirtualObjectsOptions struct {
	name        string
	hierarchy   string
	objectTypes []string
	objectName  string
	limit       int
	sort        string
	desc        bool
}

type inventoryVirtualDescribeOptions struct {
	name  string
	limit int
	sort  string
	desc  bool
}

type inventoryUnstructuredListOptions struct {
	name     string
	contains string
	types    []string
	limit    int
	sort     string
	desc     bool
}

type inventoryUnstructuredDescribeOptions struct {
	name  string
	id    string
	limit int
}

type inventoryVirtualRow struct {
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Host     string `json:"Host"`
	Size     string `json:"Size"`
	Platform string `json:"-"`
}

func (r inventoryVirtualRow) platformKey() string {
	switch strings.ToLower(strings.ReplaceAll(r.Platform, " ", "")) {
	case "vmwarevsphere", "vmware":
		return "vsphere"
	case "microsofthyper-v", "hyper-v", "hyperv":
		return "hyperv"
	case "vmwareclouddirector", "clouddirector":
		return "clouddirector"
	default:
		return strings.ToLower(strings.ReplaceAll(r.Platform, " ", ""))
	}
}

type inventoryVirtualObjectRow struct {
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Host     string `json:"Host"`
	ObjectID string `json:"Object ID"`
	Size     string `json:"Size"`
}

type inventoryVirtualDescribeDetail struct {
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Host     string `json:"Host"`
	ObjectID string `json:"Object ID"`
	URN      string `json:"URN"`
	Platform string `json:"Platform"`
	Size     string `json:"Size"`
}

type inventoryUnstructuredRow struct {
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Location string `json:"Location"`
	ID       string `json:"ID"`
}

type inventoryUnstructuredDetailRow struct {
	Name                string `json:"Name"`
	Type                string `json:"Type"`
	Location            string `json:"Location"`
	ID                  string `json:"ID"`
	CredentialsRequired string `json:"Credentials Required"`
	CredentialsID       string `json:"Credentials ID"`
	CacheRepositoryID   string `json:"Cache Repository"`
}

type inventoryProtectionGroupListOptions struct {
	name  string
	ptype string
	limit int
	sort  string
	desc  bool
}

type inventoryProtectionGroupAgentsOptions struct {
	groupName string
	limit     int
}

type inventoryProtectionGroupDescribeAgentOptions struct {
	groupName string
	agentName string
}

type inventoryProtectionGroupRow struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
}

type inventoryProtectionGroupAgentRow struct {
	Group           string `json:"Protection Group"`
	Server          string `json:"Server"`
	IP              string `json:"IP Address"`
	LastSeen        string `json:"Last Seen"`
	HasBackupAgent  string `json:"Backup Agent"`
	HasAppPlugin    string `json:"Application Plugin"`
	OperatingSystem string `json:"Operating System"`
}

type inventoryProtectionGroupAgentDetail struct {
	Group                  string `json:"Protection Group"`
	Name                   string `json:"Name"`
	Type                   string `json:"Type"`
	State                  string `json:"State"`
	AgentStatus            string `json:"Agent Status"`
	AgentVersion           string `json:"Agent Version"`
	DriverStatus           string `json:"Driver Status"`
	DriverVersion          string `json:"Driver Version"`
	RebootRequired         string `json:"Reboot Required"`
	IPAddresses            string `json:"IP Addresses"`
	LastSeen               string `json:"Last Seen"`
	OperatingSystem        string `json:"Operating System"`
	OperatingSystemVersion string `json:"Operating System Version"`
	Platform               string `json:"Platform"`
}

// --- Helpers ---

func normalizeUnstructuredOrderColumn(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "name":
		return "Name"
	case "description":
		return "Description"
	default:
		return value
	}
}

func bestUnstructuredName(entry map[string]any) string {
	if entry == nil {
		return ""
	}
	if name := stringFromMap(entry, "name"); name != "" {
		return name
	}
	if friendly := stringFromMap(entry, "friendlyName"); friendly != "" {
		return friendly
	}
	if account := mapFromMap(entry, "account"); len(account) > 0 {
		if friendly := stringFromMap(account, "friendlyName"); friendly != "" {
			return friendly
		}
		if bucket := stringFromMap(account, "bucket"); bucket != "" {
			return bucket
		}
	}
	return stringFromMap(entry, "id")
}

func unstructuredLocation(entry map[string]any) string {
	if entry == nil {
		return ""
	}
	if path := stringFromMap(entry, "path"); path != "" {
		return path
	}
	if host := stringFromMap(entry, "hostName"); host != "" {
		return host
	}
	if hostID := stringFromMap(entry, "hostId"); hostID != "" {
		return hostID
	}
	if storageHostID := stringFromMap(entry, "storageHostId"); storageHostID != "" {
		return storageHostID
	}
	if friendly := stringFromMap(entry, "friendlyName"); friendly != "" {
		return friendly
	}
	if account := mapFromMap(entry, "account"); len(account) > 0 {
		if service := stringFromMap(account, "servicePoint"); service != "" {
			return service
		}
		if region := stringFromMap(account, "regionId"); region != "" {
			return region
		}
		if friendly := stringFromMap(account, "friendlyName"); friendly != "" {
			return friendly
		}
	}
	return ""
}

func unstructuredCacheRepository(entry map[string]any) string {
	if entry == nil {
		return ""
	}
	if processing := mapFromMap(entry, "processing"); len(processing) > 0 {
		if repo := stringFromMap(processing, "cacheRepositoryId"); repo != "" {
			return repo
		}
	}
	return ""
}

func unstructuredCredentialsID(entry map[string]any) string {
	if entry == nil {
		return ""
	}
	if id := stringFromMap(entry, "accessCredentialsId"); id != "" {
		return id
	}
	if id := stringFromMap(entry, "credentialsId"); id != "" {
		return id
	}
	if account := mapFromMap(entry, "account"); len(account) > 0 {
		if id := stringFromMap(account, "credentialsId"); id != "" {
			return id
		}
	}
	return ""
}

func boolFromMap(entry map[string]any, key string) bool {
	if entry == nil {
		return false
	}
	value, ok := entry[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		if parsed, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return parsed
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return false
}

func normalizeSlice(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			set[strings.ToLower(trimmed)] = true
		}
	}
	return set
}

func normalizePlatformFilters(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "vsphere", "vmware":
			set["vsphere"] = true
		case "hyperv", "hyper-v":
			set["hyperv"] = true
		case "clouddirector", "cloud-director":
			set["clouddirector"] = true
		default:
			set[strings.ToLower(strings.TrimSpace(v))] = true
		}
	}
	return set
}

func friendlyPlatform(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "vmware":
		return "VMware vSphere"
	case "clouddirector":
		return "VMware Cloud Director"
	case "hyperv", "hyper-v":
		return "Microsoft Hyper-V"
	default:
		return strings.TrimSpace(value)
	}
}

func containsInsensitive(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}

func sortDirection(desc bool) string {
	if desc {
		return "descending"
	}
	return "ascending"
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func mapFromMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if typed, ok := v.(map[string]any); ok {
			return typed
		}
	}
	return nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return formatTimestampValue(*t)
}

func yesNoString(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func isAgentInstalled(status string) bool {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return false
	}
	if strings.Contains(status, "notinstalled") {
		return false
	}
	return true
}

func findProtectionGroupByName(ctx context.Context, httpClient *client.Client, name string) (*client.ProtectionGroup, error) {
	filter := client.ProtectionGroupFilter{
		Name:     name,
		MaxItems: 0,
	}
	groups, err := httpClient.ProtectionGroups(ctx, filter)
	if err != nil {
		return nil, err
	}

	var match *client.ProtectionGroup
	for i := range groups {
		if strings.EqualFold(groups[i].Name, name) {
			if match != nil {
				return nil, fmt.Errorf("multiple protection groups matched %q; refine the name", name)
			}
			match = &groups[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("protection group %q not found", name)
	}
	return match, nil
}

func resolveAgentEntity(ctx context.Context, httpClient *client.Client, groupName, agentName string) (*client.ProtectionGroup, *client.DiscoveredEntity, error) {
	var groups []client.ProtectionGroup
	if strings.TrimSpace(groupName) != "" {
		group, err := findProtectionGroupByName(ctx, httpClient, groupName)
		if err != nil {
			return nil, nil, err
		}
		groups = append(groups, *group)
	} else {
		var err error
		groups, err = httpClient.ProtectionGroups(ctx, client.ProtectionGroupFilter{MaxItems: 0})
		if err != nil {
			return nil, nil, err
		}
	}

	var matchGroup *client.ProtectionGroup
	var matchEntity *client.DiscoveredEntity

	for i := range groups {
		group := &groups[i]
		filter := client.DiscoveredEntityFilter{
			Name:     agentName,
			MaxItems: 200,
		}
		entities, err := httpClient.ProtectionGroupDiscoveredEntities(ctx, group.ID, filter)
		if err != nil {
			return nil, nil, err
		}
		for j := range entities {
			entity := &entities[j]
			if strings.EqualFold(entity.Name, agentName) {
				if matchEntity != nil {
					return nil, nil, fmt.Errorf("multiple agents named %q found; specify --group to disambiguate", agentName)
				}
				matchGroup = group
				match := *entity
				matchEntity = &match
			}
		}
	}

	if matchEntity == nil {
		if groupName != "" {
			return nil, nil, fmt.Errorf("agent %q not found in protection group %q", agentName, groupName)
		}
		return nil, nil, fmt.Errorf("agent %q not found; specify --group to narrow the search", agentName)
	}

	return matchGroup, matchEntity, nil
}
