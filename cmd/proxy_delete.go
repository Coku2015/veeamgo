package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func proxyDeleteCmd() *cobra.Command {
	var (
		proxyName string
		proxyID   string
		proxyType string
		yes       bool
	)

	cmd := &cobra.Command{
		Use:   cmdProxyUse,
		Short: "Delete a backup proxy",
		Long:  "Removes a backup proxy from the Veeam Backup & Replication infrastructure.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxyDelete(cmd, proxyName, proxyID, proxyType, yes)
		},
	}

	cmd.Flags().StringVar(&proxyName, "name", "", "Proxy name to delete (optional)")
	cmd.Flags().StringVar(&proxyID, "id", "", "Proxy ID to delete (optional)")
	cmd.Flags().StringVar(&proxyType, "type", "", "Proxy type to disambiguate by name (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func runProxyDelete(cmd *cobra.Command, proxyName, proxyID, proxyType string, assumeYes bool) error {
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

	var hostSuffix string
	if info.HostName != "" {
		hostSuffix = fmt.Sprintf(" on host %q", info.HostName)
	}

	if !assumeYes {
		message := fmt.Sprintf("Permanently delete %s %q%s", label, targetName, hostSuffix)
		confirmed, err := promptForConfirmation(cmd, message)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled deleting %s %q.\n", label, targetName)
			return nil
		}
	}

	if err := httpClient.DeleteProxy(ctx, info.ID); err != nil {
		return fmt.Errorf("delete proxy %q (%s): %w", targetName, info.ID, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s %q (%s).\n", label, targetName, info.ID)
	return nil
}
