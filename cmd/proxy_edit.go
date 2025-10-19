package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

type proxyEditOptions struct {
	targetName             string
	targetID               string
	targetType             string
	assumeYes              bool
	wait                   bool
	newName                string
	description            string
	managedServer          string
	maxTasks               int
	transportMode          string
	failoverToNetwork      bool
	hostToProxyEncryption  bool
	autoSelectDatastores   bool
	noAutoSelectDatastores bool
}

func proxyEditCmd() *cobra.Command {
	opts := proxyEditOptions{}

	cmd := &cobra.Command{
		Use:   cmdProxyUse,
		Short: "Edit a backup proxy",
		Long:  "Updates mutable backup proxy settings such as name, description, host, and transport configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxyEdit(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.targetName, "name", "", "Proxy name to edit (optional)")
	cmd.Flags().StringVar(&opts.targetID, "id", "", "Proxy ID to edit (optional)")
	cmd.Flags().StringVar(&opts.targetType, "type", "", "Proxy type to disambiguate by name (optional)")
	cmd.Flags().BoolVar(&opts.assumeYes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the proxy reconfiguration session to finish")

	cmd.Flags().StringVar(&opts.newName, "new-name", "", "New proxy name")
	cmd.Flags().StringVar(&opts.description, "description", "", "Proxy description")
	cmd.Flags().StringVar(&opts.managedServer, "managed-server", "", "Managed server host name to associate with the proxy")
	cmd.Flags().IntVar(&opts.maxTasks, "max-tasks", 0, "Maximum concurrent tasks handled by the proxy")
	cmd.Flags().StringVar(&opts.transportMode, "transport-mode", "", "Transport mode for ViProxy (auto, directAccess, virtualAppliance, network)")
	cmd.Flags().BoolVar(&opts.failoverToNetwork, "failover-to-network", false, "Allow fallback to network mode if the primary transport fails (ViProxy only)")
	cmd.Flags().BoolVar(&opts.hostToProxyEncryption, "host-to-proxy-encryption", false, "Enable TLS encryption between host and proxy in network mode (ViProxy only)")
	cmd.Flags().BoolVar(&opts.autoSelectDatastores, "auto-select-datastores", false, "Automatically select accessible datastores (ViProxy only)")
	cmd.Flags().BoolVar(&opts.noAutoSelectDatastores, "no-auto-select-datastores", false, "Disable automatic datastore selection (ViProxy only)")

	return cmd
}

func runProxyEdit(cmd *cobra.Command, opts proxyEditOptions) error {
	nameTrimmed := strings.TrimSpace(opts.targetName)
	idTrimmed := strings.TrimSpace(opts.targetID)

	if nameTrimmed == "" && idTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if nameTrimmed != "" && idTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	if cmd.Flags().Changed("auto-select-datastores") && cmd.Flags().Changed("no-auto-select-datastores") {
		return fmt.Errorf("--auto-select-datastores and --no-auto-select-datastores are mutually exclusive")
	}

	if cmd.Flags().Changed("new-name") && strings.TrimSpace(opts.newName) == "" {
		return fmt.Errorf("--new-name cannot be empty")
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

	target, err := resolveProxyTarget(ctx, httpClient, nameTrimmed, idTrimmed, opts.targetType)
	if err != nil {
		return err
	}

	detail := target.Detail
	if detail == nil {
		detail, err = httpClient.ProxyDetail(ctx, target.ID)
		if err != nil {
			return err
		}
		target.Detail = detail
	}

	updatedPayload, modified, err := buildProxyUpdatePayload(ctx, cmd, opts, target, httpClient)
	if err != nil {
		return err
	}

	if !modified {
		label := proxyTypeDisplay(target.Type)
		if label == "" {
			label = "Proxy"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No changes supplied; %s %q left unchanged.\n", label, target.Name)
		return nil
	}

	session, err := httpClient.UpdateProxy(ctx, target.ID, updatedPayload)
	if err != nil {
		return fmt.Errorf("update proxy %q (%s): %w", target.Name, target.ID, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for proxy configuration session: %w", waitErr)
		}
		session = final
		waited = true
	}

	return printProxyUpdateResult(cmd, session, target.Name, target.Type, target.HostName, waited)
}

func buildProxyUpdatePayload(ctx context.Context, cmd *cobra.Command, opts proxyEditOptions, target *proxyTarget, api *client.Client) (map[string]any, bool, error) {
	detail := target.Detail
	if detail == nil {
		return nil, false, fmt.Errorf("missing proxy detail payload")
	}

	base := cloneMap(detail.Proxy.Raw)
	if base == nil {
		base = make(map[string]any)
	}

	modified := false

	name := detail.Proxy.Name
	if cmd.Flags().Changed("new-name") {
		trimmed := strings.TrimSpace(opts.newName)
		if trimmed != name {
			name = trimmed
			modified = true
		}
	}
	base["name"] = name

	description := detail.Proxy.Description
	if cmd.Flags().Changed("description") {
		description = opts.description
		if description != detail.Proxy.Description {
			modified = true
		}
	}
	base["description"] = description

	proxyType := detail.Proxy.Type
	base["type"] = proxyType
	base["id"] = target.ID

	serverRaw := make(map[string]any)
	if existing, ok := base["server"].(map[string]any); ok {
		serverRaw = cloneMap(existing)
	}

	hostID := detail.Proxy.HostID
	hostName := firstNonEmpty(detail.Proxy.HostName, target.HostName)
	if cmd.Flags().Changed("managed-server") {
		managed := strings.TrimSpace(opts.managedServer)
		if managed == "" {
			return nil, false, fmt.Errorf("--managed-server cannot be empty")
		}
		host, err := resolveManagedServerByName(ctx, api, managed)
		if err != nil {
			return nil, false, err
		}
		if host.ID != "" && host.ID != hostID {
			hostID = host.ID
			modified = true
		}
		if host.Name != "" && host.Name != hostName {
			hostName = host.Name
			modified = true
		}
	}
	serverRaw["hostId"] = hostID
	serverRaw["hostName"] = hostName

	maxTasks := detail.Proxy.MaxTaskCount
	if cmd.Flags().Changed("max-tasks") {
		maxTasks = opts.maxTasks
		if maxTasks != detail.Proxy.MaxTaskCount {
			modified = true
		}
	}
	if maxTasks > 0 {
		serverRaw["maxTaskCount"] = maxTasks
	} else {
		delete(serverRaw, "maxTaskCount")
	}

	if proxyType == "ViProxy" {
		if cmd.Flags().Changed("transport-mode") {
			mode, err := normalizeTransportMode(opts.transportMode)
			if err != nil {
				return nil, false, err
			}
			if mode != detail.Proxy.TransportMode {
				serverRaw["transportMode"] = mode
				modified = true
			}
		} else if existingMode := strings.TrimSpace(detail.Proxy.TransportMode); existingMode != "" {
			serverRaw["transportMode"] = existingMode
		}

		if cmd.Flags().Changed("failover-to-network") {
			serverRaw["failoverToNetwork"] = opts.failoverToNetwork
			if opts.failoverToNetwork != detail.Proxy.FailoverToNetwork {
				modified = true
			}
		} else if detail.Proxy.FailoverToNetwork {
			serverRaw["failoverToNetwork"] = detail.Proxy.FailoverToNetwork
		} else {
			delete(serverRaw, "failoverToNetwork")
		}

		if cmd.Flags().Changed("host-to-proxy-encryption") {
			serverRaw["hostToProxyEncryption"] = opts.hostToProxyEncryption
			if opts.hostToProxyEncryption != detail.Proxy.HostToProxyEncryption {
				modified = true
			}
		} else if detail.Proxy.HostToProxyEncryption {
			serverRaw["hostToProxyEncryption"] = detail.Proxy.HostToProxyEncryption
		} else {
			delete(serverRaw, "hostToProxyEncryption")
		}

		if cmd.Flags().Changed("auto-select-datastores") {
			connected := cloneMap(asMap(serverRaw["connectedDatastores"]))
			connected["autoSelectEnabled"] = true
			serverRaw["connectedDatastores"] = connected
			if !detail.Proxy.AutoSelectDatastores {
				modified = true
			}
		} else if cmd.Flags().Changed("no-auto-select-datastores") {
			connected := cloneMap(asMap(serverRaw["connectedDatastores"]))
			connected["autoSelectEnabled"] = false
			serverRaw["connectedDatastores"] = connected
			if detail.Proxy.AutoSelectDatastores {
				modified = true
			}
		} else if existing := asMap(serverRaw["connectedDatastores"]); existing != nil {
			serverRaw["connectedDatastores"] = existing
		}
	}

	base["server"] = serverRaw

	if !modified {
		return base, false, nil
	}

	target.Name = name
	target.HostName = hostName
	target.Detail = detail

	return base, true, nil
}

func asMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return cloneMap(m)
	}
	return nil
}

func printProxyUpdateResult(cmd *cobra.Command, session *client.Session, proxyName, proxyType, hostName string, waited bool) error {
	label := proxyTypeDisplay(proxyType)
	if label == "" {
		label = "Proxy"
	}

	format := outputFormat()
	if format == "json" {
		status := "update started"
		switch {
		case session == nil:
			status = "update completed"
		case waited:
			status = "update finished"
		}
		payload := map[string]any{
			"message": fmt.Sprintf("%s %q on host %q %s", label, proxyName, hostName, status),
			"proxy": map[string]string{
				"name": proxyName,
				"type": proxyType,
				"host": hostName,
			},
			"waited": waited,
		}
		if session != nil {
			payload["session"] = session
		}
		return output.Print(format, payload)
	}

	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q on host %q update completed.\n", label, proxyName, hostName)
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q on host %q update finished with result %s (session %s).\n", label, proxyName, hostName, result, session.ID)
		if session.ResourceID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Proxy ID: %s\n", session.ResourceID)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started updating %s %q on host %q (session %s).\n", label, proxyName, hostName, session.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}
