package cmd

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

func serverGetCmd() *cobra.Command {
	get := &cobra.Command{
		Use:   cmdGetUse,
		Short: helptext.ServerGetRootShort,
	}
	get.AddCommand(serverInfoCmd())
	get.AddCommand(serverTimeCmd())
	return get
}

func serverInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdInfoUse,
		Short: helptext.ServerInfoShort,
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
		Short: helptext.ServerTimeShort,
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
