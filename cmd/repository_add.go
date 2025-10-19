package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
)

type repositoryAddOptions struct {
	name        string
	description string
	repoType    string

	host            string
	path            string
	sharePath       string
	credentialsID   string
	gatewayServers  []string
	mountServer     string
	mountCache      string
	mountVPower     bool
	mountMountPort  int
	mountVPowerPort int

	maxTasks       int
	readWriteLimit int
	fastClone      bool
	immutableDays  int
	importBackup   bool
	importIndex    bool
	disabled       bool
	gatewayAuto    bool

	wait bool
	yes  bool

	// flag tracking
	importBackupSet   bool
	importIndexSet    bool
	readWriteLimitSet bool
	maxTasksSet       bool
	fastCloneSet      bool
	immutableDaysSet  bool
	gatewayAutoSet    bool
	mountVPowerSet    bool
	mountPortSet      bool
	vPowerPortSet     bool
}

func repositoryAddCmd() *cobra.Command {
	opts := repositoryAddOptions{
		gatewayAuto:     true,
		mountVPower:     true,
		mountMountPort:  1058,
		mountVPowerPort: 2049,
	}

	cmd := &cobra.Command{
		Use:   cmdAddUse,
		Short: "Add a backup repository",
		Long:  "Creates a new backup repository. Supported types: WinLocal, LinuxLocal, LinuxHardened, SMB, NFS.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryAdd(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Repository name")
	cmd.Flags().StringVar(&opts.description, "description", "", "Repository description")
	cmd.Flags().StringVar(&opts.repoType, "type", "", "Repository type (WinLocal, LinuxLocal, LinuxHardened, SMB, NFS)")
	cmd.Flags().StringVar(&opts.host, "host", "", "Managed server host name or ID (WinLocal, LinuxLocal, LinuxHardened)")
	cmd.Flags().StringVar(&opts.path, "path", "", "Local path for storing backups (WinLocal, LinuxLocal, LinuxHardened)")
	cmd.Flags().StringVar(&opts.sharePath, "share-path", "", "UNC or NFS share path (SMB, NFS)")
	cmd.Flags().StringVar(&opts.credentialsID, "credentials-id", "", "Credentials ID for SMB share access")
	cmd.Flags().StringSliceVar(&opts.gatewayServers, "gateway-server", nil, "Gateway server name or ID (repeatable)")
	cmd.Flags().BoolVar(&opts.gatewayAuto, "gateway-auto", true, "Automatically select a gateway server for SMB/NFS repositories")
	cmd.Flags().StringVar(&opts.mountServer, "mount-server", "", "Mount server name or ID (defaults to repository host where applicable)")
	cmd.Flags().StringVar(&opts.mountCache, "mount-server-cache", "", "Path for vPower write cache on the mount server")
	cmd.Flags().BoolVar(&opts.mountVPower, "mount-server-vpower", true, "Enable vPower NFS on the mount server")
	cmd.Flags().IntVar(&opts.mountMountPort, "mount-server-mount-port", 1058, "Mount port for vPower NFS")
	cmd.Flags().IntVar(&opts.mountVPowerPort, "mount-server-nfs-port", 2049, "vPower NFS port")
	cmd.Flags().IntVar(&opts.maxTasks, "max-tasks", 0, "Maximum concurrent tasks (0 disables the limit)")
	cmd.Flags().IntVar(&opts.readWriteLimit, "read-write-limit", 0, "Read/write speed limit in MB/s (0 disables the limit)")
	cmd.Flags().BoolVar(&opts.fastClone, "fast-clone", false, "Enable fast cloning on XFS volumes (LinuxLocal, LinuxHardened)")
	cmd.Flags().IntVar(&opts.immutableDays, "immutable-days", 0, "Number of days to keep recent backups immutable (LinuxHardened)")
	cmd.Flags().BoolVar(&opts.importBackup, "import-backup", false, "Search the repository for existing backups after creation")
	cmd.Flags().BoolVar(&opts.importIndex, "import-index", false, "Import guest OS file system index when importing backups")
	cmd.Flags().BoolVar(&opts.disabled, "disable", false, "Create the repository in a disabled state")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the repository provisioning session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	return cmd
}

type repositoryCreationResult struct {
	hostName     string
	sharePath    string
	path         string
	objectTarget string
}

func runRepositoryAdd(cmd *cobra.Command, opts *repositoryAddOptions) error {
	opts.importBackupSet = cmd.Flags().Changed("import-backup")
	opts.importIndexSet = cmd.Flags().Changed("import-index")
	opts.readWriteLimitSet = cmd.Flags().Changed("read-write-limit")
	opts.maxTasksSet = cmd.Flags().Changed("max-tasks")
	opts.fastCloneSet = cmd.Flags().Changed("fast-clone")
	opts.immutableDaysSet = cmd.Flags().Changed("immutable-days")
	opts.gatewayAutoSet = cmd.Flags().Changed("gateway-auto")
	opts.mountVPowerSet = cmd.Flags().Changed("mount-server-vpower")
	opts.mountPortSet = cmd.Flags().Changed("mount-server-mount-port")
	opts.vPowerPortSet = cmd.Flags().Changed("mount-server-nfs-port")

	opts.name = strings.TrimSpace(opts.name)
	if opts.name == "" {
		return requireFlag("--name", "provide the repository name")
	}

	repoType, err := normalizeLocalRepositoryType(opts.repoType)
	if err != nil {
		return err
	}
	opts.repoType = repoType

	switch repoType {
	case "WinLocal", "LinuxLocal", "LinuxHardened":
		if strings.TrimSpace(opts.path) == "" {
			return requireFlag("--path", "provide the repository path")
		}
		if strings.TrimSpace(opts.host) == "" {
			return requireFlag("--host", "provide the managed server host name or ID")
		}
		if strings.TrimSpace(opts.mountServer) == "" {
			opts.mountServer = opts.host
		}
	case "Smb", "Nfs":
		if strings.TrimSpace(opts.sharePath) == "" {
			return requireFlag("--share-path", "provide the share path")
		}
		if repoType == "Smb" && strings.TrimSpace(opts.credentialsID) == "" {
			return requireFlag("--credentials-id", "provide the credentials ID for the SMB share")
		}
		if strings.TrimSpace(opts.mountServer) == "" {
			return requireFlag("--mount-server", "provide the mount server name or ID for SMB/NFS repositories")
		}
	default:
		return fmt.Errorf("repository type %s is not supported", repoType)
	}

	timeout := 5 * time.Minute
	if opts.wait {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	spec, info, err := buildRepositorySpec(ctx, httpClient, opts)
	if err != nil {
		return err
	}

	label := repositoryTypeDisplay(opts.repoType)
	if label == "" {
		label = "Repository"
	}

	if !opts.yes {
		target := infoDescription(opts.repoType, opts.name, info)
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %s", label, target))
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled.\n")
			return nil
		}
	}

	session, err := httpClient.CreateRepository(ctx, spec)
	if err != nil {
		return fmt.Errorf("create repository %q: %w", opts.name, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for repository provisioning session: %w", waitErr)
		}
		session = final
		waited = true
	}

	return printRepositoryProvisioningResult(cmd, session, opts.repoType, opts.name, infoLocation(opts.repoType, info), waited)
}

func infoDescription(repoType, name string, info repositoryCreationResult) string {
	switch {
	case isLocalRepositoryType(repoType):
		switch repoType {
		case "WinLocal", "LinuxLocal", "LinuxHardened":
			return fmt.Sprintf("%q on host %q (path %s)", name, info.hostName, info.path)
		case "Smb":
			return fmt.Sprintf("%q for share %q", name, info.sharePath)
		case "Nfs":
			return fmt.Sprintf("%q for NFS export %q", name, info.sharePath)
		}
	case isObjectRepositoryType(repoType):
		if info.objectTarget != "" {
			return fmt.Sprintf("%q targeting %s", name, info.objectTarget)
		}
	}
	return fmt.Sprintf("%q", name)
}

func infoLocation(repoType string, info repositoryCreationResult) string {
	switch {
	case isLocalRepositoryType(repoType):
		switch repoType {
		case "WinLocal", "LinuxLocal", "LinuxHardened":
			return fmt.Sprintf("on host %q", info.hostName)
		case "Smb":
			return fmt.Sprintf("for share %q", info.sharePath)
		case "Nfs":
			return fmt.Sprintf("for export %q", info.sharePath)
		}
	case isObjectRepositoryType(repoType):
		if info.objectTarget != "" {
			return fmt.Sprintf("target %s", info.objectTarget)
		}
	}
	return ""
}

func buildRepositorySpec(ctx context.Context, api *client.Client, opts *repositoryAddOptions) (map[string]any, repositoryCreationResult, error) {
	spec := map[string]any{
		"name":        opts.name,
		"description": opts.description,
		"type":        opts.repoType,
		"isDisabled":  opts.disabled,
	}
	if opts.importBackupSet {
		spec["importBackup"] = opts.importBackup
	}
	if opts.importIndexSet {
		spec["importIndex"] = opts.importIndex
	}

	result := repositoryCreationResult{}

	switch opts.repoType {
	case "WinLocal":
		host, err := resolveManagedServer(ctx, api, opts.host)
		if err != nil {
			return nil, result, err
		}
		if !strings.EqualFold(host.Type, "WindowsHost") && !strings.EqualFold(host.Type, "WindowsServer") {
			return nil, result, fmt.Errorf("host %q is not a Windows managed server", host.Name)
		}
		spec["hostId"] = host.ID
		result.hostName = host.Name
		result.path = opts.path

		repository := buildLocalRepositorySettings(opts, host.Type)
		spec["repository"] = repository

		mount, err := buildMountServerSettings(ctx, api, opts, host.Type)
		if err != nil {
			return nil, result, err
		}
		spec["mountServer"] = mount

	case "LinuxLocal":
		host, err := resolveManagedServer(ctx, api, opts.host)
		if err != nil {
			return nil, result, err
		}
		if !strings.Contains(strings.ToLower(host.Type), "linux") {
			return nil, result, fmt.Errorf("host %q is not a Linux managed server", host.Name)
		}
		spec["hostId"] = host.ID
		result.hostName = host.Name
		result.path = opts.path

		repository := buildLinuxLocalRepositorySettings(opts)
		spec["repository"] = repository

		mount, err := buildMountServerSettings(ctx, api, opts, host.Type)
		if err != nil {
			return nil, result, err
		}
		spec["mountServer"] = mount

	case "LinuxHardened":
		host, err := resolveManagedServer(ctx, api, opts.host)
		if err != nil {
			return nil, result, err
		}
		if !strings.Contains(strings.ToLower(host.Type), "linux") {
			return nil, result, fmt.Errorf("host %q is not a Linux managed server", host.Name)
		}
		spec["hostId"] = host.ID
		result.hostName = host.Name
		result.path = opts.path

		repository := buildLinuxHardenedRepositorySettings(opts)
		spec["repository"] = repository

		mount, err := buildMountServerSettings(ctx, api, opts, host.Type)
		if err != nil {
			return nil, result, err
		}
		spec["mountServer"] = mount

	case "Smb", "Nfs":
		repository := buildNetworkRepositorySettings(opts)
		spec["repository"] = repository

		share, err := buildShareSettings(ctx, api, opts)
		if err != nil {
			return nil, result, err
		}
		spec["share"] = share
		result.sharePath = opts.sharePath

		mount, err := buildMountServerSettings(ctx, api, opts, "")
		if err != nil {
			return nil, result, err
		}
		spec["mountServer"] = mount

	default:
		return nil, result, fmt.Errorf("repository type %s is not supported", opts.repoType)
	}

	return spec, result, nil
}

func buildLocalRepositorySettings(opts *repositoryAddOptions, hostType string) map[string]any {
	settings := map[string]any{
		"path": opts.path,
	}

	if opts.maxTasksSet {
		settings["taskLimitEnabled"] = opts.maxTasks > 0
		if opts.maxTasks > 0 {
			settings["maxTaskCount"] = opts.maxTasks
		}
	} else {
		settings["taskLimitEnabled"] = false
	}

	if opts.readWriteLimitSet {
		settings["readWriteLimitEnabled"] = opts.readWriteLimit > 0
		settings["readWriteRate"] = opts.readWriteLimit
	} else {
		settings["readWriteLimitEnabled"] = false
	}

	return settings
}

func buildLinuxLocalRepositorySettings(opts *repositoryAddOptions) map[string]any {
	settings := buildLocalRepositorySettings(opts, "Linux")
	if opts.fastCloneSet {
		settings["useFastCloningOnXFSVolumes"] = opts.fastClone
	}
	return settings
}

func buildLinuxHardenedRepositorySettings(opts *repositoryAddOptions) map[string]any {
	settings := buildLinuxLocalRepositorySettings(opts)
	if opts.immutableDaysSet {
		settings["makeRecentBackupsImmutableDays"] = opts.immutableDays
	}
	return settings
}

func buildNetworkRepositorySettings(opts *repositoryAddOptions) map[string]any {
	settings := map[string]any{}
	if opts.maxTasksSet {
		settings["taskLimitEnabled"] = opts.maxTasks > 0
		if opts.maxTasks > 0 {
			settings["maxTaskCount"] = opts.maxTasks
		}
	} else {
		settings["taskLimitEnabled"] = false
	}

	if opts.readWriteLimitSet {
		settings["readWriteLimitEnabled"] = opts.readWriteLimit > 0
		settings["readWriteRate"] = opts.readWriteLimit
	} else {
		settings["readWriteLimitEnabled"] = false
	}
	return settings
}

func buildShareSettings(ctx context.Context, api *client.Client, opts *repositoryAddOptions) (map[string]any, error) {
	share := map[string]any{
		"sharePath": opts.sharePath,
	}

	if opts.repoType == "Smb" {
		share["credentialsId"] = strings.TrimSpace(opts.credentialsID)
	}

	gateway, err := buildGatewaySettings(ctx, api, opts)
	if err != nil {
		return nil, err
	}
	if len(gateway) > 0 {
		share["gatewayServer"] = gateway
	}

	return share, nil
}

func buildGatewaySettings(ctx context.Context, api *client.Client, opts *repositoryAddOptions) (map[string]any, error) {
	if opts.repoType != "Smb" && opts.repoType != "Nfs" {
		return nil, nil
	}

	gateway := map[string]any{}
	autoSelect := opts.gatewayAuto

	if len(opts.gatewayServers) > 0 {
		if opts.gatewayAutoSet && autoSelect {
			return nil, fmt.Errorf("--gateway-auto=true cannot be combined with --gateway-server")
		}
		autoSelect = false
		ids := make([]string, 0, len(opts.gatewayServers))
		for _, ref := range opts.gatewayServers {
			server, err := resolveManagedServer(ctx, api, ref)
			if err != nil {
				return nil, err
			}
			ids = append(ids, server.ID)
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("provide at least one gateway server or set --gateway-auto=true")
		}
		gateway["gatewayServerIds"] = ids
	}

	if !autoSelect && len(opts.gatewayServers) == 0 && opts.gatewayAutoSet {
		return nil, fmt.Errorf("provide --gateway-server when --gateway-auto=false")
	}

	gateway["autoSelectEnabled"] = autoSelect
	return gateway, nil
}

func buildMountServerSettings(ctx context.Context, api *client.Client, opts *repositoryAddOptions, repoHostType string) (map[string]any, error) {
	server, err := resolveManagedServer(ctx, api, opts.mountServer)
	if err != nil {
		return nil, err
	}

	cache := strings.TrimSpace(opts.mountCache)
	if cache == "" {
		cache = defaultMountCachePath(server.Type)
	}

	mount := map[string]any{
		"mountServerId":    server.ID,
		"writeCacheFolder": cache,
		"vPowerNFSEnabled": opts.mountVPower,
	}

	portSettings := map[string]any{}
	if opts.mountPortSet || opts.mountMountPort != 1058 {
		portSettings["mountPort"] = opts.mountMountPort
	}
	if opts.vPowerPortSet || opts.mountVPowerPort != 2049 {
		portSettings["vPowerNFSPort"] = opts.mountVPowerPort
	}
	if len(portSettings) > 0 {
		mount["vPowerNFSPortSettings"] = portSettings
	}

	return mount, nil
}
