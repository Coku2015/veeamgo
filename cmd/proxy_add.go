package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

func proxyAddCmd() *cobra.Command {
	opts := struct {
		name                  string
		description           string
		proxyType             string
		managedServer         string
		maxTasks              int
		transportMode         string
		failoverToNetwork     bool
		hostToProxyEncryption bool
		autoSelectDatastores  bool
		wait                  bool
		yes                   bool
	}{}

	cmd := &cobra.Command{
		Use:   cmdProxyUse,
		Short: helptext.ProxyAddShort,
		Long:  helptext.ProxyAddLong,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rawProxyType := strings.TrimSpace(opts.proxyType)
			if rawProxyType == "" {
				rawProxyType = "vmware"
			}
			proxyType, err := normalizeProxyType(rawProxyType)
			if err != nil {
				return err
			}

			hostName := strings.TrimSpace(opts.managedServer)
			if hostName == "" {
				return requireFlag("--managed-server", "provide the managed server host name")
			}

			proxyName := strings.TrimSpace(opts.name)
			if proxyName == "" {
				proxyName = hostName
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

			host, err := resolveManagedServerByName(ctx, httpClient, hostName)
			if err != nil {
				return err
			}

			if err = ensureProxyNotExists(ctx, httpClient, proxyType, proxyName, host.Name); err != nil {
				return err
			}

			description := strings.TrimSpace(opts.description)
			if description == "" {
				description = fmt.Sprintf("Proxy for %s (%s)", host.Name, proxyType)
			}

			spec, err := buildProxySpec(cmd, opts, proxyType, proxyName, description, host)
			if err != nil {
				return err
			}

			if !opts.yes {
				label := proxyTypeDisplay(proxyType)
				if label == "" {
					label = "Proxy"
				}
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %q on host %q", label, proxyName, host.Name))
				if err != nil {
					return err
				}
				if !confirmed {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			session, err := httpClient.CreateProxy(ctx, spec)
			if err != nil {
				return fmt.Errorf("create proxy: %w", err)
			}

			if session != nil && opts.wait {
				final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
				if waitErr != nil {
					return fmt.Errorf("wait for proxy provisioning session: %w", waitErr)
				}
				session = final
			}

			return printProxySession(cmd, session, proxyName, proxyType, host.Name, opts.wait)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Proxy name (defaults to managed server name if omitted)")
	cmd.Flags().StringVar(&opts.description, "description", "", "Proxy description")
	cmd.Flags().StringVar(&opts.proxyType, "type", "vmware", "Proxy platform (vmware, hyperv, general)")
	cmd.Flags().StringVar(&opts.managedServer, "managed-server", "", "Managed server host name already added to inventory")
	cmd.Flags().IntVar(&opts.maxTasks, "max-tasks", 0, "Maximum concurrent tasks handled by the proxy (default: platform default)")
	cmd.Flags().StringVar(&opts.transportMode, "transport-mode", "auto", "Transport mode for VMware proxies (auto, directAccess, virtualAppliance, network)")
	cmd.Flags().BoolVar(&opts.failoverToNetwork, "failover-to-network", false, "Allow fallback to network mode if the primary transport fails (VMware proxies only)")
	cmd.Flags().BoolVar(&opts.hostToProxyEncryption, "host-to-proxy-encryption", false, "Enable TLS encryption between host and proxy in network mode (VMware proxies only)")
	cmd.Flags().BoolVar(&opts.autoSelectDatastores, "auto-select-datastores", false, "Automatically select accessible datastores (VMware proxies only)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to complete")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	return cmd
}

func buildProxySpec(cmd *cobra.Command, opts struct {
	name                  string
	description           string
	proxyType             string
	managedServer         string
	maxTasks              int
	transportMode         string
	failoverToNetwork     bool
	hostToProxyEncryption bool
	autoSelectDatastores  bool
	wait                  bool
	yes                   bool
}, proxyType, proxyName, description string, host *client.ManagedServer) (map[string]any, error) {
	spec := map[string]any{
		"name":        proxyName,
		"description": description,
		"type":        proxyType,
	}

	server := map[string]any{
		"hostId":   host.ID,
		"hostName": host.Name,
	}

	if opts.maxTasks > 0 {
		server["maxTaskCount"] = opts.maxTasks
	}

	switch proxyType {
	case "ViProxy":
		mode, err := normalizeTransportMode(opts.transportMode)
		if err != nil {
			return nil, err
		}
		server["transportMode"] = mode

		if cmd.Flags().Changed("failover-to-network") {
			server["failoverToNetwork"] = opts.failoverToNetwork
		}
		if cmd.Flags().Changed("host-to-proxy-encryption") {
			server["hostToProxyEncryption"] = opts.hostToProxyEncryption
		}
		if cmd.Flags().Changed("auto-select-datastores") {
			server["connectedDatastores"] = map[string]any{
				"autoSelectEnabled": opts.autoSelectDatastores,
			}
		}
	case "HvProxy", "GeneralPurposeProxy":
		// No additional fields required.
	default:
		return nil, fmt.Errorf("proxy type %q is not supported", proxyType)
	}

	spec["server"] = server
	return spec, nil
}

func normalizeProxyType(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "vmware", "vmware proxy", "vi", "viproxy", "vsphere":
		return "ViProxy", nil
	case "hvproxy", "hv", "hyperv", "hyper-v", "hyper v":
		return "HvProxy", nil
	case "generalpurposeproxy", "generalpurpose", "general-purpose", "general purpose", "general", "gp":
		return "GeneralPurposeProxy", nil
	default:
		return "", fmt.Errorf("proxy type %q is not supported; use vmware, hyperv, or general", v)
	}
}

func normalizeTransportMode(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "auto":
		return "auto", nil
	case "directaccess", "direct", "direct_access":
		return "directAccess", nil
	case "virtualappliance", "virtual_appliance", "va":
		return "virtualAppliance", nil
	case "network":
		return "network", nil
	default:
		return "", fmt.Errorf("transport mode %q is not supported; use auto, directAccess, virtualAppliance, or network", v)
	}
}

func ensureProxyNotExists(ctx context.Context, api *client.Client, proxyType, proxyName, hostName string) error {
	proxies, err := api.Proxies(ctx, client.ProxyFilter{
		Type: proxyType,
	})
	if err != nil {
		return err
	}

	for _, proxy := range proxies.Proxies {
		if strings.EqualFold(proxy.Name, proxyName) {
			return fmt.Errorf("proxy %q already exists (type %s)", proxyName, proxy.Type)
		}
		if strings.EqualFold(proxy.HostName, hostName) && strings.EqualFold(proxy.Type, proxyType) {
			return fmt.Errorf("%s proxy for host %q already exists", proxyType, hostName)
		}
	}

	return nil
}

func printProxySession(cmd *cobra.Command, session *client.Session, proxyName, proxyType, hostName string, waited bool) error {
	label := proxyTypeDisplay(proxyType)
	if label == "" {
		label = "Proxy"
	}

	format := outputFormat()
	if format == "json" {
		status := "provisioning started"
		switch {
		case session == nil:
			status = "provisioning completed"
		case waited:
			status = "provisioning finished"
		}
		payload := map[string]any{
			"message": fmt.Sprintf("%s proxy %q on host %q %s", label, proxyName, hostName, status),
			"proxy": map[string]string{
				"name": proxyName,
				"type": proxyType,
				"host": hostName,
			},
		}
		if session != nil {
			payload["session"] = session
		}
		payload["waited"] = waited
		return output.Print(format, payload)
	}

	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s proxy %q on host %q provisioning completed.\n", label, proxyName, hostName)
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s proxy %q on host %q provisioning finished with result %s (session %s).\n", label, proxyName, hostName, result, session.ID)
		if session.ResourceID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Proxy ID: %s\n", session.ResourceID)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started provisioning %s proxy %q on host %q (session %s).\n", label, proxyName, hostName, session.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}
