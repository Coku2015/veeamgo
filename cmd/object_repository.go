package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func objectRepositoryGetCmd() *cobra.Command {
	var (
		typeFilters []string
		nameFilter  string
		limit       int
	)

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List object storage repositories",
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
					normalized, err := normalizeObjectRepositoryType(t)
					if err != nil {
						return err
					}
					normalizedTypes = append(normalizedTypes, normalized)
				}
			} else {
				normalizedTypes = objectRepositoryTypes()
			}

			states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{
				Name:     nameFilter,
				Types:    normalizedTypes,
				MaxItems: limit,
			})
			if err != nil {
				return err
			}

			filtered := make([]client.RepositoryState, 0, len(states))
			for _, st := range states {
				if isObjectRepositoryType(st.Type) {
					filtered = append(filtered, st)
				}
			}
			states = filtered

			format := outputFormat()
			if format == "json" {
				return output.Print(format, states)
			}

			rows := make([]objectRepositoryRow, 0, len(states))
			for _, st := range states {
				rows = append(rows, objectRepositoryRow{
					Name:        st.Name,
					Type:        repositoryTypeDisplay(st.Type),
					Description: st.Description,
					Online:      yesNo(st.IsOnline),
					ID:          st.ID,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringSliceVar(&typeFilters, "type", nil, "Filter by repository type (e.g. AmazonS3, AzureBlob, S3Compatible)")
	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by repository name (supports * wildcards)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of repositories to return (default: all)")

	return cmd
}

func objectRepositoryDescribeCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Describe an object storage repository",
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
			if !isObjectRepositoryType(state.Type) {
				return fmt.Errorf("repository %q is not an object storage repository; use veeamgo describe repository", name)
			}

			detail, err := httpClient.Repository(ctx, state.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail)
			}

			view := objectRepositoryDetail{
				Name:        state.Name,
				Type:        repositoryTypeDisplay(state.Type),
				Description: state.Description,
				Online:      yesNo(state.IsOnline),
				ID:          state.ID,
				Config:      formatYAMLBlock(detail),
			}
			location := deriveObjectTarget(state.Type, detail)
			if location != "" {
				view.Target = location
			}

			return output.Print(format, view)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Object storage repository name to describe")
	return cmd
}

func objectRepositoryRescanCmd() *cobra.Command {
	var (
		ids   []string
		names []string
		all   bool
		wait  bool
	)

	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: "Rescan object storage repositories",
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
				states, err := httpClient.RepositoryStates(ctx, client.RepositoryStatesFilter{Types: objectRepositoryTypes()})
				if err != nil {
					return err
				}
				if len(states) == 0 {
					return fmt.Errorf("no object storage repositories found")
				}
				for _, st := range states {
					if !isObjectRepositoryType(st.Type) {
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
					if !isObjectRepositoryType(states[0].Type) {
						return fmt.Errorf("repository with id %q is not an object storage repository; use veeamgo rescan repository", clean)
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
					if !isObjectRepositoryType(state.Type) {
						return fmt.Errorf("repository %q is not an object storage repository; use veeamgo rescan repository", clean)
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

			sess, err := httpClient.RescanRepositories(ctx, targetIDs)
			if err != nil {
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
	cmd.Flags().BoolVar(&all, "all", false, "Rescan every object storage repository")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the rescan session to finish")

	return cmd
}

func objectRepositoryDeleteCmd() *cobra.Command {
	var (
		repoName      string
		repoID        string
		deleteBackups bool
		yes           bool
	)

	cmd := &cobra.Command{
		Use:   cmdDeleteUse,
		Short: "Delete an object storage repository",
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

			if !isObjectRepositoryType(state.Type) {
				return fmt.Errorf("repository %q is not an object storage repository; use veeamgo delete repository", state.Name)
			}

			label := repositoryTypeDisplay(state.Type)
			if label == "" {
				label = "Object repository"
			}

			repoDisplay := state.Name
			if repoDisplay == "" {
				repoDisplay = idTrim
			}

			if !yes {
				prompt := fmt.Sprintf("Permanently delete %s %q", label, repoDisplay)
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

type objectRepositoryAddOptions struct {
	specPath    string
	setPairs    []string
	repoType    string
	name        string
	description string
	disable     bool
	wait        bool
	yes         bool
}

func objectRepositoryAddCmd() *cobra.Command {
	opts := objectRepositoryAddOptions{}

	cmd := &cobra.Command{
		Use:   cmdObjectRepositoryUse,
		Short: "Add an object storage repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			disableSet := cmd.Flags().Changed("disable")
			return runObjectRepositoryAdd(cmd, &opts, disableSet)
		},
	}

	cmd.Flags().StringVar(&opts.repoType, "type", "", "Object repository type (AzureBlob, AmazonS3, S3Compatible, ...)")
	cmd.Flags().StringVar(&opts.specPath, "spec", "", "Path to JSON spec describing the repository (use '-' for stdin)")
	cmd.Flags().StringSliceVar(&opts.setPairs, "set", nil, "Override spec value (repeatable, dot notation, e.g. bucket.bucketName=my-bucket)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Override repository name")
	cmd.Flags().StringVar(&opts.description, "description", "", "Override repository description")
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Create the repository in a disabled state")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required to execute)")

	return cmd
}

func runObjectRepositoryAdd(cmd *cobra.Command, opts *objectRepositoryAddOptions, disableSet bool) error {
	if strings.TrimSpace(opts.specPath) == "" {
		return requireFlag("--spec", "provide the repository spec file path")
	}

	payload, err := loadSpecFile(opts.specPath)
	if err != nil {
		return err
	}

	repoType := strings.TrimSpace(opts.repoType)
	if repoType == "" {
		if v, ok := payload["type"].(string); ok {
			repoType = v
		}
	}
	if repoType == "" {
		return requireFlag("--type", "provide the repository type")
	}

	canonicalType, err := normalizeObjectRepositoryType(repoType)
	if err != nil {
		return err
	}
	payload["type"] = canonicalType

	if strings.TrimSpace(opts.name) != "" {
		payload["name"] = opts.name
	}
	if strings.TrimSpace(opts.description) != "" {
		payload["description"] = opts.description
	}
	if _, ok := payload["name"].(string); !ok || strings.TrimSpace(payload["name"].(string)) == "" {
		return requireFlag("--name", "provide the repository name (or include it in the spec)")
	}
	if _, ok := payload["description"].(string); !ok {
		payload["description"] = ""
	}

	if disableSet {
		payload["isDisabled"] = opts.disable
	} else if _, ok := payload["isDisabled"]; !ok {
		payload["isDisabled"] = false
	}

	overrides, err := parseOverridePairs(opts.setPairs)
	if err != nil {
		return err
	}
	if len(overrides) > 0 {
		if err := applyOverridesToMap(payload, overrides); err != nil {
			return err
		}
	}

	info := repositoryCreationResult{
		objectTarget: deriveObjectTarget(canonicalType, payload),
	}

	timeout := 5 * time.Minute
	if opts.wait {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	label := repositoryTypeDisplay(canonicalType)
	if label == "" {
		label = "Object repository"
	}

	repoName := strings.TrimSpace(payload["name"].(string))
	if repoName == "" {
		repoName = "(unnamed)"
	}

	if !opts.yes {
		targetText := infoDescription(canonicalType, repoName, info)
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %s", label, targetText))
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled.\n")
			return nil
		}
	}

	session, err := httpClient.CreateRepository(ctx, payload)
	if err != nil {
		return fmt.Errorf("create repository %q: %w", repoName, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for repository provisioning session: %w", waitErr)
		}
		session = final
		waited = true
	}

	location := infoLocation(canonicalType, info)
	return printRepositoryProvisioningResult(cmd, session, canonicalType, repoName, location, waited)
}

type objectRepositoryEditOptions struct {
	targetName  string
	targetID    string
	typeHint    string
	specPath    string
	setPairs    []string
	newName     string
	description string
	disable     bool
	wait        bool
	yes         bool
}

func objectRepositoryEditCmd() *cobra.Command {
	opts := objectRepositoryEditOptions{}

	cmd := &cobra.Command{
		Use:   cmdObjectRepositoryUse,
		Short: "Edit an object storage repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			disableSet := cmd.Flags().Changed("disable")
			return runObjectRepositoryEdit(cmd, &opts, disableSet)
		},
	}

	cmd.Flags().StringVar(&opts.targetName, "name", "", "Repository name to edit")
	cmd.Flags().StringVar(&opts.targetID, "id", "", "Repository ID to edit")
	cmd.Flags().StringVar(&opts.typeHint, "type", "", "Expected repository type (optional)")
	cmd.Flags().StringVar(&opts.specPath, "spec", "", "Path to JSON spec used as the base payload (optional)")
	cmd.Flags().StringSliceVar(&opts.setPairs, "set", nil, "Override value (repeatable, dot notation)")
	cmd.Flags().StringVar(&opts.newName, "new-name", "", "New repository name")
	cmd.Flags().StringVar(&opts.description, "description", "", "New description")
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Disable the repository (set to true or false)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the repository reconfiguration session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required to execute)")

	return cmd
}

func runObjectRepositoryEdit(cmd *cobra.Command, opts *objectRepositoryEditOptions, disableSet bool) error {
	nameTrim := strings.TrimSpace(opts.targetName)
	idTrim := strings.TrimSpace(opts.targetID)
	if nameTrim == "" && idTrim == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if nameTrim != "" && idTrim != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	timeout := 5 * time.Minute
	if opts.wait {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	target, err := resolveObjectRepositoryTarget(ctx, httpClient, nameTrim, idTrim, opts.typeHint)
	if err != nil {
		return err
	}

	var payload map[string]any
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return err
		}
	} else {
		payload, err = deepCopyMap(target.Detail)
		if err != nil {
			return err
		}
	}

	repoType := target.Type
	if v, ok := payload["type"].(string); ok && strings.TrimSpace(v) != "" {
		canonical, err := normalizeObjectRepositoryType(v)
		if err != nil {
			return err
		}
		repoType = canonical
	}
	if strings.TrimSpace(opts.typeHint) != "" {
		canonical, err := normalizeObjectRepositoryType(opts.typeHint)
		if err != nil {
			return err
		}
		repoType = canonical
	}
	canonicalType, err := normalizeObjectRepositoryType(repoType)
	if err != nil {
		return err
	}
	payload["type"] = canonicalType
	payload["id"] = target.ID

	repoName := target.Name
	if cmd.Flags().Changed("new-name") {
		trimmed := strings.TrimSpace(opts.newName)
		if trimmed == "" {
			return fmt.Errorf("--new-name cannot be empty")
		}
		payload["name"] = trimmed
		repoName = trimmed
	} else if v, ok := payload["name"].(string); ok && strings.TrimSpace(v) != "" {
		repoName = strings.TrimSpace(v)
	}

	if cmd.Flags().Changed("description") {
		payload["description"] = opts.description
	} else if _, ok := payload["description"].(string); !ok {
		payload["description"] = target.State.Description
	}

	if disableSet {
		payload["isDisabled"] = opts.disable
	}

	overrides, err := parseOverridePairs(opts.setPairs)
	if err != nil {
		return err
	}
	if len(overrides) > 0 {
		if err := applyOverridesToMap(payload, overrides); err != nil {
			return err
		}
	}

	info := repositoryCreationResult{
		objectTarget: deriveObjectTarget(canonicalType, payload),
	}

	label := repositoryTypeDisplay(canonicalType)
	if label == "" {
		label = "Object repository"
	}

	if !opts.yes {
		targetText := infoDescription(canonicalType, repoName, info)
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Update %s %s", label, targetText))
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled updating %s %q.\n", label, target.Name)
			return nil
		}
	}

	session, err := httpClient.UpdateRepository(ctx, target.ID, payload)
	if err != nil {
		return fmt.Errorf("update repository %q (%s): %w", target.Name, target.ID, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for repository reconfiguration session: %w", waitErr)
		}
		session = final
		waited = true
	}

	location := infoLocation(canonicalType, info)
	return printRepositoryUpdateResult(cmd, session, canonicalType, repoName, location, waited)
}

type objectRepositoryRow struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
	Online      string `json:"Online"`
	ID          string `json:"Id"`
}

type objectRepositoryDetail struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
	Online      string `json:"Online"`
	Target      string `json:"Target,omitempty"`
	ID          string `json:"Id"`
	Config      string `json:"Config"`
}
