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

func wanAcceleratorGetCmd() *cobra.Command {
	var (
		nameFilter string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: helptext.WanAcceleratorListShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.WANAcceleratorFilter{
				Name:     nameFilter,
				MaxItems: limit,
			}

			accelerators, err := httpClient.WANAccelerators(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, accelerators)
			}

			rows := make([]wanAcceleratorRow, 0, len(accelerators))
			for _, accel := range accelerators {
				rows = append(rows, newWanAcceleratorRow(accel))
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by WAN accelerator name (supports * wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of WAN accelerators to return (default: all)")

	return cmd
}

func wanAcceleratorDescribeCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: helptext.WanAcceleratorDescribeShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return requireFlag("--name", "provide the WAN accelerator name")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			accel, err := httpClient.WANAcceleratorByName(ctx, name)
			if err != nil {
				return err
			}

			detail, err := httpClient.WANAccelerator(ctx, accel.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail.Raw)
			}

			view := wanAcceleratorDetail{
				Name:          detail.Name,
				Description:   serverDescription(detail.Server),
				ID:            detail.ID,
				HostID:        serverHostID(detail.Server),
				TrafficPort:   serverTrafficPort(detail.Server),
				Streams:       serverStreams(detail.Server),
				HighBandwidth: yesNo(serverHighBandwidth(detail.Server)),
				CacheFolder:   cacheFolder(detail.Cache),
				CacheSize:     formatWanCacheSize(detail.Cache),
				Config:        formatYAMLBlock(detail.Raw),
			}

			return output.Print(format, view)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "WAN accelerator name to describe")

	return cmd
}

type wanAcceleratorRow struct {
	Name          string `json:"Name"`
	Description   string `json:"Description"`
	TrafficPort   int    `json:"Traffic Port"`
	Streams       int    `json:"Streams"`
	HighBandwidth string `json:"High Bandwidth Mode"`
	CacheFolder   string `json:"Cache Folder"`
	CacheSize     string `json:"Cache Size"`
}

func newWanAcceleratorRow(accel client.WANAccelerator) wanAcceleratorRow {
	return wanAcceleratorRow{
		Name:          accel.Name,
		Description:   serverDescription(accel.Server),
		TrafficPort:   serverTrafficPort(accel.Server),
		Streams:       serverStreams(accel.Server),
		HighBandwidth: yesNo(serverHighBandwidth(accel.Server)),
		CacheFolder:   cacheFolder(accel.Cache),
		CacheSize:     formatWanCacheSize(accel.Cache),
	}
}

type wanAcceleratorDetail struct {
	Name          string `json:"Name"`
	Description   string `json:"Description"`
	ID            string `json:"Id"`
	HostID        string `json:"Host Id"`
	TrafficPort   int    `json:"Traffic Port"`
	Streams       int    `json:"Streams"`
	HighBandwidth string `json:"High Bandwidth Mode"`
	CacheFolder   string `json:"Cache Folder"`
	CacheSize     string `json:"Cache Size"`
	Config        string `json:"Config"`
}

func formatWanCacheSize(cache *client.WANAcceleratorCache) string {
	if cache == nil || cache.CacheSize == 0 {
		return ""
	}
	unit := strings.TrimSpace(cache.CacheSizeUnit)
	if unit == "" {
		return fmt.Sprintf("%d", cache.CacheSize)
	}
	return fmt.Sprintf("%d %s", cache.CacheSize, unit)
}

func serverDescription(server *client.WANAcceleratorServer) string {
	if server == nil {
		return ""
	}
	return server.Description
}

func serverHostID(server *client.WANAcceleratorServer) string {
	if server == nil {
		return ""
	}
	return server.HostID
}

func serverTrafficPort(server *client.WANAcceleratorServer) int {
	if server == nil {
		return 0
	}
	return server.TrafficPort
}

func serverStreams(server *client.WANAcceleratorServer) int {
	if server == nil {
		return 0
	}
	return server.StreamsCount
}

func serverHighBandwidth(server *client.WANAcceleratorServer) bool {
	if server == nil {
		return false
	}
	return server.HighBandwidthModeEnabled
}

func cacheFolder(cache *client.WANAcceleratorCache) string {
	if cache == nil {
		return ""
	}
	return cache.CacheFolder
}
