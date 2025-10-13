package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func serviceCmd() *cobra.Command {
	var (
		nameFilter  string
		orderColumn string
		descending  bool
		limit       int
	)

	cmd := &cobra.Command{
		Use:     "service",
		Aliases: []string{"services"},
		Short:   "Associated service inventory",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.ServicesFilter{
				Name:        nameFilter,
				OrderColumn: orderColumn,
				MaxItems:    limit,
			}
			if orderColumn != "" {
				asc := !descending
				filter.OrderAscending = &asc
			}

			result, err := httpClient.Services(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]serviceRow, 0, len(result.Services))
			for _, svc := range result.Services {
				rows = append(rows, serviceRow{
					Name: svc.Name,
					Port: svc.Port,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by service name (supports * wildcards)")
	cmd.Flags().StringVar(&orderColumn, "sort", "", "Sort results by column (e.g. name)")
	cmd.Flags().BoolVar(&descending, "desc", false, "Sort results in descending order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

type serviceRow struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}
