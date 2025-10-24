package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
)

type repositoryEditOptions struct {
	targetName string
	targetID   string
	typeHint   string

	newName        string
	description    string
	host           string
	path           string
	sharePath      string
	credentialsID  string
	gatewayServers []string
	gatewayAuto    bool

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
	disable        bool

	wait bool
	yes  bool

	// flag tracking
	hostSet           bool
	pathSet           bool
	sharePathSet      bool
	credentialsSet    bool
	gatewayAutoSet    bool
	gatewayServersSet bool
	mountServerSet    bool
	mountCacheSet     bool
	mountVPowerSet    bool
	mountPortSet      bool
	vPowerPortSet     bool
	maxTasksSet       bool
	readWriteLimitSet bool
	fastCloneSet      bool
	immutableDaysSet  bool
	importBackupSet   bool
	importIndexSet    bool
	disableSet        bool
	newNameSet        bool
	descriptionSet    bool
}

func repositoryEditCmd() *cobra.Command {
	opts := repositoryEditOptions{
		gatewayAuto:     false,
		mountVPower:     true,
		mountMountPort:  1058,
		mountVPowerPort: 2049,
	}

	cmd := &cobra.Command{
		Use:   cmdEditUse,
		Short: helptext.RepositoryEditShort,
		Long:  helptext.RepositoryEditLong,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepositoryEdit(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.targetName, "name", "", "Repository name to edit")
	cmd.Flags().StringVar(&opts.targetID, "id", "", "Repository ID to edit")
	cmd.Flags().StringVar(&opts.typeHint, "type", "", "Expected repository type (optional, helps when names are duplicated)")

	cmd.Flags().StringVar(&opts.newName, "new-name", "", "New repository name")
	cmd.Flags().StringVar(&opts.description, "description", "", "New repository description")
	cmd.Flags().StringVar(&opts.host, "host", "", "Managed server host name or ID (WinLocal, LinuxLocal, LinuxHardened)")
	cmd.Flags().StringVar(&opts.path, "path", "", "Local path for storing backups (WinLocal, LinuxLocal, LinuxHardened)")
	cmd.Flags().StringVar(&opts.sharePath, "share-path", "", "UNC or NFS share path (SMB, NFS)")
	cmd.Flags().StringVar(&opts.credentialsID, "credentials-id", "", "Credentials ID for SMB share access")
	cmd.Flags().StringSliceVar(&opts.gatewayServers, "gateway-server", nil, "Gateway server name or ID (repeatable)")
	cmd.Flags().BoolVar(&opts.gatewayAuto, "gateway-auto", false, "Automatically select a gateway server for SMB/NFS repositories")
	cmd.Flags().StringVar(&opts.mountServer, "mount-server", "", "Mount server name or ID")
	cmd.Flags().StringVar(&opts.mountCache, "mount-server-cache", "", "Path for vPower write cache on the mount server")
	cmd.Flags().BoolVar(&opts.mountVPower, "mount-server-vpower", true, "Enable vPower NFS on the mount server")
	cmd.Flags().IntVar(&opts.mountMountPort, "mount-server-mount-port", 1058, "Mount port for vPower NFS")
	cmd.Flags().IntVar(&opts.mountVPowerPort, "mount-server-nfs-port", 2049, "vPower NFS port")
	cmd.Flags().IntVar(&opts.maxTasks, "max-tasks", 0, "Maximum concurrent tasks (0 disables the limit)")
	cmd.Flags().IntVar(&opts.readWriteLimit, "read-write-limit", 0, "Read/write speed limit in MB/s (0 disables the limit)")
	cmd.Flags().BoolVar(&opts.fastClone, "fast-clone", false, "Enable fast cloning on XFS volumes (LinuxLocal, LinuxHardened)")
	cmd.Flags().IntVar(&opts.immutableDays, "immutable-days", 0, "Number of days to keep recent backups immutable (LinuxHardened)")
	cmd.Flags().BoolVar(&opts.importBackup, "import-backup", false, "Search the repository for existing backups")
	cmd.Flags().BoolVar(&opts.importIndex, "import-index", false, "Import guest OS file system index when importing backups")
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Disable the repository (set to true or false)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the repository reconfiguration session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	return cmd
}

func runRepositoryEdit(cmd *cobra.Command, opts *repositoryEditOptions) error {
	opts.hostSet = cmd.Flags().Changed("host")
	opts.pathSet = cmd.Flags().Changed("path")
	opts.sharePathSet = cmd.Flags().Changed("share-path")
	opts.credentialsSet = cmd.Flags().Changed("credentials-id")
	opts.gatewayAutoSet = cmd.Flags().Changed("gateway-auto")
	opts.gatewayServersSet = cmd.Flags().Changed("gateway-server")
	opts.mountServerSet = cmd.Flags().Changed("mount-server")
	opts.mountCacheSet = cmd.Flags().Changed("mount-server-cache")
	opts.mountVPowerSet = cmd.Flags().Changed("mount-server-vpower")
	opts.mountPortSet = cmd.Flags().Changed("mount-server-mount-port")
	opts.vPowerPortSet = cmd.Flags().Changed("mount-server-nfs-port")
	opts.maxTasksSet = cmd.Flags().Changed("max-tasks")
	opts.readWriteLimitSet = cmd.Flags().Changed("read-write-limit")
	opts.fastCloneSet = cmd.Flags().Changed("fast-clone")
	opts.immutableDaysSet = cmd.Flags().Changed("immutable-days")
	opts.importBackupSet = cmd.Flags().Changed("import-backup")
	opts.importIndexSet = cmd.Flags().Changed("import-index")
	opts.disableSet = cmd.Flags().Changed("disable")
	opts.newNameSet = cmd.Flags().Changed("new-name")
	opts.descriptionSet = cmd.Flags().Changed("description")

	cleanedName := strings.TrimSpace(opts.targetName)
	cleanedID := strings.TrimSpace(opts.targetID)
	if cleanedName == "" && cleanedID == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if cleanedName != "" && cleanedID != "" {
		return fmt.Errorf("provide either --name or --id, not both")
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

	target, err := resolveLocalRepositoryTarget(ctx, httpClient, cleanedName, cleanedID, opts.typeHint)
	if err != nil {
		return err
	}

	payload, info, modified, err := buildRepositoryUpdatePayload(ctx, cmd, httpClient, target, opts)
	if err != nil {
		return err
	}

	if !modified {
		location := infoLocation(target.Type, info)
		if location != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q %s left unchanged.\n", repositoryTypeDisplay(target.Type), target.Name, location)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q left unchanged.\n", repositoryTypeDisplay(target.Type), target.Name)
		}
		return nil
	}

	label := repositoryTypeDisplay(target.Type)
	if label == "" {
		label = "Repository"
	}

	repoName := target.Name
	if v, ok := payload["name"].(string); ok && strings.TrimSpace(v) != "" {
		repoName = v
	}

	if !opts.yes {
		targetText := infoDescription(target.Type, repoName, info)
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Update %s %s", label, targetText))
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled updating %s %q.\n", label, target.Name)
			return nil
		}
	}

	session, err := httpClient.UpdateRepository(ctx, target.ID, payload)
	if err != nil {
		return fmt.Errorf("update repository %q (%s): %w", target.Name, target.ID, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for repository reconfiguration session: %w", waitErr)
		}
		session = final
		waited = true
	}

	location := infoLocation(target.Type, info)
	return printRepositoryUpdateResult(cmd, session, target.Type, repoName, location, waited)
}

func buildRepositoryUpdatePayload(ctx context.Context, cmd *cobra.Command, api *client.Client, target *repositoryTarget, opts *repositoryEditOptions) (map[string]any, repositoryCreationResult, bool, error) {
	payload := cloneMap(target.Detail)
	if payload == nil {
		payload = make(map[string]any)
	}

	result := repositoryCreationResult{
		hostName:  target.HostName,
		sharePath: firstNonEmpty(asString(target.Detail["sharePath"]), asString(target.Detail["share_path"])),
	}

	modified := false

	if opts.newNameSet {
		trimmed := strings.TrimSpace(opts.newName)
		if trimmed == "" {
			return nil, result, false, fmt.Errorf("--new-name cannot be empty")
		}
		if !strings.EqualFold(trimmed, firstNonEmpty(asString(payload["name"]), target.Name)) {
			payload["name"] = trimmed
			modified = true
		}
	}

	if opts.descriptionSet {
		payload["description"] = opts.description
		modified = true
	}

	if opts.disableSet {
		payload["isDisabled"] = opts.disable
		modified = true
	}

	if opts.importBackupSet {
		payload["importBackup"] = opts.importBackup
		modified = true
	}
	if opts.importIndexSet {
		payload["importIndex"] = opts.importIndex
		modified = true
	}

	repoType := target.Type
	switch repoType {
	case "WinLocal":
		changed, err := applyWindowsLocalUpdates(ctx, api, payload, target, opts, &result)
		if err != nil {
			return nil, result, false, err
		}
		modified = modified || changed
	case "LinuxLocal":
		changed, err := applyLinuxLocalUpdates(ctx, api, payload, target, opts, &result)
		if err != nil {
			return nil, result, false, err
		}
		modified = modified || changed
	case "LinuxHardened":
		changed, err := applyLinuxHardenedUpdates(ctx, api, payload, target, opts, &result)
		if err != nil {
			return nil, result, false, err
		}
		modified = modified || changed
	case "Smb", "Nfs":
		changed, err := applyNetworkRepositoryUpdates(ctx, api, payload, target, opts, &result)
		if err != nil {
			return nil, result, false, err
		}
		modified = modified || changed
	default:
		return nil, result, false, fmt.Errorf("repository type %s is not supported by this command", repoType)
	}

	return payload, result, modified, nil
}

func applyWindowsLocalUpdates(ctx context.Context, api *client.Client, payload map[string]any, target *repositoryTarget, opts *repositoryEditOptions, info *repositoryCreationResult) (bool, error) {
	return applyLocalRepositoryUpdates(ctx, api, payload, target, opts, info, []string{"WindowsHost", "WindowsServer"}, func(settings map[string]any) {
		// Windows local has no additional fields beyond base local settings.
	})
}

func applyLinuxLocalUpdates(ctx context.Context, api *client.Client, payload map[string]any, target *repositoryTarget, opts *repositoryEditOptions, info *repositoryCreationResult) (bool, error) {
	return applyLocalRepositoryUpdates(ctx, api, payload, target, opts, info, []string{"LinuxHost", "LinuxServer"}, func(settings map[string]any) {
		if opts.fastCloneSet {
			settings["useFastCloningOnXFSVolumes"] = opts.fastClone
		}
	})
}

func applyLinuxHardenedUpdates(ctx context.Context, api *client.Client, payload map[string]any, target *repositoryTarget, opts *repositoryEditOptions, info *repositoryCreationResult) (bool, error) {
	return applyLocalRepositoryUpdates(ctx, api, payload, target, opts, info, []string{"LinuxHost", "LinuxServer"}, func(settings map[string]any) {
		if opts.fastCloneSet {
			settings["useFastCloningOnXFSVolumes"] = opts.fastClone
		}
		if opts.immutableDaysSet {
			settings["makeRecentBackupsImmutableDays"] = opts.immutableDays
		}
	})
}

func applyLocalRepositoryUpdates(ctx context.Context, api *client.Client, payload map[string]any, target *repositoryTarget, opts *repositoryEditOptions, info *repositoryCreationResult, expectedHostTypes []string, customize func(map[string]any)) (bool, error) {
	modified := false

	currentHostID := asString(payload["hostId"])
	currentHostName := target.HostName
	newHostID := currentHostID
	newHostName := currentHostName

	if opts.hostSet {
		host, err := resolveManagedServer(ctx, api, opts.host)
		if err != nil {
			return false, err
		}
		matchesType := false
		for _, allowed := range expectedHostTypes {
			if strings.EqualFold(host.Type, allowed) {
				matchesType = true
				break
			}
		}
		if !matchesType {
			return false, fmt.Errorf("managed server %q is not a supported host for this repository type", host.Name)
		}
		newHostID = host.ID
		newHostName = host.Name
		payload["hostId"] = host.ID
		modified = modified || !strings.EqualFold(currentHostID, host.ID)
	}

	repository := cloneMap(asMap(payload["repository"]))
	if repository == nil {
		repository = make(map[string]any)
	}

	if opts.pathSet {
		repository["path"] = opts.path
		info.path = opts.path
		modified = true
	} else if existingPath := asString(repository["path"]); existingPath != "" {
		info.path = existingPath
	}

	if opts.maxTasksSet {
		repository["taskLimitEnabled"] = opts.maxTasks > 0
		if opts.maxTasks > 0 {
			repository["maxTaskCount"] = opts.maxTasks
		} else {
			delete(repository, "maxTaskCount")
		}
		modified = true
	}

	if opts.readWriteLimitSet {
		repository["readWriteLimitEnabled"] = opts.readWriteLimit > 0
		repository["readWriteRate"] = opts.readWriteLimit
		if opts.readWriteLimit <= 0 {
			delete(repository, "readWriteRate")
		}
		modified = true
	}

	customize(repository)
	payload["repository"] = repository

	currentMount := cloneMap(asMap(payload["mountServer"]))
	if currentMount == nil {
		currentMount = make(map[string]any)
	}

	mountServerID := asString(currentMount["mountServerId"])

	if opts.mountServerSet || opts.mountCacheSet || opts.mountVPowerSet || opts.mountPortSet || opts.vPowerPortSet || (opts.hostSet && mountServerID != "" && strings.EqualFold(mountServerID, currentHostID)) {
		mountRef := opts.mountServer
		if mountRef == "" && opts.hostSet {
			mountRef = newHostID
		} else if mountRef == "" {
			mountRef = mountServerID
		}

		if strings.TrimSpace(mountRef) == "" {
			return false, fmt.Errorf("unable to determine mount server - provide --mount-server")
		}

		mountServer, err := resolveManagedServer(ctx, api, mountRef)
		if err != nil {
			return false, err
		}
		cachePath := opts.mountCache
		if strings.TrimSpace(cachePath) == "" {
			cachePath = defaultMountCachePath(mountServer.Type)
		}

		currentMount["mountServerId"] = mountServer.ID
		currentMount["writeCacheFolder"] = cachePath
		if opts.mountVPowerSet {
			currentMount["vPowerNFSEnabled"] = opts.mountVPower
		}

		portSettings := cloneMap(asMap(currentMount["vPowerNFSPortSettings"]))
		if portSettings == nil {
			portSettings = make(map[string]any)
		}
		if opts.mountPortSet {
			portSettings["mountPort"] = opts.mountMountPort
		}
		if opts.vPowerPortSet {
			portSettings["vPowerNFSPort"] = opts.mountVPowerPort
		}
		if len(portSettings) > 0 {
			currentMount["vPowerNFSPortSettings"] = portSettings
		}

		payload["mountServer"] = currentMount
		modified = true
	}

	if opts.hostSet {
		info.hostName = newHostName
	} else if info.hostName == "" {
		info.hostName = newHostName
	}

	if info.path == "" {
		info.path = asString(repository["path"])
	}

	return modified, nil
}

func applyNetworkRepositoryUpdates(ctx context.Context, api *client.Client, payload map[string]any, target *repositoryTarget, opts *repositoryEditOptions, info *repositoryCreationResult) (bool, error) {
	modified := false

	repository := cloneMap(asMap(payload["repository"]))
	if repository == nil {
		repository = make(map[string]any)
	}
	if opts.maxTasksSet {
		repository["taskLimitEnabled"] = opts.maxTasks > 0
		if opts.maxTasks > 0 {
			repository["maxTaskCount"] = opts.maxTasks
		} else {
			delete(repository, "maxTaskCount")
		}
		modified = true
	}
	if opts.readWriteLimitSet {
		repository["readWriteLimitEnabled"] = opts.readWriteLimit > 0
		repository["readWriteRate"] = opts.readWriteLimit
		if opts.readWriteLimit <= 0 {
			delete(repository, "readWriteRate")
		}
		modified = true
	}
	payload["repository"] = repository

	share := cloneMap(asMap(payload["share"]))
	if share == nil {
		share = make(map[string]any)
	}

	if opts.sharePathSet {
		share["sharePath"] = opts.sharePath
		info.sharePath = opts.sharePath
		modified = true
	} else if v := asString(share["sharePath"]); v != "" {
		info.sharePath = v
	}

	if target.Type == "Smb" && opts.credentialsSet {
		share["credentialsId"] = strings.TrimSpace(opts.credentialsID)
		modified = true
	}

	if opts.gatewayAutoSet || opts.gatewayServersSet {
		gateway, err := buildGatewaySettings(ctx, api, &repositoryAddOptions{
			repoType:       target.Type,
			gatewayAuto:    opts.gatewayAuto,
			gatewayAutoSet: opts.gatewayAutoSet,
			gatewayServers: opts.gatewayServers,
		})
		if err != nil {
			return false, err
		}
		if gateway != nil {
			share["gatewayServer"] = gateway
		}
		modified = true
	}
	payload["share"] = share

	currentMount := cloneMap(asMap(payload["mountServer"]))
	if currentMount == nil {
		currentMount = make(map[string]any)
	}

	if opts.mountServerSet || opts.mountCacheSet || opts.mountVPowerSet || opts.mountPortSet || opts.vPowerPortSet {
		mountServerRef := opts.mountServer
		if strings.TrimSpace(mountServerRef) == "" {
			mountServerRef = asString(currentMount["mountServerId"])
		}
		if strings.TrimSpace(mountServerRef) == "" {
			return false, fmt.Errorf("provide --mount-server to update mount server settings")
		}

		mountServer, err := resolveManagedServer(ctx, api, mountServerRef)
		if err != nil {
			return false, err
		}
		cachePath := opts.mountCache
		if strings.TrimSpace(cachePath) == "" {
			cachePath = defaultMountCachePath(mountServer.Type)
		}

		currentMount["mountServerId"] = mountServer.ID
		currentMount["writeCacheFolder"] = cachePath
		if opts.mountVPowerSet {
			currentMount["vPowerNFSEnabled"] = opts.mountVPower
		}
		portSettings := cloneMap(asMap(currentMount["vPowerNFSPortSettings"]))
		if portSettings == nil {
			portSettings = make(map[string]any)
		}
		if opts.mountPortSet {
			portSettings["mountPort"] = opts.mountMountPort
		}
		if opts.vPowerPortSet {
			portSettings["vPowerNFSPort"] = opts.mountVPowerPort
		}
		if len(portSettings) > 0 {
			currentMount["vPowerNFSPortSettings"] = portSettings
		}
		payload["mountServer"] = currentMount
		modified = true
	}

	if info.sharePath == "" {
		info.sharePath = asString(share["sharePath"])
	}

	return modified, nil
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
