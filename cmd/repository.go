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

			var normalizedTypes []string
			if len(typeFilters) > 0 {
				normalizedTypes = make([]string, 0, len(typeFilters))
				for _, t := range typeFilters {
					normalized, err := normalizeLocalRepositoryType(t)
					if err != nil {
						return err
					}
					normalizedTypes = append(normalizedTypes, normalized)
				}
			}
			filter := client.RepositoryStatesFilter{
				Name:     nameFilter,
				MaxItems: limit,
			}
			if len(normalizedTypes) > 0 {
				filter.Types = normalizedTypes
			}

			states, err := httpClient.RepositoryStates(ctx, filter)
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
			if !isLocalRepositoryType(state.Type) {
				return fmt.Errorf("repository %q is an object storage repository; use veeamgo describe objectrepository", name)
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
		ids   []string
		names []string
		all   bool
		wait  bool
	)

	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: "Rescan one or more repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && (len(ids) > 0 || len(names) > 0) {
				return fmt.Errorf("use either --all or one of --id/--name")
			}
			if !all && len(ids) == 0 && len(names) == 0 {
				return fmt.Errorf("provide at least one --id, --name, or use --all")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			seen := make(map[string]struct{})
			targetIDs := make([]string, 0)
			if all {
				states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{Types: localRepositoryTypes()})
				if err != nil {
					return err
				}
				if len(states) == 0 {
					return fmt.Errorf("no repositories found")
				}
				for _, st := range states {
					if !isLocalRepositoryType(st.Type) {
						continue
					}
					if _, ok := seen[st.ID]; !ok {
						seen[st.ID] = struct{}{}
						targetIDs = append(targetIDs, st.ID)
					}
				}
			} else {
				for _, candidate := range ids {
					clean := strings.TrimSpace(candidate)
					if clean == "" {
						continue
					}
					if _, ok := seen[clean]; ok {
						continue
					}
					states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{ID: clean, MaxItems: 1})
					if err != nil {
						return err
					}
					if len(states) == 0 {
						return fmt.Errorf("repository with id %q not found", clean)
					}
					if !isLocalRepositoryType(states[0].Type) {
						return fmt.Errorf("repository with id %q is an object storage repository; use veeamgo rescan objectrepository", clean)
					}
					seen[clean] = struct{}{}
					targetIDs = append(targetIDs, clean)
				}

				for _, candidate := range names {
					clean := strings.TrimSpace(candidate)
					if clean == "" {
						continue
					}
					state, err := httpClient.RepositoryStateByName(ctx, clean)
					if err != nil {
						return err
					}
					if !isLocalRepositoryType(state.Type) {
						return fmt.Errorf("repository %q is an object storage repository; use veeamgo rescan objectrepository", clean)
					}
					if _, ok := seen[state.ID]; ok {
						continue
					}
					seen[state.ID] = struct{}{}
					targetIDs = append(targetIDs, state.ID)
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
	cmd.Flags().StringSliceVar(&names, "name", nil, "Repository name to rescan (can be repeated)")
	cmd.Flags().BoolVar(&all, "all", false, "Rescan every repository")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the rescan session to finish")

	return cmd
}

func repositoryDeleteCmd() *cobra.Command {
	var (
		repoName      string
		repoID        string
		deleteBackups bool
		yes           bool
	)

	cmd := &cobra.Command{
		Use:   cmdDeleteUse,
		Short: "Delete a backup repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			nameTrim := strings.TrimSpace(repoName)
			idTrim := strings.TrimSpace(repoID)
			if nameTrim == "" && idTrim == "" {
				return fmt.Errorf("provide --name or --id")
			}
			if nameTrim != "" && idTrim != "" {
				return fmt.Errorf("provide either --name or --id, not both")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			var state *client.RepositoryState
			if idTrim != "" {
				states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{ID: idTrim, MaxItems: 1})
				if err != nil {
					return err
				}
				if len(states) == 0 {
					return fmt.Errorf("repository with id %q not found", idTrim)
				}
				state = &states[0]
			} else {
				var err error
				state, err = httpClient.RepositoryStateByName(ctx, nameTrim)
				if err != nil {
					return err
				}
				idTrim = state.ID
			}
			if !isLocalRepositoryType(state.Type) {
				return fmt.Errorf("repository %q is an object storage repository; use veeamgo delete objectrepository", state.Name)
			}

			label := repositoryTypeDisplay(state.Type)
			if label == "" {
				label = "Repository"
			}

			repoDisplay := state.Name
			if repoDisplay == "" {
				repoDisplay = idTrim
			}

			var hostSuffix string
			if state.HostName != "" {
				hostSuffix = fmt.Sprintf(" on host %q", state.HostName)
			}

			if !yes {
				prompt := fmt.Sprintf("Permanently delete %s %q%s", label, repoDisplay, hostSuffix)
				if deleteBackups {
					prompt += " (including backup files)"
				}
				confirmed, err := promptForConfirmation(cmd, prompt)
				if err != nil {
					return err
				}
				if !confirmed {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled deleting %s %q.\n", label, repoDisplay)
					return nil
				}
			}

			if err := httpClient.DeleteRepository(ctx, idTrim, deleteBackups); err != nil {
				return fmt.Errorf("delete repository %q (%s): %w", repoDisplay, idTrim, err)
			}

			message := fmt.Sprintf("Deleted %s %q (%s)", label, repoDisplay, idTrim)
			if deleteBackups {
				message += " and removed backup files"
			}
			message += "."
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), message)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoName, "name", "", "Repository name to delete (optional)")
	cmd.Flags().StringVar(&repoID, "id", "", "Repository ID to delete (optional)")
	cmd.Flags().BoolVar(&deleteBackups, "delete-backups", false, "Also remove backup files stored in the repository")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")

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
