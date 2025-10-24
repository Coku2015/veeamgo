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

type publishedDiskGetOptions struct {
	id    string
	all   bool
	limit int
}

func publishedDiskGetCmd() *cobra.Command {
	opts := publishedDiskGetOptions{}

	cmd := &cobra.Command{
		Use:   "publisheddisk",
		Short: helptext.PublishedDiskShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublishedDiskGet(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.id, "id", "", "Published disk mount ID to describe")
	cmd.Flags().BoolVar(&opts.all, "all", false, "List all published disks")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of mounts to return when using --all")

	return cmd
}

func runPublishedDiskGet(cmd *cobra.Command, opts *publishedDiskGetOptions) error {
	idTrim := strings.TrimSpace(opts.id)
	if opts.all {
		if idTrim != "" {
			return fmt.Errorf("do not combine --all with --id")
		}
	} else {
		if idTrim == "" {
			return fmt.Errorf("provide --id or use --all")
		}
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	format := outputFormat()

	if opts.all {
		mounts, err := httpClient.DataIntegrationMounts(ctx, opts.limit)
		if err != nil {
			return fmt.Errorf("list published disks: %w", err)
		}

		if format == "json" {
			return output.Print(format, mounts)
		}

		rows := make([]publishedDiskRow, 0, len(mounts))
		for _, m := range mounts {
			rows = append(rows, publishedDiskRowFromSummary(m))
		}
		return output.Print(format, rows)
	}

	detail, err := httpClient.DataIntegrationMount(ctx, idTrim)
	if err != nil {
		return fmt.Errorf("describe published disk %s: %w", idTrim, err)
	}

	if format == "json" {
		return output.Print(format, detail)
	}

	return output.Print(format, publishedDiskDetailViewFromDetail(detail))
}

type publishedDiskRow struct {
	ID           string `json:"Id"`
	RestorePoint string `json:"Restore Point"`
	Mode         string `json:"Mode"`
	State        string `json:"State"`
	Target       string `json:"Target"`
	IPs          string `json:"IPs"`
	Port         int    `json:"Port"`
	Error        string `json:"Error"`
}

type publishedDiskDetailView struct {
	ID             string `json:"Id"`
	RestorePoint   string `json:"Restore Point"`
	Backup         string `json:"Backup"`
	Mode           string `json:"Mode"`
	State          string `json:"State"`
	Target         string `json:"Target"`
	IPs            string `json:"IPs"`
	Port           int    `json:"Port"`
	RestorePointID string `json:"Restore Point Id"`
	SessionID      string `json:"Session Id"`
	Error          string `json:"Error"`
}

type publishedDiskDataIntegrationDetailView struct {
	ID             string `json:"Id"`
	RestorePoint   string `json:"Restore Point"`
	Backup         string `json:"Backup"`
	Mode           string `json:"Mode"`
	State          string `json:"State"`
	Initiator      string `json:"Initiator"`
	IPs            string `json:"IPs"`
	Port           int    `json:"Port"`
	RestorePointID string `json:"Restore Point Id"`
	SessionID      string `json:"Session Id"`
	Error          string `json:"Error"`
}

func publishedDiskRowFromSummary(m client.InstantRecoveryMountSummary) publishedDiskRow {
	mode := publishedDiskMode(m)
	ips := ""
	port := 0
	if di := m.DataIntegration; di != nil {
		ips = strings.Join(di.ServerIPs, ", ")
		port = di.ServerPort
	}
	return publishedDiskRow{
		ID:           m.ID,
		RestorePoint: m.Object,
		Mode:         mode,
		State:        m.State,
		Target:       m.Host,
		IPs:          ips,
		Port:         port,
		Error:        m.Error,
	}
}

func publishedDiskMode(m client.InstantRecoveryMountSummary) string {
	if di := m.DataIntegration; di != nil {
		return di.Mode
	}
	return m.Mode
}

func publishedDiskDetailViewFromDetail(detail *client.InstantRecoveryMountDetail) any {
	ips := ""
	port := 0
	if di := detail.Summary.DataIntegration; di != nil {
		ips = strings.Join(di.ServerIPs, ", ")
		port = di.ServerPort
	}
	if detail.Platform == client.InstantRecoveryPlatformDataIntegration {
		return publishedDiskDataIntegrationDetailView{
			ID:             detail.Summary.ID,
			RestorePoint:   detail.Summary.Object,
			Backup:         detail.Summary.Backup,
			Mode:           publishedDiskMode(detail.Summary),
			State:          detail.Summary.State,
			Initiator:      detail.Summary.Host,
			IPs:            ips,
			Port:           port,
			RestorePointID: detail.Summary.RestorePointID,
			SessionID:      detail.Summary.SessionID,
			Error:          detail.Summary.Error,
		}
	}
	return publishedDiskDetailView{
		ID:             detail.Summary.ID,
		RestorePoint:   detail.Summary.Object,
		Backup:         detail.Summary.Backup,
		Mode:           publishedDiskMode(detail.Summary),
		State:          detail.Summary.State,
		Target:         detail.Summary.Host,
		IPs:            ips,
		Port:           port,
		RestorePointID: detail.Summary.RestorePointID,
		SessionID:      detail.Summary.SessionID,
		Error:          detail.Summary.Error,
	}
}
