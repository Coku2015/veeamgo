package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func serverCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdServerUse,
		Short: "Server level operations",
	}
	root.AddCommand(serverGetCmd())
	root.AddCommand(serverRescanCmd())
	root.AddCommand(serverVolumeCmd())
	root.AddCommand(serverOptionalComponentsCmd())
	return root
}

func serverGetCmd() *cobra.Command {
	get := &cobra.Command{
		Use:   cmdGetUse,
		Short: "Retrieve server information",
	}
	get.AddCommand(serverInfoCmd())
	get.AddCommand(serverTimeCmd())
	return get
}

func serverInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdInfoUse,
		Short: "Show server metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}
			info, err := httpClient.ServerInfo(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, info)
			}

			view := serverInfoView{
				Name:            info.Name,
				Platform:        info.Platform,
				VBRUUID:         info.VBRID,
				BuildVersion:    info.BuildVersion,
				Database:        info.DatabaseVendor,
				DatabaseVersion: firstNonEmpty(info.SQLServerVersion, info.DatabaseSchema, info.DatabaseContent),
				DatabaseEdition: firstNonEmpty(info.SQLServerEdition, info.DatabaseEdition),
				Patches:         formatPatches(info.Patches),
			}

			return output.Print(format, view)
		},
	}
	return cmd
}

func serverTimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdTimeUse,
		Short: "Show server clock information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}
			serverTime, err := httpClient.ServerTime(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, serverTime)
			}

			timezone := firstNonEmpty(
				serverTime.TimeZone,
				serverTime.TimeZoneDisplayName,
				serverTime.TimeZoneID,
			)
			if timezone == "" {
				timezone = serverTime.Time.Location().String()
			}

			view := serverTimeView{
				ServerTime: formatTimestampValue(serverTime.Time),
				TimeZone:   timezone,
			}

			return output.Print(format, view)
		},
	}
	return cmd
}


func serverVolumeCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "volume",
		Short: "Managed server volume settings",
	}
	root.AddCommand(serverVolumeListCmd())
	return root
}

func serverVolumeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <server-id>", cmdListUse),
		Short: "List Hyper-V volumes and CBT settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			result, err := httpClient.ManagedServerVolumes(ctx, args[0])
			if err != nil {
				if volumeManagementUnsupported(err) {
					message := "Volume management is not supported for this server type."
					if format == "json" {
						return output.Print(format, map[string]any{"message": message})
					}
					fmt.Fprintln(cmd.OutOrStdout(), message)
					return nil
				}
				return err
			}

			summary := serverVolumeSummaryRow{
				ChangedBlockTracking: yesNo(result.ChangedBlockTracking),
				FailoverToVSS:        yesNo(result.FailoverToVSSProvider),
			}
			rows := make([]serverVolumeRow, 0, len(result.Volumes))
			for _, volume := range result.Volumes {
				rows = append(rows, serverVolumeRow{
					Name:         inventoryName(volume.InventoryObject),
					Platform:     inventoryPlatform(volume.InventoryObject),
					VSSProvider:  volume.VSSProvider,
					MaxSnapshots: volume.MaxSnapshots,
				})
			}

			if format == "json" {
				payload := map[string]any{
					"summary": summary,
					"volumes": rows,
				}
				return output.Print(format, payload)
			}

			if err := output.Print(format, []serverVolumeSummaryRow{summary}); err != nil {
				return err
			}
			if len(rows) == 0 {
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return output.Print(format, rows)
		},
	}
	return cmd
}

func serverOptionalComponentsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "optional-components",
		Short: "Optional component defaults",
	}
	root.AddCommand(serverOptionalComponentsDefaultsCmd())
	return root
}

func serverOptionalComponentsDefaultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defaults",
		Short: "Show default optional components for managed servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			result, err := httpClient.ManagedServerOptionalComponentDefaults(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]optionalComponentRow, 0, len(result.Components))
			for _, component := range result.Components {
				rows = append(rows, optionalComponentRow{
					DisplayName: component.DisplayName,
					Component:   component.Component,
				})
			}

			return output.Print(format, rows)
		},
	}
	return cmd
}

type serverVolumeSummaryRow struct {
	ChangedBlockTracking string `json:"changedBlockTracking"`
	FailoverToVSS        string `json:"failoverToVssProvider"`
}

type serverVolumeRow struct {
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	VSSProvider  string `json:"vssProvider"`
	MaxSnapshots int    `json:"maxSnapshots"`
}

type optionalComponentRow struct {
	DisplayName string `json:"displayName"`
	Component   string `json:"component"`
}

func volumeManagementUnsupported(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Volume management is not supported")
}

func serverRescanCmd() *cobra.Command {
	var (
		id   string
		all  bool
		wait bool
	)

	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: "Rescan managed servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && id != "" {
				return fmt.Errorf("use either --all or --id, not both")
			}
			if !all && id == "" {
				return fmt.Errorf("specify --id <server-id> or --all")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			var (
				sessionResult *client.Session
			)
			if all {
				sessionResult, err = httpClient.RescanAllManagedServers(ctx)
			} else {
				sessionResult, err = httpClient.RescanManagedServer(ctx, id)
			}
			if err != nil {
				return err
			}

			if wait {
				sessionResult, err = httpClient.WaitForSession(ctx, sessionResult.ID, 5*time.Second)
				if err != nil {
					return err
				}
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, sessionResult)
			}

			return output.Print(format, summarizeSession(sessionResult))
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Managed server ID to rescan")
	cmd.Flags().BoolVar(&all, "all", false, "Rescan all managed servers")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the rescan session to finish")

	return cmd
}

type serverInfoView struct {
	Name            string `json:"Name"`
	Platform        string `json:"Platform"`
	VBRUUID         string `json:"VBR uuid"`
	BuildVersion    string `json:"Build Version"`
	Database        string `json:"Database"`
	DatabaseVersion string `json:"Database Version"`
	DatabaseEdition string `json:"Database Edition"`
	Patches         string `json:"Patches"`
}

type serverTimeView struct {
	ServerTime string `json:"Server Time"`
	TimeZone   string `json:"Timezone"`
}

func formatPatches(patches []string) string {
	if len(patches) == 0 {
		return ""
	}
	return strings.Join(patches, ", ")
}

func managedServerTypeDisplay(raw string) string {
	if raw == "" {
		return ""
	}

	switch raw {
	case "ViHost", "ViServer":
		return "VMware"
	case "WindowsHost", "WindowsServer":
		return "Microsoft Windows"
	case "LinuxHost", "LinuxServer":
		return "Linux"
	case "HvHost", "HvServer":
		return "Hyper-V"
	case "HvCluster":
		return "Hyper-V Cluster"
	case "NasServer":
		return "NAS Server"
	case "CloudAppliance":
		return "Cloud Appliance"
	case "TapeServer":
		return "Tape Server"
	case "ProxyAppliance":
		return "Proxy Appliance"
	case "AwsAccessNode":
		return "AWS Access Node"
	case "AzureAccessNode":
		return "Azure Access Node"
	}

	runes := []rune(raw)
	var builder strings.Builder
	for i, r := range runes {
		if i > 0 && managedServerShouldInsertSpace(runes[i-1], r) {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func managedServerShouldInsertSpace(prev, current rune) bool {
	if unicode.IsLower(prev) && unicode.IsUpper(current) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsDigit(current) {
		return true
	}
	if unicode.IsDigit(prev) && unicode.IsLetter(current) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsUpper(current) && !unicode.IsUpper(prev) {
		return true
	}
	return false
}
