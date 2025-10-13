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

func restoreMountCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mount",
		Short: "Monitor Instant Recovery mount points",
	}
	root.AddCommand(restoreMountGetCmd())
	root.AddCommand(restoreMountDescribeCmd())
	root.AddCommand(restoreMountSessionsCmd())
	root.AddCommand(restoreMountSwitchoverCmd())
	return root
}

type restoreMountListOptions struct {
	platforms []string
	state     string
	name      string
	limit     int
}

func restoreMountGetCmd() *cobra.Command {
	opts := restoreMountListOptions{}

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List Instant Recovery mount points",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			platforms, err := parseMountPlatforms(opts.platforms)
			if err != nil {
				return err
			}

			summaries, err := httpClient.InstantRecoveryMounts(ctx, client.InstantRecoveryMountFilter{
				Platforms: platforms,
				State:     opts.state,
				Name:      opts.name,
				Limit:     opts.limit,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				raw := make([]map[string]any, 0, len(summaries))
				for _, summary := range summaries {
					if summary.Raw != nil {
						raw = append(raw, summary.Raw)
					}
				}
				return output.Print(format, raw)
			}

			rows := make([]restoreMountRow, 0, len(summaries))
			for _, summary := range summaries {
				rows = append(rows, mapMountSummary(summary))
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringSliceVar(&opts.platforms, "platform", nil, "Platform filter (vsphere|hyperv|azure|fcd|data-integration)")
	cmd.Flags().StringVar(&opts.state, "state", "", "Filter by mount state")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by object name (supports * wildcards for VM-based mounts)")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of mounts to return")

	return cmd
}

func restoreMountDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <mount-id>", cmdDescribeUse),
		Short: "Show detailed information about a mount",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			detail, err := httpClient.InstantRecoveryMountDetail(ctx, args[0])
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail.Raw)
			}

			view := buildMountDetailView(detail)

			if detail.Platform == client.InstantRecoveryPlatformAzure {
				if switchover, err := httpClient.AzureInstantRecoverySwitchoverSettings(ctx, args[0]); err == nil {
					view.AzureSwitchover = formatSwitchoverSettings(switchover)
				}
			}

			return output.Print(format, view)
		},
	}
	return cmd
}

func restoreMountSessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions <mount-id>",
		Short: "List lifecycle sessions for Azure mounts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			detail, err := httpClient.InstantRecoveryMountDetail(ctx, args[0])
			if err != nil {
				return err
			}
			if detail.Platform != client.InstantRecoveryPlatformAzure {
				return fmt.Errorf("sessions are only available for Azure Instant Recovery mounts")
			}

			sessions, err := httpClient.AzureInstantRecoveryMountSessions(ctx, args[0])
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, sessions)
			}

			rows := make([]restoreMountSessionRow, 0, len(sessions))
			for _, sess := range sessions {
				rows = append(rows, restoreMountSessionRow{
					ID:        sess.ID,
					Name:      sess.Name,
					Type:      sess.SessionType,
					State:     sess.State,
					Result:    sessionResult(sess.Result),
					Progress:  fmt.Sprintf("%d%%", sess.ProgressPercent),
					CreatedAt: formatTimestampValue(sess.CreationTime),
					EndedAt:   formatTimestamp(sess.EndTime),
					JobID:     sess.JobID,
				})
			}

			return output.Print(format, rows)
		},
	}
	return cmd
}

func restoreMountSwitchoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switchover <mount-id>",
		Short: "Show Azure switchover settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			detail, err := httpClient.InstantRecoveryMountDetail(ctx, args[0])
			if err != nil {
				return err
			}
			if detail.Platform != client.InstantRecoveryPlatformAzure {
				return fmt.Errorf("switchover settings are only available for Azure Instant Recovery mounts")
			}

			settings, err := httpClient.AzureInstantRecoverySwitchoverSettings(ctx, args[0])
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, settings)
			}

			view := restoreMountSwitchoverView{
				MountID:    args[0],
				Type:       settings.Type,
				Scheduled:  formatTimestamp(settings.ScheduleTime),
				VerifyBoot: yesNo(settings.VerifyVMBoot),
				PowerOn:    yesNo(settings.PowerOnVM),
			}

			return output.Print(format, view)
		},
	}
	return cmd
}

func parseMountPlatforms(values []string) ([]client.InstantRecoveryPlatform, error) {
	if len(values) == 0 {
		return nil, nil
	}
	mapped := make([]client.InstantRecoveryPlatform, 0, len(values))
	for _, raw := range values {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "vsphere", "vmware":
			mapped = append(mapped, client.InstantRecoveryPlatformVSphere)
		case "hyperv", "hyper-v":
			mapped = append(mapped, client.InstantRecoveryPlatformHyperV)
		case "azure":
			mapped = append(mapped, client.InstantRecoveryPlatformAzure)
		case "fcd":
			mapped = append(mapped, client.InstantRecoveryPlatformVSphereFCD)
		case "data-integration", "dataintegration", "data_integration":
			mapped = append(mapped, client.InstantRecoveryPlatformDataIntegration)
		default:
			return nil, fmt.Errorf("unknown platform %q (use vsphere|hyperv|azure|fcd|data-integration)", raw)
		}
	}
	return mapped, nil
}

type restoreMountRow struct {
	Platform    string `json:"Platform"`
	Object      string `json:"Object"`
	State       string `json:"State"`
	Target      string `json:"Target"`
	Session     string `json:"Session"`
	JobOrBackup string `json:"Job/Backup"`
	Restore     string `json:"Restore Point"`
	Notes       string `json:"Notes"`
	ID          string `json:"ID"`
}

func mapMountSummary(summary client.InstantRecoveryMountSummary) restoreMountRow {
	target := summary.Host
	if summary.Azure != nil {
		target = summary.Azure.ResourceGroup
		if target == "" {
			target = summary.Azure.Location
		}
	}
	if summary.DataIntegration != nil {
		if len(summary.DataIntegration.ServerIPs) > 0 {
			target = strings.Join(summary.DataIntegration.ServerIPs, ",")
			if summary.DataIntegration.ServerPort > 0 {
				target = fmt.Sprintf("%s:%d", target, summary.DataIntegration.ServerPort)
			}
		}
		if target == "" {
			target = summary.Host
		}
	}

	job := summary.Job
	if job == "" {
		job = summary.Backup
	}

	restore := ""
	if !summary.RestorePoint.IsZero() {
		restore = formatTimestampValue(summary.RestorePoint)
	}

	notes := buildMountNotes(summary)

	return restoreMountRow{
		Platform:    string(summary.Platform),
		Object:      summary.Object,
		State:       summary.State,
		Target:      target,
		Session:     summary.SessionID,
		JobOrBackup: job,
		Restore:     restore,
		Notes:       notes,
		ID:          summary.ID,
	}
}

func buildMountNotes(summary client.InstantRecoveryMountSummary) string {
	notes := make([]string, 0, 4)
	if summary.Mode != "" {
		notes = append(notes, summary.Mode)
	}
	if summary.Azure != nil {
		if summary.Azure.Network != "" {
			notes = append(notes, summary.Azure.Network)
		}
		if summary.Azure.AssignPublicIP {
			notes = append(notes, "Public IP")
		}
		if summary.Azure.VerifyBoot {
			notes = append(notes, "Verify Boot")
		}
	}
	if summary.DataIntegration != nil && summary.DataIntegration.Mode != "" {
		notes = append(notes, summary.DataIntegration.Mode)
	}
	if summary.FCD != nil && summary.FCD.DiskCount > 0 {
		notes = append(notes, fmt.Sprintf("%d disk(s)", summary.FCD.DiskCount))
	}
	if summary.Error != "" {
		notes = append(notes, "Error: "+summary.Error)
	}
	return strings.Join(notes, " | ")
}

type restoreMountDetailView struct {
	ID                      string `json:"ID"`
	Platform                string `json:"Platform"`
	Object                  string `json:"Object"`
	State                   string `json:"State"`
	RestorePointID          string `json:"Restore Point ID"`
	RestorePoint            string `json:"Restore Point"`
	Session                 string `json:"Session"`
	JobOrBackup             string `json:"Job/Backup"`
	Target                  string `json:"Target"`
	Mode                    string `json:"Mode"`
	AzureSubscription       string `json:"Azure Subscription"`
	AzureResourceGroup      string `json:"Azure Resource Group"`
	AzureLocation           string `json:"Azure Location"`
	AzureNetwork            string `json:"Azure Network"`
	AzurePublicIP           string `json:"Azure Public IP"`
	AzureVerifyBoot         string `json:"Azure Verify Boot"`
	AzureSwitchover         string `json:"Azure Switchover"`
	DataIntegrationMode     string `json:"Data Integration Mode"`
	DataIntegrationEndpoint string `json:"Data Integration Endpoint"`
	FCDDestination          string `json:"FCD Destination"`
	FCDDisks                string `json:"FCD Disks"`
	Error                   string `json:"Error"`
	Details                 string `json:"Details"`
}

func buildMountDetailView(detail *client.InstantRecoveryMountDetail) restoreMountDetailView {
	summary := detail.Summary

	restore := ""
	if !summary.RestorePoint.IsZero() {
		restore = formatTimestampValue(summary.RestorePoint)
	}

	target := summary.Host
	if summary.Azure != nil {
		target = summary.Azure.ResourceGroup
		if target == "" {
			target = summary.Azure.Location
		}
	}
	if summary.DataIntegration != nil {
		if len(summary.DataIntegration.ServerIPs) > 0 {
			target = strings.Join(summary.DataIntegration.ServerIPs, ",")
			if summary.DataIntegration.ServerPort > 0 {
				target = fmt.Sprintf("%s:%d", target, summary.DataIntegration.ServerPort)
			}
		}
	}

	job := summary.Job
	if job == "" {
		job = summary.Backup
	}

	view := restoreMountDetailView{
		ID:             summary.ID,
		Platform:       string(detail.Platform),
		Object:         summary.Object,
		State:          summary.State,
		RestorePointID: summary.RestorePointID,
		RestorePoint:   restore,
		Session:        summary.SessionID,
		JobOrBackup:    job,
		Target:         target,
		Mode:           summary.Mode,
		Error:          summary.Error,
		Details:        formatYAMLBlock(detail.Raw),
	}

	if summary.Azure != nil {
		view.AzureSubscription = summary.Azure.Subscription
		view.AzureResourceGroup = summary.Azure.ResourceGroup
		view.AzureLocation = summary.Azure.Location
		view.AzureNetwork = summary.Azure.Network
		view.AzurePublicIP = yesNo(summary.Azure.AssignPublicIP)
		view.AzureVerifyBoot = yesNo(summary.Azure.VerifyBoot)
	}

	if summary.DataIntegration != nil {
		view.DataIntegrationMode = summary.DataIntegration.Mode
		if len(summary.DataIntegration.ServerIPs) > 0 {
			view.DataIntegrationEndpoint = strings.Join(summary.DataIntegration.ServerIPs, ",")
			if summary.DataIntegration.ServerPort > 0 {
				view.DataIntegrationEndpoint = fmt.Sprintf("%s:%d", view.DataIntegrationEndpoint, summary.DataIntegration.ServerPort)
			}
		}
	}

	if summary.FCD != nil {
		view.FCDDestination = summary.FCD.Destination
		if len(summary.FCD.Disks) > 0 {
			view.FCDDisks = strings.Join(summary.FCD.Disks, ", ")
		} else if summary.FCD.DiskCount > 0 {
			view.FCDDisks = fmt.Sprintf("%d disk(s)", summary.FCD.DiskCount)
		}
	}

	return view
}

type restoreMountSwitchoverView struct {
	MountID    string `json:"Mount ID"`
	Type       string `json:"Type"`
	Scheduled  string `json:"Scheduled"`
	VerifyBoot string `json:"Verify VM Boot"`
	PowerOn    string `json:"Power On VM"`
}

type restoreMountSessionRow struct {
	ID        string `json:"ID"`
	Name      string `json:"Name"`
	Type      string `json:"Type"`
	State     string `json:"State"`
	Result    string `json:"Result"`
	Progress  string `json:"Progress"`
	CreatedAt string `json:"Created"`
	EndedAt   string `json:"Ended"`
	JobID     string `json:"Job ID"`
}

func formatSwitchoverSettings(settings *client.AzureInstantRecoverySwitchoverSettings) string {
	if settings == nil {
		return ""
	}
	parts := []string{settings.Type}
	if settings.ScheduleTime != nil && !settings.ScheduleTime.IsZero() {
		parts = append(parts, formatTimestamp(settings.ScheduleTime))
	}
	if settings.VerifyVMBoot {
		parts = append(parts, "Verify Boot")
	}
	if settings.PowerOnVM {
		parts = append(parts, "Power On")
	}
	return strings.Join(parts, " | ")
}
