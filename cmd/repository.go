package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func repositoryCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdRepositoryUse,
		Short: "Repository inventory and maintenance",
	}
	root.AddCommand(repositoryGetCmd())
	root.AddCommand(repositoryDescribeCmd())
	root.AddCommand(repositoryRescanCmd())
	return root
}

func repositoryGetCmd() *cobra.Command {
	var (
		typeFilters []string
		nameFilter  string
		limit       int
	)

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List backup repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{
				Name:     nameFilter,
				Types:    typeFilters,
				MaxItems: limit,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, states)
			}

			rows := make([]repositoryStateRow, 0, len(states))
			for _, st := range states {
				rows = append(rows, repositoryStateRow{
					Name:       st.Name,
					Type:       repositoryTypeDisplay(st.Type),
					HostName:   st.HostName,
					Path:       st.Path,
					CapacityGB: st.CapacityGB,
					FreeGB:     st.FreeGB,
					UsedGB:     st.UsedSpaceGB,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringSliceVar(&typeFilters, "type", nil, "Filter by repository type (e.g. WinLocal, LinuxLocal)")
	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by repository name (supports * wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of repositories to return (default: all)")

	return cmd
}

func repositoryDescribeCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Show detailed repository configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return requireFlag("--name", "provide the repository name")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			state, err := httpClient.RepositoryStateByName(ctx, name)
			if err != nil {
				return err
			}

			config, err := httpClient.Repository(ctx, state.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, config)
			}

			view := repositoryDetail{
				Name:        state.Name,
				Type:        state.Type,
				Description: state.Description,
				HostName:    state.HostName,
				Path:        state.Path,
				CapacityGB:  state.CapacityGB,
				FreeGB:      state.FreeGB,
				UsedGB:      state.UsedSpaceGB,
				Online:      yesNo(state.IsOnline),
				OutOfDate:   yesNo(state.IsOutOfDate),
				ID:          state.ID,
				Config:      formatYAMLBlock(config),
			}

			return output.Print(format, view)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Repository name to describe")

	return cmd
}

func repositoryRescanCmd() *cobra.Command {
	var (
		ids  []string
		all  bool
		wait bool
	)

	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: "Rescan one or more repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && len(ids) > 0 {
				return fmt.Errorf("use either --all or --id, not both")
			}
			if !all && len(ids) == 0 {
				return fmt.Errorf("provide at least one --id or use --all")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			targetIDs := make([]string, 0)
			if all {
				states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{})
				if err != nil {
					return err
				}
				if len(states) == 0 {
					return fmt.Errorf("no repositories found")
				}
				targetIDs = make([]string, 0, len(states))
				for _, st := range states {
					targetIDs = append(targetIDs, st.ID)
				}
			} else {
				seen := make(map[string]struct{})
				for _, candidate := range ids {
					clean := strings.TrimSpace(candidate)
					if clean == "" {
						continue
					}
					if _, ok := seen[clean]; ok {
						continue
					}
					seen[clean] = struct{}{}
					targetIDs = append(targetIDs, clean)
				}
			}

			if len(targetIDs) == 0 {
				return fmt.Errorf("no repositories selected for rescan")
			}

			var sess *client.Session
			if sess, err = httpClient.RescanRepositories(ctx, targetIDs); err != nil {
				return err
			}

			if wait {
				sess, err = httpClient.WaitForSession(ctx, sess.ID, 5*time.Second)
				if err != nil {
					return err
				}
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, sess)
			}

			return output.Print(format, summarizeSession(sess))
		},
	}

	cmd.Flags().StringSliceVar(&ids, "id", nil, "Repository ID to rescan (can be repeated)")
	cmd.Flags().BoolVar(&all, "all", false, "Rescan every repository")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the rescan session to finish")

	return cmd
}

type repositoryStateRow struct {
	Name       string  `json:"Name"`
	Type       string  `json:"Type"`
	HostName   string  `json:"Host"`
	Path       string  `json:"Path"`
	CapacityGB float64 `json:"Capacity (GB)"`
	FreeGB     float64 `json:"Free (GB)"`
	UsedGB     float64 `json:"Used (GB)"`
}

type repositoryDetail struct {
	Name        string  `json:"Name"`
	Type        string  `json:"Type"`
	Description string  `json:"Description"`
	HostName    string  `json:"Host"`
	Path        string  `json:"Path"`
	CapacityGB  float64 `json:"Capacity(GB)"`
	FreeGB      float64 `json:"Free(GB)"`
	UsedGB      float64 `json:"Used(GB)"`
	Online      string  `json:"Online"`
	OutOfDate   string  `json:"Out of Date"`
	ID          string  `json:"Id"`
	Config      string  `json:"Config"`
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func repositoryTypeDisplay(raw string) string {
	if raw == "" {
		return ""
	}

	custom := map[string]string{
		"LinuxHardened":   "Linux Hardened Repository",
		"LinuxLocal":      "Linux Local Repository",
		"WinLocal":        "Windows Local Repository",
		"SOBR":            "Scale-out Backup Repository",
		"SOBRArchive":     "Scale-out Archive Extent",
		"S3Compatible":    "S3 Compatible Repository",
		"AzureBlob":       "Azure Blob Repository",
		"AzureArchive":    "Azure Archive Repository",
		"AzureDataBox":    "Azure Data Box Repository",
		"CloudConnect":    "Cloud Connect Repository",
		"CloudConnectS3":  "Cloud Connect S3 Repository",
		"CloudConnectVCC": "Cloud Connect VCC Repository",
	}
	if v, ok := custom[raw]; ok {
		return v
	}

	runes := []rune(raw)
	var builder strings.Builder
	for i, r := range runes {
		if i > 0 && shouldInsertSpace(runes[i-1], r) {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
	}

	label := strings.TrimSpace(builder.String())
	if !strings.HasSuffix(label, "Repository") {
		label = strings.TrimSpace(label + " Repository")
	}
	return label
}

func shouldInsertSpace(prev, current rune) bool {
	if unicode.IsLower(prev) && unicode.IsUpper(current) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsDigit(current) {
		return true
	}
	if unicode.IsDigit(prev) && unicode.IsLetter(current) {
		return true
	}
	return false
}
