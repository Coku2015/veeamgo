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

func proxyEnableCmd() *cobra.Command {
	var (
		proxyName string
		proxyID   string
		proxyType string
		yes       bool
	)

	cmd := &cobra.Command{
		Use:   cmdProxyUse,
		Short: helptext.ProxyEnableShort,
		Long:  helptext.ProxyEnableLong,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxyToggle(cmd, proxyName, proxyID, proxyType, yes, false, "Enable", "Enabled", func(ctx context.Context, api *client.Client, id string) error {
				return api.EnableProxy(ctx, id)
			})
		},
	}

	cmd.Flags().StringVar(&proxyName, "name", "", "Proxy name to enable (optional)")
	cmd.Flags().StringVar(&proxyID, "id", "", "Proxy ID to enable (optional)")
	cmd.Flags().StringVar(&proxyType, "type", "", "Proxy platform to disambiguate by name (vmware, hyperv, general)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func proxyDisableCmd() *cobra.Command {
	var (
		proxyName string
		proxyID   string
		proxyType string
		yes       bool
	)

	cmd := &cobra.Command{
		Use:   cmdProxyUse,
		Short: helptext.ProxyDisableShort,
		Long:  helptext.ProxyDisableLong,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxyToggle(cmd, proxyName, proxyID, proxyType, yes, true, "Disable", "Disabled", func(ctx context.Context, api *client.Client, id string) error {
				return api.DisableProxy(ctx, id)
			})
		},
	}

	cmd.Flags().StringVar(&proxyName, "name", "", "Proxy name to disable (optional)")
	cmd.Flags().StringVar(&proxyID, "id", "", "Proxy ID to disable (optional)")
	cmd.Flags().StringVar(&proxyType, "type", "", "Proxy platform to disambiguate by name (vmware, hyperv, general)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func runProxyToggle(cmd *cobra.Command, proxyName, proxyID, proxyType string, assumeYes bool, desiredDisabled bool, action, actionPast string, apiCall func(context.Context, *client.Client, string) error) error {
	nameTrimmed := strings.TrimSpace(proxyName)
	idTrimmed := strings.TrimSpace(proxyID)
	typeTrimmed := strings.TrimSpace(proxyType)

	if nameTrimmed == "" && idTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if nameTrimmed != "" && idTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	info, err := resolveProxyTarget(ctx, httpClient, nameTrimmed, idTrimmed, typeTrimmed)
	if err != nil {
		return err
	}

	label := strings.TrimSpace(proxyTypeDisplay(info.Type))
	if label == "" {
		label = "Proxy"
	}

	targetName := info.Name
	if targetName == "" {
		targetName = info.ID
	}

	stateWord := strings.ToLower(actionPast)
	if info.Disabled == desiredDisabled {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q is already %s.\n", label, targetName, stateWord)
		return nil
	}

	var hostSuffix string
	if info.HostName != "" {
		hostSuffix = fmt.Sprintf(" on host %q", info.HostName)
	}

	if !assumeYes {
		message := fmt.Sprintf("%s %s %q%s", action, label, targetName, hostSuffix)
		confirmed, err := promptForConfirmation(cmd, message)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled %s %s %q.\n", strings.ToLower(action), label, targetName)
			return nil
		}
	}

	if err := apiCall(ctx, httpClient, info.ID); err != nil {
		return fmt.Errorf("%s proxy %q (%s): %w", strings.ToLower(action), targetName, info.ID, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s %q (%s).\n", actionPast, label, targetName, info.ID)
	return nil
}
