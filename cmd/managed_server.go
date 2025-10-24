package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

func managedServerGetCmd() *cobra.Command {
	var (
		typeFilters []string
		nameFilter  string
		limit       int
	)

	cmd := &cobra.Command{
		Use:   cmdManagedServerUse,
		Short: helptext.ManagedServerListShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			servers, err := httpClient.ManagedServers(ctx, client.ManagedServersFilter{
				Name:     nameFilter,
				Types:    typeFilters,
				MaxItems: limit,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, servers)
			}

			rows := make([]managedServerRow, 0, len(servers))
			for _, srv := range servers {
				rows = append(rows, managedServerRow{
					Name:        srv.Name,
					Type:        managedServerTypeDisplay(srv.Type),
					Status:      srv.Status,
					Description: srv.Description,
					ID:          srv.ID,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringSliceVar(&typeFilters, "type", nil, "Filter by managed server type (e.g. WindowsHost, LinuxHost)")
	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by name pattern (supports * wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of managed servers to return (default: all)")

	return cmd
}

func managedServerDescribeCmd() *cobra.Command {
	var serverID string

	cmd := &cobra.Command{
		Use:   cmdManagedServerUse,
		Short: helptext.ManagedServerDescribeShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverID == "" {
				return requireFlag("--id", "provide the managed server ID")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			server, err := httpClient.ManagedServer(ctx, serverID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, server)
			}

			typeLabel := managedServerTypeDisplay(server.Type)
			if typeLabel == "" {
				typeLabel = server.Type
			}

			view := managedServerDetail{
				Name:        server.Name,
				Type:        typeLabel,
				Status:      server.Status,
				Description: server.Description,
				ID:          server.ID,
			}

			return output.Print(format, view)
		},
	}
	cmd.Flags().StringVar(&serverID, "id", "", "Managed server identifier")
	return cmd
}

type managedServerRow struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Status      string `json:"Status"`
	Description string `json:"Description"`
	ID          string `json:"Id"`
}

type managedServerDetail struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Status      string `json:"Status"`
	Description string `json:"Description"`
	ID          string `json:"Id"`
}
