package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

func licenseSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: helptext.LicenseSummaryShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			summary, err := httpClient.LicenseSummary(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, summary.Raw)
			}

			row := licenseSummaryRow{
				LicensedTo:       summary.LicensedTo,
				Edition:          summary.Edition,
				Type:             summary.Type,
				Status:           summary.Status,
				ExpiresAt:        formatAPITime(summary.ExpirationDate),
				SupportExpiresAt: formatAPITime(summary.SupportExpirationDate),
			}

			return output.Print(format, []licenseSummaryRow{row})
		},
	}
	return cmd
}

func licenseVerbCmd() *cobra.Command {
	summary := licenseSummaryCmd()
	cmd := &cobra.Command{
		Use:   "license",
		Short: helptext.LicenseRootShort,
		RunE:  summary.RunE,
	}

	cmd.AddCommand(summary)
	cmd.AddCommand(licenseSocketsCmd())
	cmd.AddCommand(licenseInstancesCmd())
	cmd.AddCommand(licenseCapacityCmd())

	return cmd
}

func licenseSocketsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "sockets",
		Short: helptext.LicenseSocketsShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.LicenseSocketsFilter{
				MaxItems: limit,
			}

			result, err := httpClient.LicenseSockets(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]licenseSocketRow, 0, len(result.Workloads))
			for _, workload := range result.Workloads {
				rows = append(rows, licenseSocketRow{
					Workload: workload.Name,
					Host:     workload.HostName,
					Type:     workload.Type,
					Sockets:  workload.SocketsNumber,
					Cores:    workload.CoresNumber,
					HostID:   workload.HostID,
				})
				if limit > 0 && len(rows) >= limit {
					break
				}
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func licenseInstancesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "instances",
		Short: helptext.LicenseInstancesShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.InstanceLicensesFilter{
				MaxItems: limit,
			}

			result, err := httpClient.LicenseInstances(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]licenseInstanceRow, 0, len(result.Workloads))
			for _, workload := range result.Workloads {
				name := firstNonEmpty(workload.Name, workload.DisplayName)
				rows = append(rows, licenseInstanceRow{
					Workload:  name,
					Host:      workload.HostName,
					Type:      workload.Type,
					Platform:  workload.PlatformType,
					Used:      formatFloat(workload.UsedInstancesNumber),
					Revocable: yesNo(workload.CanBeRevoked),
				})
				if limit > 0 && len(rows) >= limit {
					break
				}
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func licenseCapacityCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "capacity",
		Short: helptext.LicenseCapacityShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			result, err := httpClient.LicenseCapacity(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]licenseCapacityRow, 0, len(result.Workloads))
			for _, workload := range result.Workloads {
				rows = append(rows, licenseCapacityRow{
					Workload:   workload.Name,
					Type:       workload.Type,
					UsedTB:     formatFloat(workload.UsedCapacityTB),
					InstanceID: workload.InstanceID,
				})
				if limit > 0 && len(rows) >= limit {
					break
				}
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")
	return cmd
}

type licenseSummaryRow struct {
	LicensedTo       string `json:"Licensed To"`
	Edition          string `json:"Edition"`
	Type             string `json:"Type"`
	Status           string `json:"Status"`
	ExpiresAt        string `json:"Expires At,omitempty"`
	SupportExpiresAt string `json:"Support Expires At,omitempty"`
}

type licenseSocketRow struct {
	Workload string `json:"Workload"`
	Host     string `json:"Host"`
	Type     string `json:"Type"`
	Sockets  int    `json:"Sockets"`
	Cores    int    `json:"Cores"`
	HostID   string `json:"Host ID"`
}

type licenseInstanceRow struct {
	Workload  string `json:"Workload"`
	Host      string `json:"Host"`
	Type      string `json:"Type"`
	Platform  string `json:"Platform"`
	Used      string `json:"Used Instances"`
	Revocable string `json:"Revocable"`
}

type licenseCapacityRow struct {
	Workload   string `json:"Workload"`
	Type       string `json:"Type"`
	UsedTB     string `json:"Used (TB)"`
	InstanceID string `json:"Instance ID,omitempty"`
}

func formatAPITime(value *client.APITime) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return formatTimeForDisplay(value.Time)
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
