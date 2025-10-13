package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func exclusionVMCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "exclusionvm",
		Short: "Global VM exclusions",
	}
	root.AddCommand(newExclusionVMGetCmd(cmdGetUse, []string{cmdListUse}, false))
	root.AddCommand(newExclusionVMDescribeCmd(cmdDescribeUse, []string{"show"}, false))
	return root
}

func exclusionLegacyCmd() *cobra.Command {
	root := &cobra.Command{
		Use:    "exclusion",
		Short:  "Global exclusion policies (deprecated)",
		Hidden: true,
	}
	vm := &cobra.Command{
		Use:    "vm",
		Short:  "Global VM exclusions (deprecated)",
		Hidden: true,
	}
	vm.AddCommand(newExclusionVMGetCmd(cmdListUse, []string{cmdGetUse}, true))
	vm.AddCommand(newExclusionVMDescribeCmd(fmt.Sprintf("%s <exclusion-id>", cmdDescribeUse), nil, true))
	root.AddCommand(vm)
	return root
}

func newExclusionVMGetCmd(use string, aliases []string, hidden bool) *cobra.Command {
	var (
		orderColumn string
		descending  bool
		limit       int
	)

	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "List globally excluded VMs",
		Hidden:  hidden,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.GlobalVMExclusionsFilter{
				OrderColumn: orderColumn,
				MaxItems:    limit,
			}
			if orderColumn != "" {
				asc := !descending
				filter.OrderAscending = &asc
			}

			result, err := httpClient.GlobalVMExclusions(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]exclusionVMRow, 0, len(result.Exclusions))
			for _, exclusion := range result.Exclusions {
				rows = append(rows, exclusionVMRow{
					ID:       exclusion.ID,
					Name:     inventoryName(exclusion.InventoryObject),
					Platform: inventoryPlatform(exclusion.InventoryObject),
					Note:     exclusion.Note,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&orderColumn, "sort", "", "Sort results by column (e.g. id)")
	cmd.Flags().BoolVar(&descending, "desc", false, "Sort results in descending order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func newExclusionVMDescribeCmd(use string, aliases []string, hidden bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Describe a VM exclusion entry",
		Hidden:  hidden,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.GlobalVMExclusionsFilter{
				MaxItems: 0,
			}

			result, err := httpClient.GlobalVMExclusions(ctx, filter)
			if err != nil {
				return err
			}

			var target *client.GlobalVMExclusion
			for _, ex := range result.Exclusions {
				if ex.ID == args[0] {
					target = &ex
					break
				}
			}
			if target == nil {
				return fmt.Errorf("global VM exclusion %q not found", args[0])
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, target.Raw)
			}

			row := exclusionVMDetailRow{
				ID:          target.ID,
				Name:        inventoryName(target.InventoryObject),
				Platform:    inventoryPlatform(target.InventoryObject),
				Note:        target.Note,
				InventoryID: inventoryObjectID(target.InventoryObject),
			}
			return output.Print(format, []exclusionVMDetailRow{row})
		},
	}
	return cmd
}

type exclusionVMRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Note     string `json:"note,omitempty"`
}

type exclusionVMDetailRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	InventoryID string `json:"inventoryId,omitempty"`
	Note        string `json:"note,omitempty"`
}

func inventoryName(obj map[string]any) string {
	if obj == nil {
		return ""
	}
	if v, ok := obj["name"].(string); ok {
		return v
	}
	return ""
}

func inventoryPlatform(obj map[string]any) string {
	if obj == nil {
		return ""
	}
	if v, ok := obj["platform"].(string); ok {
		return v
	}
	if v, ok := obj["type"].(string); ok {
		return v
	}
	return ""
}

func inventoryObjectID(obj map[string]any) string {
	if obj == nil {
		return ""
	}
	if v, ok := obj["objectId"].(string); ok {
		return v
	}
	if v, ok := obj["id"].(string); ok {
		return v
	}
	return ""
}
