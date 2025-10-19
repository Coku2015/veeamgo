package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func licenseSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "summary",
		Aliases: []string{"overview"},
		Short:   "Show license summary",
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
		Short: "License usage and allocations",
		RunE:  summary.RunE,
	}

	cmd.AddCommand(summary)
	cmd.AddCommand(licenseSocketsCmd())
	cmd.AddCommand(licenseInstancesCmd())
	cmd.AddCommand(licenseCapacityCmd())

	return cmd
}

func licenseSocketsCmd() *cobra.Command {
	var (
		nameFilter     string
		hostNameFilter string
		hostIDFilter   string
		socketCount    int
		coreCount      int
		typeFilter     string
		orderColumn    string
		descending     bool
		limit          int
	)

	cmd := &cobra.Command{
		Use:     "sockets",
		Aliases: []string{"socket"},
		Short:   "List socket-based workloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.LicenseSocketsFilter{
				Name:          nameFilter,
				HostName:      hostNameFilter,
				HostID:        hostIDFilter,
				SocketsNumber: socketCount,
				CoresNumber:   coreCount,
				Type:          typeFilter,
				OrderColumn:   orderColumn,
				MaxItems:      limit,
			}
			if orderColumn != "" {
				asc := !descending
				filter.OrderAscending = &asc
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
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by workload name (supports * wildcards)")
	cmd.Flags().StringVar(&hostNameFilter, "host-name", "", "Filter by proxy host name")
	cmd.Flags().StringVar(&hostIDFilter, "host-id", "", "Filter by proxy host id")
	cmd.Flags().IntVar(&socketCount, "sockets", 0, "Filter by socket count")
	cmd.Flags().IntVar(&coreCount, "cores", 0, "Filter by core count")
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by workload type")
	cmd.Flags().StringVar(&orderColumn, "sort", "", "Sort results by column (e.g. name)")
	cmd.Flags().BoolVar(&descending, "desc", false, "Sort results in descending order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func licenseInstancesCmd() *cobra.Command {
	var (
		nameFilter     string
		hostNameFilter string
		usedFilter     float64
		typeFilter     string
		instanceFilter string
		orderColumn    string
		descending     bool
		limit          int
	)

	cmd := &cobra.Command{
		Use:     "instances",
		Aliases: []string{"instance"},
		Short:   "List instance-based workloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.InstanceLicensesFilter{
				Name:                nameFilter,
				HostName:            hostNameFilter,
				UsedInstancesNumber: usedFilter,
				Type:                typeFilter,
				InstanceID:          instanceFilter,
				OrderColumn:         orderColumn,
				MaxItems:            limit,
			}
			if orderColumn != "" {
				asc := !descending
				filter.OrderAscending = &asc
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
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by workload name (supports * wildcards)")
	cmd.Flags().StringVar(&hostNameFilter, "host-name", "", "Filter by host name")
	cmd.Flags().Float64Var(&usedFilter, "used", 0, "Filter by consumed instances")
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by workload type")
	cmd.Flags().StringVar(&instanceFilter, "instance-id", "", "Filter by instance id")
	cmd.Flags().StringVar(&orderColumn, "sort", "", "Sort results by column (e.g. usedInstancesNumber)")
	cmd.Flags().BoolVar(&descending, "desc", false, "Sort results in descending order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func licenseCapacityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "capacity",
		Aliases: []string{"capacity-workloads"},
		Short:   "List capacity license workloads",
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
			}

			return output.Print(format, rows)
		},
	}
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
