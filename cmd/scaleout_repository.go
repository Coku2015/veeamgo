package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/features"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func scaleOutRepositoryGetCmd() *cobra.Command {
	var (
		nameFilter string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List scale-out backup repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			if err := requireFeatureSupport(httpClient, features.FeatureScaleOutRepositories); err != nil {
				return err
			}

			repos, err := httpClient.ScaleOutRepositories(ctx, client.ScaleOutRepositoriesFilter{
				Name:     nameFilter,
				MaxItems: limit,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, repos)
			}

			rows := make([]scaleOutRepositoryRow, 0, len(repos))
			for _, repo := range repos {
				rows = append(rows, newScaleOutRepositoryRow(repo))
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by scale-out repository name (supports * wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of scale-out repositories to return (default: all)")

	return cmd
}

func scaleOutRepositoryDescribeCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Show detailed scale-out repository configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return requireFlag("--name", "provide the scale-out repository name")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			if err := requireFeatureSupport(httpClient, features.FeatureScaleOutRepositories); err != nil {
				return err
			}

			repo, err := httpClient.ScaleOutRepositoryByName(ctx, name)
			if err != nil {
				return err
			}

			fullConfig, err := httpClient.ScaleOutRepository(ctx, repo.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, fullConfig)
			}

			view := scaleOutRepositoryDetail{
				Name:                  fullConfig.Name,
				Description:           fullConfig.Description,
				ID:                    fullConfig.ID,
				UniqueID:              fullConfig.UniqueID,
				PlacementPolicy:       formatScaleOutPlacement(fullConfig.PlacementPolicy),
				StrictPlacement:       yesNo(fullConfig.PlacementPolicy != nil && fullConfig.PlacementPolicy.EnforceStrictPlacementPolicy),
				PerVMBackup:           yesNo(fullConfig.PerformanceTier.AdvancedSettings != nil && fullConfig.PerformanceTier.AdvancedSettings.PerVMBackup),
				FullWhenExtentOffline: yesNo(fullConfig.PerformanceTier.AdvancedSettings != nil && fullConfig.PerformanceTier.AdvancedSettings.FullWhenExtentOffline),
				PerformanceExtents:    formatScaleOutExtentDetail(fullConfig.PerformanceTier.PerformanceExtents),
				CapacityTier:          formatScaleOutCapacityTier(fullConfig.CapacityTier),
				ArchiveTier:           formatScaleOutArchiveTier(fullConfig.ArchiveTier),
				Config:                formatYAMLBlock(fullConfig),
			}

			return output.Print(format, view)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Scale-out repository name to describe")

	return cmd
}

type scaleOutRepositoryRow struct {
	Name            string `json:"Name"`
	Extents         int    `json:"Performance Extents"`
	PlacementPolicy string `json:"Placement Policy"`
	StrictPlacement string `json:"Strict Placement"`
	CapacityTier    string `json:"Capacity Tier"`
	ArchiveTier     string `json:"Archive Tier"`
}

func newScaleOutRepositoryRow(repo client.ScaleOutRepository) scaleOutRepositoryRow {
	return scaleOutRepositoryRow{
		Name:            repo.Name,
		Extents:         len(repo.PerformanceTier.PerformanceExtents),
		PlacementPolicy: formatScaleOutPlacement(repo.PlacementPolicy),
		StrictPlacement: yesNo(repo.PlacementPolicy != nil && repo.PlacementPolicy.EnforceStrictPlacementPolicy),
		CapacityTier:    summarizeScaleOutCapacity(repo.CapacityTier),
		ArchiveTier:     summarizeScaleOutArchive(repo.ArchiveTier),
	}
}

type scaleOutRepositoryDetail struct {
	Name                  string `json:"Name"`
	Description           string `json:"Description"`
	ID                    string `json:"Id"`
	UniqueID              string `json:"Unique Id"`
	PlacementPolicy       string `json:"Placement Policy"`
	StrictPlacement       string `json:"Strict Placement"`
	PerVMBackup           string `json:"Per-VM Backup"`
	FullWhenExtentOffline string `json:"Full Backup When Extent Offline"`
	PerformanceExtents    string `json:"Performance Extents"`
	CapacityTier          string `json:"Capacity Tier"`
	ArchiveTier           string `json:"Archive Tier"`
	Config                string `json:"Config"`
}

func formatScaleOutPlacement(policy *client.ScaleOutPlacementPolicy) string {
	if policy == nil {
		return ""
	}
	if label, ok := placementPolicyLabels[strings.TrimSpace(policy.Type)]; ok {
		return label
	}
	return strings.TrimSpace(policy.Type)
}

func summarizeScaleOutCapacity(cap *client.ScaleOutCapacityTier) string {
	if cap == nil {
		return ""
	}
	if !cap.IsEnabled {
		return "Disabled"
	}

	parts := []string{"Enabled"}
	if len(cap.Extents) > 0 {
		parts = append(parts, fmt.Sprintf("%d extent(s)", len(cap.Extents)))
	}
	flags := make([]string, 0, 2)
	if cap.CopyPolicyEnabled {
		flags = append(flags, "Copy")
	}
	if cap.MovePolicyEnabled {
		flags = append(flags, "Move")
	}
	if len(flags) > 0 {
		parts = append(parts, strings.Join(flags, " & "))
	}
	return strings.Join(parts, " | ")
}

func summarizeScaleOutArchive(arch *client.ScaleOutArchiveTier) string {
	if arch == nil {
		return ""
	}
	if !arch.IsEnabled {
		return "Disabled"
	}

	details := []string{"Enabled"}
	if arch.ExtentID != "" {
		details = append(details, fmt.Sprintf("Extent %s", arch.ExtentID))
	}
	if arch.ArchivePeriodDays > 0 {
		details = append(details, fmt.Sprintf("After %d day(s)", arch.ArchivePeriodDays))
	}
	return strings.Join(details, " | ")
}

func formatScaleOutExtentDetail(extents []client.ScaleOutPerformanceExtent) string {
	if len(extents) == 0 {
		return ""
	}

	lines := make([]string, 0, len(extents))
	for _, extent := range extents {
		status := strings.Join(extent.Status, ", ")
		line := extent.Name
		if extent.ID != "" {
			line = fmt.Sprintf("%s (%s)", extent.Name, extent.ID)
		}
		if status != "" {
			line = fmt.Sprintf("%s [%s]", line, status)
		}
		lines = append(lines, line)
	}
	return "\n" + indentLines(strings.Join(lines, "\n"), "  ")
}

func formatScaleOutCapacityTier(cap *client.ScaleOutCapacityTier) string {
	if cap == nil {
		return ""
	}
	return formatScaleOutBlock(summarizeScaleOutCapacity(cap))
}

func formatScaleOutArchiveTier(arch *client.ScaleOutArchiveTier) string {
	if arch == nil {
		return ""
	}
	return formatScaleOutBlock(summarizeScaleOutArchive(arch))
}

func formatScaleOutBlock(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "\n" + indentLines(value, "  ")
}

var placementPolicyLabels = map[string]string{
	"DataLocality": "Data Locality",
	"Performance":  "Performance",
}
