package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/internal/jobconfig"
	"github.com/veeamgo/veeamgo/internal/paths"
	jobtemplates "github.com/veeamgo/veeamgo/internal/templates/job"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func jobCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdJobUse,
		Short: jobDescriptionWithTypes("Job inventory and details"),
		Long:  jobLongDescription("Job inventory and details"),
	}
	root.AddCommand(jobGetCmd())
	root.AddCommand(jobDescribeCmd())
	root.AddCommand(jobAddCmd())
	root.AddCommand(jobEditCmd())
	return root
}

type jobFilterOptions struct {
	name         string
	types        []string
	status       string
	result       string
	workload     string
	repository   string
	highPriority bool
	since        string
	before       string
	afterJob     string
	limit        int
}

type jobHistoryOptions struct {
	name          string
	types         []string
	states        []string
	results       []string
	createdAfter  string
	createdBefore string
	endedAfter    string
	endedBefore   string
	sort          string
	desc          bool
	limit         int
}

func (opts jobFilterOptions) selectedTypes() ([]string, error) {
	return normalizeJobTypes(opts.types)
}

func jobGetCmd() *cobra.Command {
	opts := jobFilterOptions{}

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: jobDescriptionWithTypes("List jobs with current state"),
		Long:  jobLongDescription("List jobs with current state"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter, err := buildJobStatesFilter(ctx, httpClient, opts)
			if err != nil {
				return err
			}

			states, err := httpClient.JobStates(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, states)
			}

			rows := make([]jobStateRow, 0, len(states))
			for _, st := range states {
				rows = append(rows, jobStateRow{
					Name:       st.Name,
					Type:       jobTypeDisplay(st.Type),
					Status:     jobStatusDisplay(st.Status),
					LastResult: st.LastResult,
					LastRun:    formatTimestamp(st.LastRun),
					NextRun:    formatTimestamp(st.NextRun),
					Repository: st.RepositoryName,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by job name (supports * wildcards)")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "Filter by job type (repeatable, accepts EJobType values)")
	cmd.Flags().StringVar(&opts.status, "status", "", "Filter by current job status (e.g. Running, Stopped)")
	cmd.Flags().StringVar(&opts.result, "result", "", "Filter by last result (e.g. Success, Warning, Failed)")
	cmd.Flags().StringVar(&opts.workload, "workload", "", "Filter by workload (e.g. Vmware, HyperV)")
	cmd.Flags().StringVar(&opts.repository, "repository", "", "Filter by target repository name")
	cmd.Flags().BoolVar(&opts.highPriority, "high-priority", false, "Only include jobs marked high priority")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only include jobs with last run ≥ RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.before, "before", "", "Only include jobs with last run ≤ RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.afterJob, "after-job", "", "Only include jobs chained after the specified job name")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of jobs to return (default: all)")

	cmd.AddCommand(jobHistoryCmd())
	return cmd
}

func jobDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <name>", cmdDescribeUse),
		Short: jobDescriptionWithTypes("Show detailed job configuration and state"),
		Long:  jobLongDescription("Show detailed job configuration and state"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			state, err := httpClient.JobStateByName(ctx, args[0])
			if err != nil {
				return err
			}

			config, err := httpClient.Job(ctx, state.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, config)
			}

			disabled := extractBool(config, "isDisabled")
			runAfter := ""
			if state.RunAfterJob != nil {
				switch {
				case state.RunAfterJob.JobName != "" && state.RunAfterJob.JobID != "":
					runAfter = fmt.Sprintf("%s (%s)", state.RunAfterJob.JobName, state.RunAfterJob.JobID)
				case state.RunAfterJob.JobName != "":
					runAfter = state.RunAfterJob.JobName
				case state.RunAfterJob.JobID != "":
					runAfter = state.RunAfterJob.JobID
				}
			}

			detail := jobDetail{
				Name:          state.Name,
				Type:          jobTypeDisplay(state.Type),
				Workload:      state.Workload,
				Description:   state.Description,
				Status:        jobStatusDisplay(state.Status),
				LastResult:    state.LastResult,
				LastRun:       formatTimestamp(state.LastRun),
				NextRun:       formatTimestamp(state.NextRun),
				NextRunPolicy: state.NextRunPolicy,
				Repository:    formatRepo(state.RepositoryName, state.RepositoryID),
				Objects:       state.ObjectsCount,
				HighPriority:  yesNo(state.HighPriority),
				LastSession:   state.SessionID,
				RunAfter:      runAfter,
				Disabled:      yesNo(disabled),
				Config:        formatYAMLBlock(config),
			}

			return output.Print(format, detail)
		},
	}
	return cmd
}

func jobAddCmd() *cobra.Command {
	return newJobAddCommand("add")
}

func jobAddVerbCmd() *cobra.Command {
	return newJobAddCommand("job")
}

func newJobAddCommand(use string) *cobra.Command {
	opts := struct {
		specPath        string
		overrides       []string
		assumeYes       bool
		dryRun          bool
		templateType    string
		templateVariant string
	}{}

	cmd := &cobra.Command{
		Use:   use,
		Short: jobDescriptionWithTypes("Create a new job"),
		Long:  jobLongDescription("Create a new job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides, err := parseOverridePairs(opts.overrides)
			if err != nil {
				return err
			}

			blueprint, _, err := loadBlueprintSources(opts.specPath, opts.templateType, opts.templateVariant)
			if err != nil {
				return err
			}

			if err := blueprint.ApplyOverrides(overrides); err != nil {
				return err
			}
			if err := blueprint.Validate(jobconfig.ModeCreate); err != nil {
				return err
			}

			payload := blueprint.CreatePayload()
			summary := jobBlueprintSummary(payload)

			if opts.dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run – resolved blueprint for %s:\n", summary)
				return renderBlueprint(cmd, payload)
			}

			if !opts.assumeYes {
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Create job %s", summary))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintf(cmd.OutOrStdout(), "Cancelled creating %s.\n", summary)
					return nil
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			created, err := httpClient.CreateJob(ctx, payload)
			if err != nil {
				return fmt.Errorf("create job %s: %w", summary, err)
			}

			name := firstNonEmpty(extractJobName(created), extractJobName(payload))
			id := firstNonEmpty(extractJobID(created), extractJobID(payload))
			jobType := jobTypeDisplay(firstNonEmpty(extractJobType(created), extractJobType(payload)))
			target := jobTargetLabel(name, id)

			switch {
			case target != "" && jobType != "":
				fmt.Fprintf(cmd.OutOrStdout(), "Created job %q (%s).\n", target, jobType)
			case target != "":
				fmt.Fprintf(cmd.OutOrStdout(), "Created job %q.\n", target)
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "Created job successfully.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.specPath, "from", "", "Path to job blueprint (YAML or JSON)")
	cmd.Flags().StringArrayVar(&opts.overrides, "set", nil, "Override blueprint value (repeatable, format key=value)")
	cmd.Flags().StringVar(&opts.templateType, "template", "", "Seed from a built-in job template (EJobType value)")
	cmd.Flags().StringVar(&opts.templateVariant, "template-variant", string(jobtemplates.VariantMinimal), "Template variant (e.g. minimal)")
	cmd.Flags().BoolVar(&opts.assumeYes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate and print the resolved blueprint without calling the API")

	return cmd
}

func jobEditCmd() *cobra.Command {
	return newJobEditCommand("edit")
}

func jobEditVerbCmd() *cobra.Command {
	return newJobEditCommand("job")
}

func newJobEditCommand(use string) *cobra.Command {
	opts := struct {
		jobName         string
		jobID           string
		specPath        string
		overrides       []string
		assumeYes       bool
		dryRun          bool
		fromLive        bool
		templatePath    string
		templateType    string
		templateVariant string
	}{}

	cmd := &cobra.Command{
		Use:   use,
		Short: jobDescriptionWithTypes("Edit an existing job"),
		Long:  jobLongDescription("Edit an existing job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.fromLive && strings.TrimSpace(opts.specPath) != "" {
				return fmt.Errorf("--from-live cannot be combined with --from in the same invocation")
			}
			if opts.fromLive && strings.TrimSpace(opts.templateType) != "" {
				return fmt.Errorf("--from-live cannot be combined with --template")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			jobID, jobName, state, config, err := resolveJobTarget(ctx, httpClient, opts.jobName, opts.jobID)
			if err != nil {
				return err
			}

			if jobStateIndicatesActive(state) {
				target := jobTargetLabel(jobName, jobID)
				status := jobStatusDisplay(state.Status)
				if status == "" {
					status = "active"
				}
				return fmt.Errorf("job %q is currently %s; wait for the session to finish or re-run with --wait", target, strings.ToLower(status))
			}

			if opts.fromLive {
				template := jobconfig.FromExisting(config)
				path := strings.TrimSpace(opts.templatePath)
				if path == "" {
					path, err = paths.JobTemplatePath(jobID)
					if err != nil {
						return err
					}
				}
				if err := writeBlueprintFile(path, template.Spec); err != nil {
					return fmt.Errorf("write blueprint template: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote job blueprint to %s.\n", path)
				return nil
			}

			interactiveEdit := strings.TrimSpace(opts.specPath) == "" &&
				len(opts.overrides) == 0 &&
				strings.TrimSpace(opts.templateType) == "" &&
				!opts.fromLive

			var blueprint *jobconfig.Blueprint
			if interactiveEdit {
				original := jobconfig.FromExisting(config)
				edited, changed, err := editBlueprintInteractively(cmd, original)
				if err != nil {
					return err
				}
				if !changed {
					fmt.Fprintln(cmd.OutOrStdout(), "No changes detected; nothing to do.")
					return nil
				}
				blueprint = edited
			} else {
				overrides, err := parseOverridePairs(opts.overrides)
				if err != nil {
					return err
				}

				blueprintSource, provided, err := loadBlueprintSources(opts.specPath, opts.templateType, opts.templateVariant)
				if err != nil {
					return err
				}

				if provided {
					blueprint = blueprintSource
				} else {
					blueprint = jobconfig.FromExisting(config)
				}

				if err := blueprint.ApplyOverrides(overrides); err != nil {
					return err
				}
			}

			if err := blueprint.Validate(jobconfig.ModeUpdate); err != nil {
				return err
			}

			payload, err := blueprint.UpdatePayload(config)
			if err != nil {
				return err
			}

			summary := jobBlueprintSummary(payload)

			if opts.dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run – updated blueprint for %s:\n", summary)
				return renderBlueprint(cmd, payload)
			}

			if !opts.assumeYes {
				target := jobTargetLabel(jobName, jobID)
				if target == "" {
					target = summary
				}
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Update job %s", target))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintf(cmd.OutOrStdout(), "Cancelled updating %s.\n", target)
					return nil
				}
			}

			updated, err := httpClient.UpdateJob(ctx, jobID, payload)
			if err != nil {
				target := jobTargetLabel(jobName, jobID)
				if target == "" {
					target = summary
				}
				return fmt.Errorf("update job %s: %w", target, err)
			}

			name := firstNonEmpty(extractJobName(updated), jobName)
			jobType := jobTypeDisplay(firstNonEmpty(extractJobType(updated), extractJobType(payload)))
			target := jobTargetLabel(name, jobID)

			switch {
			case target != "" && jobType != "":
				fmt.Fprintf(cmd.OutOrStdout(), "Updated job %q (%s).\n", target, jobType)
			case target != "":
				fmt.Fprintf(cmd.OutOrStdout(), "Updated job %q.\n", target)
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "Updated job successfully.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&opts.jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().StringVar(&opts.specPath, "from", "", "Path to job blueprint (YAML or JSON)")
	cmd.Flags().StringArrayVar(&opts.overrides, "set", nil, "Override blueprint value (repeatable, format key=value)")
	cmd.Flags().BoolVar(&opts.assumeYes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate and print the resolved blueprint without calling the API")
	cmd.Flags().BoolVar(&opts.fromLive, "from-live", false, "Export the current job configuration to a blueprint and exit")
	cmd.Flags().StringVar(&opts.templatePath, "template-path", "", "Destination for --from-live blueprint (defaults to project cache)")
	cmd.Flags().StringVar(&opts.templateType, "template", "", "Seed overrides from a built-in job template (EJobType value)")
	cmd.Flags().StringVar(&opts.templateVariant, "template-variant", string(jobtemplates.VariantMinimal), "Template variant to seed when using --template")

	return cmd
}

func jobHistoryCmd() *cobra.Command {
	opts := jobHistoryOptions{}

	cmd := &cobra.Command{
		Use:   "history",
		Short: jobDescriptionWithTypes("List session history for a job"),
		Long:  jobLongDescription("List session history for a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.name) == "" {
				return requireFlag("--name", "provide the job name to inspect")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			job, err := httpClient.JobStateByName(ctx, opts.name)
			if err != nil {
				return fmt.Errorf("resolve job %q: %w", opts.name, err)
			}

			filter := client.SessionsFilter{
				JobID:    job.ID,
				Types:    opts.types,
				States:   opts.states,
				Results:  opts.results,
				MaxItems: opts.limit,
			}

			if opts.sort != "" {
				filter.OrderColumn = opts.sort
				if opts.desc {
					value := false
					filter.OrderAscending = &value
				}
			}

			if opts.createdAfter != "" {
				ts, err := parseTimeFlag(opts.createdAfter)
				if err != nil {
					return err
				}
				filter.CreatedAfter = ts
			}
			if opts.createdBefore != "" {
				ts, err := parseTimeFlag(opts.createdBefore)
				if err != nil {
					return err
				}
				filter.CreatedBefore = ts
			}
			if opts.endedAfter != "" {
				ts, err := parseTimeFlag(opts.endedAfter)
				if err != nil {
					return err
				}
				filter.EndedAfter = ts
			}
			if opts.endedBefore != "" {
				ts, err := parseTimeFlag(opts.endedBefore)
				if err != nil {
					return err
				}
				filter.EndedBefore = ts
			}

			sessions, err := httpClient.Sessions(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, sessions)
			}

			rows := make([]sessionRow, 0, len(sessions))
			for _, sess := range sessions {
				rows = append(rows, sessionRow{
					Name:      sess.Name,
					Type:      sess.SessionType,
					State:     sess.State,
					Result:    sessionResultShort(sess.Result),
					Progress:  fmt.Sprintf("%d%%", sess.ProgressPercent),
					CreatedAt: formatTimestampValue(sess.CreationTime),
					EndedAt:   formatTimestamp(sess.EndTime),
					SessionID: sess.ID,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Job name to inspect")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "Filter by session type (repeatable)")
	cmd.Flags().StringSliceVar(&opts.states, "state", nil, "Filter by session state")
	cmd.Flags().StringSliceVar(&opts.results, "result", nil, "Filter by session result")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Include sessions created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Include sessions created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.endedAfter, "ended-since", "", "Include sessions ended on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.endedBefore, "ended-before", "", "Include sessions ended on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort sessions by column (e.g. creationTime, endTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order (default ascending)")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of sessions to return (default: all)")

	return cmd
}

func jobEnableCmd() *cobra.Command {
	return newJobToggleCommand("enable", "Enabled", false, func(ctx context.Context, c *client.Client, id string) error {
		return c.EnableJob(ctx, id)
	})
}

func jobDisableCmd() *cobra.Command {
	return newJobToggleCommand("disable", "Disabled", true, func(ctx context.Context, c *client.Client, id string) error {
		return c.DisableJob(ctx, id)
	})
}

func jobStartCmd() *cobra.Command {
	var (
		jobName           string
		jobID             string
		yes               bool
		performActiveFull bool
		startChained      bool
		syncRestorePoints string
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes("Start a job"),
		Long:  jobLongDescription("Start a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobStart(cmd, jobName, jobID, yes, performActiveFull, startChained, syncRestorePoints)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&performActiveFull, "active-full", false, "Perform an active full backup")
	cmd.Flags().BoolVar(&startChained, "start-chained", false, "Start chained jobs as well")
	cmd.Flags().StringVar(&syncRestorePoints, "sync-restore-points", "", "For backup copy jobs: sync restore points (All|Latest)")

	return cmd
}

func jobStopCmd() *cobra.Command {
	var (
		jobName       string
		jobID         string
		yes           bool
		cancelChained bool
		graceful      = true
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes("Stop a job"),
		Long:  jobLongDescription("Stop a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobStop(cmd, jobName, jobID, yes, graceful, cancelChained)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&graceful, "graceful", graceful, "Perform a graceful stop")
	cmd.Flags().BoolVar(&cancelChained, "cancel-chained", false, "Cancel chained jobs as well")

	return cmd
}

func jobRetryCmd() *cobra.Command {
	var (
		jobName      string
		jobID        string
		yes          bool
		startChained bool
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes("Retry a job"),
		Long:  jobLongDescription("Retry a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobRetry(cmd, jobName, jobID, yes, startChained)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&startChained, "start-chained", false, "Start chained jobs as well")

	return cmd
}

func jobQuickBackupCmd() *cobra.Command {
	var (
		jobName string
		jobID   string
		yes     bool
		vmName  string
	)

	cmd := &cobra.Command{
		Use:   "quickbackup",
		Short: jobDescriptionWithTypes("Start a quick backup"),
		Long:  jobLongDescription("Start a quick backup"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobQuickBackup(cmd, jobName, jobID, yes, vmName)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().StringVar(&vmName, "vm-name", "", "Virtual machine name to protect")

	return cmd
}

func jobCloneCmd() *cobra.Command {
	var (
		jobName string
		jobID   string
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes("Clone a job"),
		Long:  jobLongDescription("Clone a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobClone(cmd, jobName, jobID, yes)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func jobDeleteCmd() *cobra.Command {
	var (
		jobName string
		jobID   string
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes("Delete a job"),
		Long:  jobLongDescription("Delete a job"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobDelete(cmd, jobName, jobID, yes)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func newJobToggleCommand(action, actionPast string, desiredDisabled bool, apiCall func(context.Context, *client.Client, string) error) *cobra.Command {
	var (
		jobName string
		jobID   string
		yes     bool
	)

	label := action
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}

	cmd := &cobra.Command{
		Use:   "job",
		Short: jobDescriptionWithTypes(fmt.Sprintf("%s a job", label)),
		Long:  jobLongDescription(fmt.Sprintf("%s a job", label)),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobToggle(cmd, jobName, jobID, yes, action, actionPast, desiredDisabled, apiCall)
		},
	}

	cmd.Flags().StringVar(&jobName, "name", "", "Job name to target (optional)")
	cmd.Flags().StringVar(&jobID, "id", "", "Job ID to target (optional)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm without prompting (required to execute)")
	return cmd
}

func runJobToggle(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes bool, action, actionPast string, desiredDisabled bool, apiCall func(context.Context, *client.Client, string) error) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, config, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	if jobStateIndicatesActive(state) {
		target := jobTargetLabel(jobName, jobID)
		status := ""
		if state != nil {
			status = jobStatusDisplay(state.Status)
		}
		if strings.TrimSpace(status) == "" {
			status = "active"
		}
		return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to %s it", target, strings.ToLower(status), action)
	}

	currentlyDisabled := extractBool(config, "isDisabled")
	if currentlyDisabled == desiredDisabled {
		stateLabel := "enabled"
		if desiredDisabled {
			stateLabel = "disabled"
		}
		target := jobTargetLabel(jobName, jobID)
		fmt.Fprintf(cmd.OutOrStdout(), "Job %q is already %s.\n", target, stateLabel)
		return nil
	}

	target := jobTargetLabel(jobName, jobID)
	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Proceed to %s job %q", action, target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled %s job %q.\n", action, target)
			return nil
		}
	}

	if err := apiCall(ctx, httpClient, jobID); err != nil {
		return fmt.Errorf("%s job %q (%s): %w", action, target, jobID, err)
	}

	if target == jobID {
		fmt.Fprintf(cmd.OutOrStdout(), "%s job %s.\n", actionPast, target)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s job %q (%s).\n", actionPast, target, jobID)
	}
	return nil
}

func runJobStart(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes, performActiveFull, startChained bool, syncRestorePoints string) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	normalizedSync, err := normalizeSyncRestorePoints(syncRestorePoints)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, config, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	target := jobTargetLabel(jobName, jobID)

	if extractBool(config, "isDisabled") {
		return fmt.Errorf("job %q is disabled; enable it before starting", target)
	}

	if jobStateIndicatesActive(state) {
		status := ""
		if state != nil {
			status = jobStatusDisplay(state.Status)
		}
		if strings.TrimSpace(status) == "" {
			status = "active"
		}
		return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to start it", target, strings.ToLower(status))
	}

	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Start job %q", target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled starting job %q.\n", target)
			return nil
		}
	}

	spec := client.JobStartOptions{
		PerformActiveFull: performActiveFull,
	}
	if startChained {
		spec.StartChainedJobs = true
	}
	if normalizedSync != "" {
		spec.SyncRestorePoints = normalizedSync
	}

	session, err := httpClient.StartJob(ctx, jobID, spec)
	if err != nil {
		return fmt.Errorf("start job %q (%s): %w", target, jobID, err)
	}

	return printSessionMessage(cmd, session, fmt.Sprintf("Started job %q (%s).%s", target, jobID, formatSessionTail(session)))
}

func runJobStop(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes, graceful, cancelChained bool) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, _, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	target := jobTargetLabel(jobName, jobID)

	if !jobStateIndicatesActive(state) {
		fmt.Fprintf(cmd.OutOrStdout(), "Job %q is not currently running.\n", target)
		return nil
	}

	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Stop job %q", target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled stopping job %q.\n", target)
			return nil
		}
	}

	spec := client.JobStopOptions{
		GracefulStop: graceful,
	}
	if cancelChained {
		spec.CancelChainedJobs = true
	}

	session, err := httpClient.StopJob(ctx, jobID, spec)
	if err != nil {
		return fmt.Errorf("stop job %q (%s): %w", target, jobID, err)
	}

	return printSessionMessage(cmd, session, fmt.Sprintf("Initiated stop for job %q (%s).%s", target, jobID, formatSessionTail(session)))
}

func runJobRetry(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes, startChained bool) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, config, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	target := jobTargetLabel(jobName, jobID)

	if extractBool(config, "isDisabled") {
		return fmt.Errorf("job %q is disabled; enable it before retrying", target)
	}

	if jobStateIndicatesActive(state) {
		status := ""
		if state != nil {
			status = jobStatusDisplay(state.Status)
		}
		if strings.TrimSpace(status) == "" {
			status = "active"
		}
		return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to retry it", target, strings.ToLower(status))
	}

	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Retry job %q", target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled retrying job %q.\n", target)
			return nil
		}
	}

	spec := client.JobRetryOptions{}
	if startChained {
		spec.StartChainedJobs = true
	}

	session, err := httpClient.RetryJob(ctx, jobID, spec)
	if err != nil {
		return fmt.Errorf("retry job %q (%s): %w", target, jobID, err)
	}

	return printSessionMessage(cmd, session, fmt.Sprintf("Retry initiated for job %q (%s).%s", target, jobID, formatSessionTail(session)))
}

func runJobQuickBackup(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes bool, vmName string) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)
	vmNameTrimmed := strings.TrimSpace(vmName)

	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	if vmNameTrimmed == "" {
		return requireFlag("--vm-name", "provide the virtual machine name")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	var (
		jobID     string
		jobName   string
		state     *client.JobState
		config    map[string]any
		chosen    quickBackupCandidate
		cancelled bool
	)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		jobID, jobName, state, config, chosen, cancelled, err = autoResolveQuickBackupJob(ctx, cmd, httpClient, vmNameTrimmed, assumeYes)
		if err != nil {
			return err
		}
		if cancelled {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled quick backup for %q.\n", vmNameTrimmed)
			return nil
		}
	} else {
		jobID, jobName, state, config, err = resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
		if err != nil {
			return err
		}

		target := jobTargetLabel(jobName, jobID)

		if extractBool(config, "isDisabled") {
			return fmt.Errorf("job %q is disabled; enable it before running quick backup", target)
		}

		if jobStateIndicatesActive(state) {
			status := ""
			if state != nil {
				status = jobStatusDisplay(state.Status)
			}
			if strings.TrimSpace(status) == "" {
				status = "active"
			}
			return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to start a quick backup", target, strings.ToLower(status))
		}

		candidates := quickBackupCandidatesFromJobConfig(config)
		if len(candidates) == 0 {
			return fmt.Errorf("job %q does not include any VMware virtual machines eligible for quick backup", target)
		}

		match := findQuickBackupMatch(vmNameTrimmed, candidates)
		chosen, cancelled, err = resolveCandidateFromMatch(cmd, target, vmNameTrimmed, match, assumeYes)
		if err != nil {
			return err
		}
		if cancelled {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled quick backup for %q via job %q.\n", vmNameTrimmed, target)
			return nil
		}
	}

	target := jobTargetLabel(jobName, jobID)

	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Start quick backup for %q via job %q", chosen.Request.Name, target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled quick backup for %q via job %q.\n", chosen.Request.Name, target)
			return nil
		}
	}

	session, err := httpClient.StartQuickBackupVSphere(ctx, chosen.Request)
	if err != nil {
		return fmt.Errorf("start quick backup for %q via job %q (%s): %w", chosen.Request.Name, target, jobID, err)
	}

	message := fmt.Sprintf("Started quick backup for %q on %q via job %q (%s).%s", chosen.Request.Name, chosen.Request.HostName, target, jobID, formatSessionTail(session))
	return printSessionMessage(cmd, session, message)
}
func resolveJobTarget(ctx context.Context, httpClient *client.Client, jobNameFlag, jobIDFlag string) (string, string, *client.JobState, map[string]any, error) {
	jobName := strings.TrimSpace(jobNameFlag)
	jobID := strings.TrimSpace(jobIDFlag)

	if jobName != "" && jobID != "" {
		return "", "", nil, nil, fmt.Errorf("provide either --name or --id, not both")
	}

	if jobID != "" {
		state, err := jobStateByID(ctx, httpClient, jobID)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("fetch job state %s: %w", jobID, err)
		}
		config, err := httpClient.Job(ctx, jobID)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("fetch job %s: %w", jobID, err)
		}
		name := firstNonEmpty(extractJobName(config), jobStateName(state))
		return jobID, name, state, config, nil
	}

	if jobName == "" {
		return "", "", nil, nil, fmt.Errorf("provide --name or --id")
	}

	state, err := httpClient.JobStateByName(ctx, jobName)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("resolve job %q: %w", jobName, err)
	}

	config, err := httpClient.Job(ctx, state.ID)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("fetch job %q: %w", state.Name, err)
	}

	return state.ID, firstNonEmpty(extractJobName(config), state.Name), state, config, nil
}

func runJobClone(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes bool) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, _, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	if jobStateIndicatesActive(state) {
		target := jobTargetLabel(jobName, jobID)
		status := ""
		if state != nil {
			status = jobStatusDisplay(state.Status)
		}
		if strings.TrimSpace(status) == "" {
			status = "active"
		}
		return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to clone it", target, strings.ToLower(status))
	}

	target := jobTargetLabel(jobName, jobID)
	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Clone job %q", target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled cloning job %q.\n", target)
			return nil
		}
	}

	cloned, err := httpClient.CloneJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("clone job %q (%s): %w", target, jobID, err)
	}

	newID := strings.TrimSpace(extractJobID(cloned))
	newName := strings.TrimSpace(extractJobName(cloned))
	newType := strings.TrimSpace(jobTypeDisplay(extractJobType(cloned)))
	newTarget := jobTargetLabel(newName, newID)

	switch {
	case newID != "" && newType != "":
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned job %q (%s) to %q (%s) [%s].\n", target, jobID, newTarget, newID, newType)
	case newID != "":
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned job %q (%s) to %q (%s).\n", target, jobID, newTarget, newID)
	case newType != "":
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned job %q (%s) to %q [%s].\n", target, jobID, newTarget, newType)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned job %q (%s) to %q.\n", target, jobID, newTarget)
	}
	return nil
}

func runJobDelete(cmd *cobra.Command, jobNameFlag, jobIDFlag string, assumeYes bool) error {
	jobNameTrimmed := strings.TrimSpace(jobNameFlag)
	jobIDTrimmed := strings.TrimSpace(jobIDFlag)

	if jobNameTrimmed == "" && jobIDTrimmed == "" {
		return fmt.Errorf("provide --name or --id")
	}
	if jobNameTrimmed != "" && jobIDTrimmed != "" {
		return fmt.Errorf("provide either --name or --id, not both")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	jobID, jobName, state, _, err := resolveJobTarget(ctx, httpClient, jobNameFlag, jobIDFlag)
	if err != nil {
		return err
	}

	if jobStateIndicatesActive(state) {
		target := jobTargetLabel(jobName, jobID)
		status := ""
		if state != nil {
			status = jobStatusDisplay(state.Status)
		}
		if strings.TrimSpace(status) == "" {
			status = "active"
		}
		return fmt.Errorf("job %q is currently %s; wait for the session to finish before attempting to delete it", target, strings.ToLower(status))
	}

	target := jobTargetLabel(jobName, jobID)
	if !assumeYes {
		confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Permanently delete job %q", target))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(cmd.OutOrStdout(), "Cancelled deleting job %q.\n", target)
			return nil
		}
	}

	if err := httpClient.DeleteJob(ctx, jobID); err != nil {
		return fmt.Errorf("delete job %q (%s): %w", target, jobID, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted job %q (%s).\n", target, jobID)
	return nil
}

func jobStateByID(ctx context.Context, httpClient *client.Client, jobID string) (*client.JobState, error) {
	states, err := httpClient.JobStates(ctx, client.JobStatesFilter{
		ID:       strings.TrimSpace(jobID),
		MaxItems: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, fmt.Errorf("job with id %q not found", jobID)
	}
	return &states[0], nil
}

func jobStateName(state *client.JobState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.Name)
}

func jobStateIndicatesActive(state *client.JobState) bool {
	if state == nil {
		return false
	}

	status := strings.ToLower(strings.TrimSpace(state.Status))
	if status != "" {
		activeTokens := []string{
			"running",
			"stopping",
			"starting",
			"inprogress",
			"processing",
			"rescan",
			"working",
			"waiting",
		}
		for _, token := range activeTokens {
			if strings.Contains(status, token) {
				// Waiting for resource (e.g. waiting repository) is still an active session.
				if token != "waiting" || (token == "waiting" && !strings.Contains(status, "for new backup cycle")) {
					return true
				}
			}
		}
	}

	if strings.TrimSpace(state.SessionID) != "" {
		if status == "" {
			return true
		}
		safeStatus := []string{"stopped", "disabled", "idle", "scheduled", "none"}
		for _, safe := range safeStatus {
			if status == safe {
				return false
			}
		}
		return true
	}

	if state.SessionProgress != nil {
		return true
	}

	return false
}

func jobTargetLabel(jobName, jobID string) string {
	if strings.TrimSpace(jobName) != "" {
		return strings.TrimSpace(jobName)
	}
	return strings.TrimSpace(jobID)
}

func promptForConfirmation(cmd *cobra.Command, question string) (bool, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	for {
		fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, fmt.Errorf("read confirmation: %w", err)
		}

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return false, nil
		}

		switch strings.ToLower(trimmed) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "Please respond with y or n.")
			if errors.Is(err, io.EOF) {
				return false, nil
			}
		}
	}
}
func extractJobName(config map[string]any) string {
	if config == nil {
		return ""
	}
	if raw, ok := config["name"]; ok {
		if name, ok := raw.(string); ok {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

func extractJobID(config map[string]any) string {
	if config == nil {
		return ""
	}
	if raw, ok := config["id"]; ok {
		if id, ok := raw.(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func extractJobType(config map[string]any) string {
	if config == nil {
		return ""
	}
	for _, key := range []string{"type", "jobType"} {
		if raw, ok := config[key]; ok {
			if value, ok := raw.(string); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func loadBlueprintSources(specPath, templateType, templateVariant string) (*jobconfig.Blueprint, bool, error) {
	cleanPath := strings.TrimSpace(specPath)
	cleanTemplate := strings.TrimSpace(templateType)
	if cleanPath != "" && cleanTemplate != "" {
		return nil, false, fmt.Errorf("provide either --from or --template, not both")
	}
	if cleanPath != "" {
		bp, err := jobconfig.LoadFile(cleanPath)
		if err != nil {
			return nil, false, err
		}
		return bp, true, nil
	}
	if cleanTemplate != "" {
		variant := jobtemplates.NormalizeVariant(templateVariant)
		if !variantSupported(cleanTemplate, variant) {
			available := jobtemplates.VariantsFor(cleanTemplate)
			if len(available) == 0 {
				return nil, false, fmt.Errorf("job type %q is not supported; see 'veeamgo template job --list'", cleanTemplate)
			}
			return nil, false, fmt.Errorf("variant %q is not available for %q (supported: %s)", variant, cleanTemplate, joinVariants(available))
		}
		spec, err := jobtemplates.Spec(cleanTemplate, variant)
		if err != nil {
			return nil, false, err
		}
		return &jobconfig.Blueprint{Spec: spec}, true, nil
	}
	return &jobconfig.Blueprint{Spec: map[string]any{}}, false, nil
}

func parseOverridePairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(pairs))
	for _, entry := range pairs {
		if !strings.Contains(entry, "=") {
			return nil, fmt.Errorf("override %q must follow key=value format", entry)
		}
		parts := strings.SplitN(entry, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("override %q has empty key", entry)
		}
		result[key] = parts[1]
	}
	return result, nil
}

func renderBlueprint(cmd *cobra.Command, payload map[string]any) error {
	if payload == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "No blueprint data.")
		return nil
	}
	format := outputFormat()
	if format == "json" {
		return output.Print(format, payload)
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode blueprint: %w", err)
	}
	text := strings.TrimRight(string(data), "\n")
	fmt.Fprintln(cmd.OutOrStdout(), text)
	return nil
}

func writeBlueprintFile(path string, spec map[string]any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("ensure template directory: %w", err)
	}
	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("encode blueprint: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write blueprint file: %w", err)
	}
	return nil
}

func editBlueprintInteractively(cmd *cobra.Command, original *jobconfig.Blueprint) (*jobconfig.Blueprint, bool, error) {
	if original == nil {
		return nil, false, errors.New("original blueprint is nil")
	}

	data, err := yaml.Marshal(original.Spec)
	if err != nil {
		return nil, false, fmt.Errorf("encode blueprint: %w", err)
	}

	tempFile, err := os.CreateTemp("", "veeamgo-job-edit-*.yaml")
	if err != nil {
		return nil, false, fmt.Errorf("create temporary blueprint: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	initialContent := append(append([]byte{}, data...), '\n')
	if _, err := tempFile.Write(initialContent); err != nil {
		tempFile.Close()
		return nil, false, fmt.Errorf("write temporary blueprint: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, false, fmt.Errorf("close temporary blueprint: %w", err)
	}

	editorCmd, err := resolveEditorCommand()
	if err != nil {
		return nil, false, err
	}

	execArgs := append([]string{}, editorCmd...)
	execArgs = append(execArgs, tempPath)

	if len(editorCmd) == 0 {
		return nil, false, errors.New("no editor configured")
	}

	editor := exec.Command(editorCmd[0], execArgs[1:]...) //nolint:gosec // editor is user-controlled
	editor.Stdin = os.Stdin
	editor.Stdout = os.Stdout
	editor.Stderr = os.Stderr

	if err := editor.Run(); err != nil {
		return nil, false, fmt.Errorf("launch editor: %w", err)
	}

	editedData, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, false, fmt.Errorf("read edited blueprint: %w", err)
	}
	trimmed := strings.TrimSpace(string(editedData))
	if trimmed == "" {
		return nil, false, nil
	}

	if bytes.Equal(initialContent, editedData) {
		return nil, false, nil
	}

	editedBlueprint, err := jobconfig.Parse(editedData)
	if err != nil {
		return nil, false, fmt.Errorf("parse edited blueprint: %w", err)
	}

	if reflect.DeepEqual(original.Spec, editedBlueprint.Spec) {
		return nil, false, nil
	}

	return editedBlueprint, true, nil
}

func resolveEditorCommand() ([]string, error) {
	for _, candidate := range []string{
		strings.TrimSpace(os.Getenv("VEEAMGO_EDITOR")),
		strings.TrimSpace(os.Getenv("VISUAL")),
		strings.TrimSpace(os.Getenv("EDITOR")),
	} {
		if candidate == "" {
			continue
		}
		parts := splitEditorCommand(candidate)
		if len(parts) > 0 {
			return parts, nil
		}
	}

	if runtime.GOOS == "windows" {
		return []string{"notepad"}, nil
	}
	for _, name := range []string{"vim", "nvim", "vi", "nano"} {
		if path, err := exec.LookPath(name); err == nil {
			return []string{path}, nil
		}
	}
	return nil, errors.New("no editor configured; set $VEEAMGO_EDITOR or $EDITOR")
}

func splitEditorCommand(command string) []string {
	var (
		result     []string
		current    strings.Builder
		quote      rune
		escapeNext bool
	)

	flush := func() {
		if current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
		}
	}

	for _, r := range command {
		switch {
		case escapeNext:
			current.WriteRune(r)
			escapeNext = false
		case r == '\\' && quote == '"':
			escapeNext = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return result
}

func jobBlueprintSummary(spec map[string]any) string {
	name := extractJobName(spec)
	jobType := jobTypeDisplay(extractJobType(spec))

	switch {
	case name != "" && jobType != "":
		return fmt.Sprintf("%q [%s]", name, jobType)
	case name != "":
		return fmt.Sprintf("%q", name)
	case jobType != "":
		return fmt.Sprintf("[%s]", jobType)
	default:
		return "job"
	}
}

func buildJobStatesFilter(ctx context.Context, httpClient *client.Client, opts jobFilterOptions) (client.JobStatesFilter, error) {
	types, err := opts.selectedTypes()
	if err != nil {
		return client.JobStatesFilter{}, err
	}

	filter := client.JobStatesFilter{
		Name:       opts.name,
		Types:      types,
		Status:     opts.status,
		LastResult: opts.result,
		Workload:   opts.workload,
		MaxItems:   opts.limit,
	}

	if opts.highPriority {
		value := true
		filter.HighPriority = &value
	}

	if opts.repository != "" {
		repo, err := httpClient.RepositoryStateByName(ctx, opts.repository)
		if err != nil {
			return filter, fmt.Errorf("resolve repository %q: %w", opts.repository, err)
		}
		filter.RepositoryID = repo.ID
	}

	if opts.afterJob != "" {
		after, err := httpClient.JobStateByName(ctx, opts.afterJob)
		if err != nil {
			return filter, fmt.Errorf("resolve after-job %q: %w", opts.afterJob, err)
		}
		filter.AfterJobID = after.ID
		filter.AfterJobName = after.Name
	}

	if opts.since != "" {
		parsed, err := time.Parse(time.RFC3339, opts.since)
		if err != nil {
			return filter, fmt.Errorf("parse --since: %w", err)
		}
		filter.LastRunAfter = &parsed
	}

	if opts.before != "" {
		parsed, err := time.Parse(time.RFC3339, opts.before)
		if err != nil {
			return filter, fmt.Errorf("parse --before: %w", err)
		}
		filter.LastRunBefore = &parsed
	}

	return filter, nil
}

func formatRepo(name, id string) string {
	switch {
	case name != "" && id != "":
		return fmt.Sprintf("%s (%s)", name, id)
	case name != "":
		return name
	case id != "":
		return id
	default:
		return ""
	}
}

func extractBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	val, ok := m[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || v == "1"
	default:
		return false
	}
}

type jobStateRow struct {
	Name       string `json:"Name"`
	Type       string `json:"Type"`
	Status     string `json:"Status"`
	LastResult string `json:"Last Result"`
	LastRun    string `json:"Last Run"`
	NextRun    string `json:"Next Run"`
	Repository string `json:"Repository"`
}

type jobDetail struct {
	Name          string `json:"Name"`
	Type          string `json:"Type"`
	Workload      string `json:"Workload"`
	Description   string `json:"Description"`
	Status        string `json:"Status"`
	LastResult    string `json:"Last Result"`
	LastRun       string `json:"Last Run"`
	NextRun       string `json:"Next Run"`
	NextRunPolicy string `json:"Next Run Policy"`
	Repository    string `json:"Repository"`
	Objects       int    `json:"Objects"`
	HighPriority  string `json:"High Priority"`
	LastSession   string `json:"Last Session"`
	RunAfter      string `json:"Run After"`
	Disabled      string `json:"Disabled"`
	Config        string `json:"Config"`
}

var (
	jobTypeCanonical = []string{
		"Unknown",
		"VSphereBackup",
		"HyperVBackup",
		"VSphereReplica",
		"CloudDirectorBackup",
		"EntraIDTenantBackup",
		"EntraIDAuditLogBackup",
		"FileBackupCopy",
		"LegacyBackupCopy",
		"BackupCopy",
		"WindowsAgentBackup",
		"LinuxAgentBackup",
		"EntraIDTenantBackupCopy",
		"NasBackup",
		"NasBackupCopy",
		"OracleRMANBackup",
		"SAPHANAPluginBackup",
		"AwsBackupCopy",
	}

	jobTypeLookup       = makeJobTypeLookup()
	jobTypeDisplayNames = makeJobTypeDisplayNames()
)

type quickBackupCandidate struct {
	Request  client.QuickBackupRequest
	Enabled  bool
	Eligible bool
	Reason   string
	Source   string
}

type quickBackupMatchKind int

const (
	quickBackupMatchNone quickBackupMatchKind = iota
	quickBackupMatchSingle
	quickBackupMatchMultiple
	quickBackupMatchDisabled
	quickBackupMatchIneligible
)

type quickBackupMatch struct {
	kind       quickBackupMatchKind
	candidates []quickBackupCandidate
}

type quickBackupJobOption struct {
	JobID   string
	JobName string
	State   *client.JobState
	Config  map[string]any
	Match   quickBackupMatch
}

type quickBackupJobFailure struct {
	JobName string
	Reason  string
}

func normalizeJobTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	var unknown []string

	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		key := normalizeJobTypeKey(trimmed)
		canonical, ok := jobTypeLookup[key]
		if !ok {
			unknown = append(unknown, trimmed)
			continue
		}
		if _, dup := seen[canonical]; dup {
			continue
		}
		seen[canonical] = struct{}{}
		result = append(result, canonical)
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unsupported job type(s) %s; supported values: %s", quoteList(unknown), strings.Join(jobTypeCanonical, ", "))
	}

	return result, nil
}

func makeJobTypeLookup() map[string]string {
	m := make(map[string]string, len(jobTypeCanonical)*3)
	for _, canonical := range jobTypeCanonical {
		if canonical == "" {
			continue
		}
		normalized := normalizeJobTypeKey(canonical)
		m[normalized] = canonical
		m[strings.ToLower(canonical)] = canonical
	}

	m["entratenantbackup"] = "EntraIDTenantBackup"
	m["entraauditlogbackup"] = "EntraIDAuditLogBackup"
	m["entratenantbackupcopy"] = "EntraIDTenantBackupCopy"

	return m
}

func makeJobTypeDisplayNames() []string {
	names := make([]string, 0, len(jobTypeCanonical))
	for _, canonical := range jobTypeCanonical {
		if canonical == "" || canonical == "Unknown" {
			continue
		}
		display := strings.TrimSpace(jobTypeDisplay(canonical))
		if display == "" {
			continue
		}
		names = append(names, display)
	}
	return names
}

func jobDescriptionWithTypes(base string) string {
	return strings.TrimSpace(base)
}

func jobLongDescription(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Job operations"
	}

	if len(jobTypeDisplayNames) == 0 {
		return base
	}

	var builder strings.Builder
	builder.WriteString(base)
	builder.WriteString("\n\nSupported job types:\n")
	for _, name := range jobTypeDisplayNames {
		builder.WriteString("  - ")
		builder.WriteString(name)
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func normalizeJobTypeKey(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}

func quoteList(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%q", values[0])
	default:
		quoted := make([]string, 0, len(values))
		for _, v := range values {
			quoted = append(quoted, fmt.Sprintf("%q", v))
		}
		return strings.Join(quoted, ", ")
	}
}

func jobTypeDisplay(raw string) string {
	if raw == "" {
		return ""
	}

	switch raw {
	case "VSphereBackup":
		return "VMware Backup"
	case "HyperVBackup":
		return "Hyper-V Backup"
	case "VSphereReplica":
		return "VMware Replication"
	case "CloudDirectorBackup":
		return "Cloud Director Backup"
	case "EntraIDTenantBackup":
		return "Microsoft Entra Tenant Backup"
	case "EntraIDAuditLogBackup":
		return "Microsoft Entra Audit Log Backup"
	case "FileBackupCopy":
		return "File Backup Copy"
	case "LegacyBackupCopy":
		return "Legacy Backup Copy"
	case "BackupCopy":
		return "Backup Copy"
	case "WindowsAgentBackup":
		return "Windows Agent Backup"
	case "LinuxAgentBackup":
		return "Linux Agent Backup"
	case "EntraIDTenantBackupCopy":
		return "Microsoft Entra Tenant Backup Copy"
	case "NasBackup":
		return "NAS Backup"
	case "NasBackupCopy":
		return "NAS Backup Copy"
	case "OracleRMANBackup":
		return "Oracle RMAN Backup"
	case "SAPHANAPluginBackup":
		return "SAP HANA Backup"
	case "AwsBackupCopy":
		return "AWS Backup Copy"
	}

	runes := []rune(raw)
	var builder strings.Builder
	for i, r := range runes {
		if i > 0 && jobShouldInsertSpace(runes[i-1], r) {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func jobStatusDisplay(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}

	clean = strings.ReplaceAll(clean, "_", " ")
	words := strings.Fields(strings.ToLower(clean))
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}

func jobShouldInsertSpace(prev, current rune) bool {
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

func normalizeSyncRestorePoints(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	switch strings.ToLower(trimmed) {
	case "all":
		return "All", nil
	case "latest":
		return "Latest", nil
	default:
		return "", fmt.Errorf("unsupported restore point sync value %q; supported values: All, Latest", trimmed)
	}
}

func quickBackupCandidatesFromJobConfig(config map[string]any) []quickBackupCandidate {
	vmSection := mapFromAnyValue(config["virtualMachines"])
	if vmSection == nil {
		return nil
	}

	includes := sliceFromAnyValue(vmSection["includes"])
	candidates := make([]quickBackupCandidate, 0, len(includes))
	for _, entry := range includes {
		include := mapFromAnyValue(entry)
		if include == nil {
			continue
		}
		candidate := quickBackupCandidateFromInclude(include)
		if candidate.Request.Name == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func quickBackupCandidateFromInclude(include map[string]any) quickBackupCandidate {
	candidate := quickBackupCandidate{
		Enabled: boolFromMapDefault(include, "isEnabled", true),
	}

	base := mapFromAnyValue(include["inventoryObject"])
	if base == nil {
		base = mapFromAnyValue(include["vmObject"])
	}
	if base == nil {
		base = include
	}

	candidate.Request.Platform = strings.TrimSpace(stringFromAnyValue(base["platform"]))
	if candidate.Request.Platform == "" {
		candidate.Request.Platform = strings.TrimSpace(stringFromAnyValue(include["platform"]))
	}
	if candidate.Request.Platform == "" {
		candidate.Request.Platform = "VSphere"
	}

	candidate.Request.Type = strings.TrimSpace(stringFromAnyValue(base["type"]))
	candidate.Request.HostName = strings.TrimSpace(stringFromAnyValue(base["hostName"]))
	candidate.Request.Name = strings.TrimSpace(stringFromAnyValue(base["name"]))
	candidate.Request.ObjectID = strings.TrimSpace(stringFromAnyValue(base["objectId"]))
	candidate.Request.URN = strings.TrimSpace(stringFromAnyValue(base["urn"]))
	candidate.Request.Size = strings.TrimSpace(stringFromAnyValue(base["size"]))

	if candidate.Request.ObjectID == "" {
		candidate.Request.ObjectID = strings.TrimSpace(stringFromAnyValue(include["objectId"]))
	}
	if candidate.Request.URN == "" {
		candidate.Request.URN = strings.TrimSpace(stringFromAnyValue(include["urn"]))
	}

	candidate.Source = strings.TrimSpace(stringFromAnyValue(base["containerName"]))
	if candidate.Source == "" {
		candidate.Source = strings.TrimSpace(stringFromAnyValue(include["containerName"]))
	}
	if candidate.Source == "" {
		candidate.Source = strings.TrimSpace(stringFromAnyValue(include["path"]))
	}

	if !strings.EqualFold(candidate.Request.Platform, "VSphere") {
		candidate.Reason = fmt.Sprintf("platform %s is not supported for quick backup", friendlyType(candidate.Request.Platform))
	} else if !strings.EqualFold(candidate.Request.Type, "VirtualMachine") {
		candidate.Reason = fmt.Sprintf("object is a %s rather than a virtual machine", friendlyType(candidate.Request.Type))
	} else if candidate.Request.Name == "" {
		candidate.Reason = "the job entry is missing the virtual machine name"
	} else if candidate.Request.ObjectID == "" {
		candidate.Reason = "the VM is missing its vSphere object ID in the job configuration"
	} else if candidate.Request.HostName == "" {
		candidate.Reason = "the VM entry is missing the vSphere host reference"
	}

	candidate.Eligible = candidate.Reason == ""
	return candidate
}

func findQuickBackupMatch(name string, candidates []quickBackupCandidate) quickBackupMatch {
	target := strings.ToLower(strings.TrimSpace(name))

	var eligible []quickBackupCandidate
	var disabled []quickBackupCandidate
	var ineligible []quickBackupCandidate

	for _, cand := range candidates {
		if strings.ToLower(strings.TrimSpace(cand.Request.Name)) != target {
			continue
		}
		if cand.Eligible && cand.Enabled {
			eligible = append(eligible, cand)
		} else if cand.Eligible && !cand.Enabled {
			disabled = append(disabled, cand)
		} else {
			ineligible = append(ineligible, cand)
		}
	}

	switch {
	case len(eligible) == 0 && len(disabled) == 0 && len(ineligible) == 0:
		return quickBackupMatch{kind: quickBackupMatchNone}
	case len(eligible) == 1:
		return quickBackupMatch{kind: quickBackupMatchSingle, candidates: eligible}
	case len(eligible) > 1:
		return quickBackupMatch{kind: quickBackupMatchMultiple, candidates: eligible}
	case len(disabled) > 0:
		return quickBackupMatch{kind: quickBackupMatchDisabled, candidates: disabled}
	default:
		return quickBackupMatch{kind: quickBackupMatchIneligible, candidates: ineligible}
	}
}

func resolveCandidateFromMatch(cmd *cobra.Command, jobName, vmName string, match quickBackupMatch, assumeYes bool) (quickBackupCandidate, bool, error) {
	switch match.kind {
	case quickBackupMatchNone:
		return quickBackupCandidate{}, false, fmt.Errorf("job %q does not include an enabled VMware virtual machine named %q", jobName, vmName)
	case quickBackupMatchDisabled:
		details := joinCandidateDetails(match.candidates)
		if details != "" {
			return quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q exists in job %q but is disabled (%s); enable it in the job configuration and retry", vmName, jobName, details)
		}
		return quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q exists in job %q but is disabled; enable it in the job configuration and retry", vmName, jobName)
	case quickBackupMatchIneligible:
		reason := ""
		if len(match.candidates) > 0 {
			reason = match.candidates[0].Reason
		}
		if reason == "" {
			reason = "the job does not include enough information to perform a quick backup"
		}
		details := joinCandidateDetails(match.candidates)
		if details != "" {
			reason = fmt.Sprintf("%s (%s)", reason, details)
		}
		return quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q is present in job %q but cannot be used for quick backup: %s", vmName, jobName, reason)
	case quickBackupMatchSingle:
		candidate := match.candidates[0]
		if !candidate.Enabled {
			return quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q exists in job %q but is disabled; enable it in the job configuration and retry", candidate.Request.Name, jobName)
		}
		return candidate, false, nil
	case quickBackupMatchMultiple:
		if assumeYes {
			return quickBackupCandidate{}, false, fmt.Errorf("multiple virtual machines named %q found in job %q; rerun without --yes to choose one", vmName, jobName)
		}
		selected, cancelled, err := promptForQuickBackupSelection(cmd, jobName, vmName, match.candidates)
		if err != nil {
			return quickBackupCandidate{}, false, err
		}
		if cancelled {
			return quickBackupCandidate{}, true, nil
		}
		if !selected.Enabled {
			return quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q exists in job %q but is disabled; enable it in the job configuration and retry", selected.Request.Name, jobName)
		}
		return selected, false, nil
	default:
		return quickBackupCandidate{}, false, fmt.Errorf("job %q does not include an enabled VMware virtual machine named %q", jobName, vmName)
	}
}

func promptForQuickBackupSelection(cmd *cobra.Command, jobName, vmName string, candidates []quickBackupCandidate) (quickBackupCandidate, bool, error) {
	reader := bufio.NewReader(cmd.InOrStdin())

	fmt.Fprintf(cmd.OutOrStdout(), "Multiple virtual machines named %q are configured in job %q:\n", vmName, jobName)
	for i, cand := range candidates {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d) %s\n", i+1, quickBackupCandidateSummary(cand))
	}

	for {
		fmt.Fprintf(cmd.OutOrStdout(), "Select a virtual machine [1-%d] or press Enter to cancel: ", len(candidates))
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return quickBackupCandidate{}, false, fmt.Errorf("read selection: %w", err)
		}
		choice := strings.TrimSpace(input)
		if choice == "" || strings.EqualFold(choice, "c") || strings.EqualFold(choice, "cancel") || strings.EqualFold(choice, "q") || strings.EqualFold(choice, "quit") {
			return quickBackupCandidate{}, true, nil
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 1 || index > len(candidates) {
			fmt.Fprintln(cmd.OutOrStdout(), "Please enter a number from the list or press Enter to cancel.")
			if errors.Is(err, io.EOF) {
				return quickBackupCandidate{}, true, nil
			}
			continue
		}
		return candidates[index-1], false, nil
	}
}

func joinCandidateDetails(candidates []quickBackupCandidate) string {
	if len(candidates) == 0 {
		return ""
	}
	parts := make([]string, 0, len(candidates))
	for _, cand := range candidates {
		summary := quickBackupCandidateSummary(cand)
		if !cand.Enabled {
			summary = fmt.Sprintf("%s (disabled)", summary)
		}
		if !cand.Eligible && cand.Reason != "" {
			summary = fmt.Sprintf("%s (%s)", summary, cand.Reason)
		}
		parts = append(parts, summary)
	}
	return strings.Join(parts, "; ")
}

func quickBackupCandidateSummary(c quickBackupCandidate) string {
	segments := []string{}
	if c.Request.HostName != "" {
		segments = append(segments, fmt.Sprintf("host %s", c.Request.HostName))
	}
	if c.Request.ObjectID != "" {
		segments = append(segments, fmt.Sprintf("ID %s", c.Request.ObjectID))
	}
	if c.Request.URN != "" {
		segments = append(segments, fmt.Sprintf("URN %s", c.Request.URN))
	}
	if c.Source != "" {
		segments = append(segments, fmt.Sprintf("path %s", c.Source))
	}
	if len(segments) == 0 {
		return "no additional metadata"
	}
	return strings.Join(segments, ", ")
}

func promptForQuickBackupJob(cmd *cobra.Command, vmName string, options []quickBackupJobOption) (quickBackupJobOption, bool, error) {
	reader := bufio.NewReader(cmd.InOrStdin())

	fmt.Fprintf(cmd.OutOrStdout(), "Virtual machine %q is included in multiple jobs:\n", vmName)
	for i, opt := range options {
		summary := "no additional metadata"
		if len(opt.Match.candidates) > 0 {
			summary = quickBackupCandidateSummary(opt.Match.candidates[0])
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %d) %s (%s)\n", i+1, opt.JobName, summary)
	}

	for {
		fmt.Fprintf(cmd.OutOrStdout(), "Select a job [1-%d] or press Enter to cancel: ", len(options))
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return quickBackupJobOption{}, false, fmt.Errorf("read selection: %w", err)
		}
		choice := strings.TrimSpace(input)
		if choice == "" || strings.EqualFold(choice, "c") || strings.EqualFold(choice, "cancel") || strings.EqualFold(choice, "q") || strings.EqualFold(choice, "quit") {
			return quickBackupJobOption{}, true, nil
		}
		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(options) {
			fmt.Fprintln(cmd.OutOrStdout(), "Please enter a number from the list or press Enter to cancel.")
			if errors.Is(err, io.EOF) {
				return quickBackupJobOption{}, true, nil
			}
			continue
		}
		return options[idx-1], false, nil
	}
}

func formatQuickBackupFailures(failures []quickBackupJobFailure) string {
	if len(failures) == 0 {
		return ""
	}
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		if strings.TrimSpace(failure.Reason) == "" {
			parts = append(parts, failure.JobName)
		} else {
			parts = append(parts, fmt.Sprintf("%s (%s)", failure.JobName, failure.Reason))
		}
	}
	return strings.Join(parts, "; ")
}

func autoResolveQuickBackupJob(ctx context.Context, cmd *cobra.Command, httpClient *client.Client, vmName string, assumeYes bool) (string, string, *client.JobState, map[string]any, quickBackupCandidate, bool, error) {
	states, err := httpClient.JobStates(ctx, client.JobStatesFilter{})
	if err != nil {
		return "", "", nil, nil, quickBackupCandidate{}, false, err
	}

	var eligible []quickBackupJobOption
	var failures []quickBackupJobFailure

	for _, state := range states {
		if !strings.EqualFold(state.Type, "VSphereBackup") {
			continue
		}

		config, err := httpClient.Job(ctx, state.ID)
		if err != nil {
			failures = append(failures, quickBackupJobFailure{
				JobName: state.Name,
				Reason:  fmt.Sprintf("fetch configuration: %v", err),
			})
			continue
		}

		candidates := quickBackupCandidatesFromJobConfig(config)
		match := findQuickBackupMatch(vmName, candidates)
		if match.kind == quickBackupMatchNone {
			continue
		}

		stateCopy := state
		if jobStateIndicatesActive(&stateCopy) {
			failures = append(failures, quickBackupJobFailure{
				JobName: state.Name,
				Reason:  "job is currently running",
			})
			continue
		}

		if extractBool(config, "isDisabled") {
			failures = append(failures, quickBackupJobFailure{
				JobName: state.Name,
				Reason:  "job is disabled",
			})
			continue
		}

		switch match.kind {
		case quickBackupMatchSingle, quickBackupMatchMultiple:
			eligible = append(eligible, quickBackupJobOption{
				JobID:   state.ID,
				JobName: state.Name,
				State:   &stateCopy,
				Config:  config,
				Match:   match,
			})
		case quickBackupMatchDisabled:
			details := joinCandidateDetails(match.candidates)
			if details == "" {
				details = "virtual machine entry is disabled"
			}
			failures = append(failures, quickBackupJobFailure{
				JobName: state.Name,
				Reason:  details,
			})
		case quickBackupMatchIneligible:
			reason := ""
			if len(match.candidates) > 0 {
				reason = match.candidates[0].Reason
			}
			if reason == "" {
				reason = "virtual machine entry is missing required metadata"
			}
			failures = append(failures, quickBackupJobFailure{
				JobName: state.Name,
				Reason:  reason,
			})
		}
	}

	if len(eligible) == 0 {
		if len(failures) > 0 {
			return "", "", nil, nil, quickBackupCandidate{}, false, fmt.Errorf("virtual machine %q was located but cannot be protected because %s", vmName, formatQuickBackupFailures(failures))
		}
		return "", "", nil, nil, quickBackupCandidate{}, false, fmt.Errorf("no job containing virtual machine %q was found", vmName)
	}

	var option quickBackupJobOption
	if len(eligible) == 1 {
		option = eligible[0]
	} else {
		if assumeYes {
			names := make([]string, 0, len(eligible))
			for _, opt := range eligible {
				names = append(names, fmt.Sprintf("%q", opt.JobName))
			}
			return "", "", nil, nil, quickBackupCandidate{}, false, fmt.Errorf("multiple jobs contain virtual machine %q (%s); specify --name/--id or rerun without --yes to choose one", vmName, strings.Join(names, ", "))
		}
		selected, cancelled, err := promptForQuickBackupJob(cmd, vmName, eligible)
		if err != nil {
			return "", "", nil, nil, quickBackupCandidate{}, false, err
		}
		if cancelled {
			return "", "", nil, nil, quickBackupCandidate{}, true, nil
		}
		option = selected
	}

	candidate, cancelled, err := resolveCandidateFromMatch(cmd, option.JobName, vmName, option.Match, assumeYes)
	if err != nil {
		return "", "", nil, nil, quickBackupCandidate{}, false, err
	}
	if cancelled {
		return option.JobID, option.JobName, option.State, option.Config, quickBackupCandidate{}, true, nil
	}

	return option.JobID, option.JobName, option.State, option.Config, candidate, false, nil
}

func mapFromAnyValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func sliceFromAnyValue(value any) []any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for i, item := range typed {
			result[i] = item
		}
		return result
	default:
		return nil
	}
}

func stringFromAnyValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case float32:
		return strings.TrimSpace(strconv.FormatFloat(float64(v), 'f', -1, 64))
	case int:
		return strings.TrimSpace(strconv.Itoa(v))
	case int64:
		return strings.TrimSpace(strconv.FormatInt(v, 10))
	case uint64:
		return strings.TrimSpace(strconv.FormatUint(v, 10))
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func boolFromMapDefault(m map[string]any, key string, fallback bool) bool {
	if m == nil {
		return fallback
	}

	value, ok := m[key]
	if !ok || value == nil {
		return fallback
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		clean := strings.ToLower(strings.TrimSpace(typed))
		switch clean {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n":
			return false
		default:
			return fallback
		}
	case float64:
		return typed != 0
	case float32:
		return typed != 0
	case int:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case uint:
		return typed != 0
	case uint32:
		return typed != 0
	case uint64:
		return typed != 0
	default:
		return fallback
	}
}

func friendlyType(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "unknown"
	}
	clean = strings.ReplaceAll(clean, "_", " ")
	switch {
	case strings.EqualFold(clean, "VirtualMachine"):
		return "virtual machine"
	case strings.EqualFold(clean, "VSphere"):
		return "VMware vSphere"
	case strings.EqualFold(clean, "CloudDirector"):
		return "VMware Cloud Director"
	default:
		return strings.ToLower(clean)
	}
}

func formatSessionTail(session *client.Session) string {
	if session == nil {
		return ""
	}
	sessionID := strings.TrimSpace(session.ID)
	if sessionID == "" {
		return ""
	}
	state := strings.TrimSpace(jobStatusDisplay(session.State))
	if state != "" {
		return fmt.Sprintf(" Session %s (%s).", sessionID, state)
	}
	return fmt.Sprintf(" Session %s.", sessionID)
}

func printSessionMessage(cmd *cobra.Command, session *client.Session, message string) error {
	format := outputFormat()
	if format == "json" {
		if session != nil {
			return output.Print(format, session)
		}
		return output.Print(format, map[string]any{"message": message})
	}
	fmt.Fprintln(cmd.OutOrStdout(), message)
	return nil
}
