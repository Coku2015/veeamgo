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
