package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func inventoryCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdInventoryUse,
		Short: "Explore inventory objects using stable endpoints",
	}
	root.AddCommand(inventoryServersCmd())
	root.AddCommand(inventoryObjectsCmd())
	root.AddCommand(inventoryPhysicalCmd())
	root.AddCommand(inventoryPhysicalItemsCmd())
	return root
}

type inventoryOptions struct {
	limit      int
	skip       int
	hierarchy  string
	filterJSON string
	filterFile string
	sort       string
	desc       bool
}

func inventoryServersCmd() *cobra.Command {
	opts := inventoryOptions{
		limit: 200,
	}

	cmd := &cobra.Command{
		Use:   "servers",
		Short: "List managed servers and hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req, err := buildInventoryRequest(opts)
			if err != nil {
				return err
			}

			result, err := httpClient.InventoryServers(ctx, req)
			if err != nil {
				return err
			}

			return renderInventoryResult(result)
		},
	}

	addInventoryFlags(cmd, &opts)
	return cmd
}

func inventoryObjectsCmd() *cobra.Command {
	opts := inventoryOptions{
		limit: 200,
	}

	cmd := &cobra.Command{
		Use:   "objects <hostname>",
		Short: "List inventory objects for a specific host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req, err := buildInventoryRequest(opts)
			if err != nil {
				return err
			}

			result, err := httpClient.InventoryObjects(ctx, args[0], req)
			if err != nil {
				return err
			}

			return renderInventoryResult(result)
		},
	}

	addInventoryFlags(cmd, &opts)
	return cmd
}

func inventoryPhysicalCmd() *cobra.Command {
	opts := inventoryOptions{
		limit: 200,
	}

	cmd := &cobra.Command{
		Use:   "physical",
		Short: "List protection groups and discovered entities",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req, err := buildInventoryRequest(opts)
			if err != nil {
				return err
			}

			result, err := httpClient.InventoryProtectionGroups(ctx, req)
			if err != nil {
				return err
			}

			return renderInventoryResult(result)
		},
	}

	addInventoryFlags(cmd, &opts)
	return cmd
}

func inventoryPhysicalItemsCmd() *cobra.Command {
	opts := inventoryOptions{
		limit: 200,
	}

	cmd := &cobra.Command{
		Use:   "physical-items <protection-group-id>",
		Short: "List machines within a protection group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req, err := buildInventoryRequest(opts)
			if err != nil {
				return err
			}

			result, err := httpClient.InventoryProtectionGroupItems(ctx, args[0], req)
			if err != nil {
				return err
			}

			return renderInventoryResult(result)
		},
	}

	addInventoryFlags(cmd, &opts)
	return cmd
}

func addInventoryFlags(cmd *cobra.Command, opts *inventoryOptions) {
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum records to return (default 200)")
	cmd.Flags().IntVar(&opts.skip, "skip", 0, "Number of records to skip")
	cmd.Flags().StringVar(&opts.hierarchy, "hierarchy", "", "Hierarchy type (e.g. HostsAndClusters, VmsAndTemplates)")
	cmd.Flags().StringVar(&opts.filterJSON, "filter", "", "Raw JSON filter payload")
	cmd.Flags().StringVar(&opts.filterFile, "filter-file", "", "Path to JSON file with filter payload")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by property (e.g. name, creationTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order (default ascending)")
}

func buildInventoryRequest(opts inventoryOptions) (client.InventoryRequest, error) {
	if opts.filterJSON != "" && opts.filterFile != "" {
		return client.InventoryRequest{}, fmt.Errorf("use either --filter or --filter-file, not both")
	}

	req := client.InventoryRequest{}
	if opts.limit > 0 || opts.skip > 0 {
		req.Pagination = &client.InventoryPagination{
			Skip:  opts.skip,
			Limit: opts.limit,
		}
	}
	if opts.hierarchy != "" {
		req.HierarchyType = opts.hierarchy
	}

	if opts.filterJSON != "" {
		filter, err := parseJSONMap(opts.filterJSON)
		if err != nil {
			return client.InventoryRequest{}, err
		}
		req.Filter = filter
	}
	if opts.filterFile != "" {
		filter, err := parseJSONFile(opts.filterFile)
		if err != nil {
			return client.InventoryRequest{}, err
		}
		req.Filter = filter
	}
	if opts.sort != "" {
		direction := "ascending"
		if opts.desc {
			direction = "descending"
		}
		req.Sorting = map[string]any{
			"property":  opts.sort,
			"direction": direction,
		}
	}

	return req, nil
}

func renderInventoryResult(result *client.InventoryResult) error {
	format := outputFormat()
	if format == "json" {
		return output.Print(format, result)
	}

	rows := make([]inventoryRow, 0, len(result.Data))
	for _, item := range result.Data {
		rows = append(rows, newInventoryRow(item))
	}
	if len(rows) == 0 {
		return output.Print(format, rows)
	}
	return output.Print(format, rows)
}

type inventoryRow struct {
	Name     string `json:"Name"`
	Kind     string `json:"Kind"`
	Platform string `json:"Platform"`
	Host     string `json:"Host"`
	ID       string `json:"ID"`
	Parent   string `json:"Parent"`
	GroupID  string `json:"Group ID"`
	Path     string `json:"Path"`
	Size     string `json:"Size"`
}

func newInventoryRow(item map[string]any) inventoryRow {
	row := inventoryRow{
		Name:     stringFromMap(item, "name"),
		Kind:     stringFromMap(item, "type"),
		Platform: stringFromMap(item, "platform"),
		Host:     stringFromMap(item, "hostName"),
		ID:       firstNonEmpty(stringFromMap(item, "objectId"), stringFromMap(item, "id"), stringFromMap(item, "resourceId")),
		Parent:   stringFromMap(item, "parentObjectId"),
		GroupID:  stringFromMap(item, "protectionGroupId"),
		Path:     stringFromMap(item, "path"),
		Size:     stringFromMap(item, "size"),
	}
	return row
}

func stringFromMap(item map[string]any, key string) string {
	if item == nil {
		return ""
	}
	if v, ok := item[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
