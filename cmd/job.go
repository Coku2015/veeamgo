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

func jobCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdJobUse,
		Short: "Job inventory and details",
	}
	root.AddCommand(jobGetCmd())
	root.AddCommand(jobDescribeCmd())
	return root
}

type jobFilterOptions struct {
	name                  string
	types                 []string
	vsphereBackup         bool
	hyperVBackup          bool
	vsphereReplica        bool
	cloudDirectorBackup   bool
	entraTenantBackup     bool
	entraAuditLogBackup   bool
	fileBackupCopy        bool
	legacyBackupCopy      bool
	backupCopy            bool
	windowsAgentBackup    bool
	linuxAgentBackup      bool
	entraTenantBackupCopy bool
	status                string
	result                string
	workload              string
	repository            string
	highPriority          bool
	since                 string
	before                string
	afterJob              string
	limit                 int
}

func (opts jobFilterOptions) selectedTypes() []string {
	types := append([]string{}, opts.types...)
	if opts.vsphereBackup {
		types = append(types, "VSphereBackup")
	}
	if opts.hyperVBackup {
		types = append(types, "HyperVBackup")
	}
	if opts.vsphereReplica {
		types = append(types, "VSphereReplica")
	}
	if opts.cloudDirectorBackup {
		types = append(types, "CloudDirectorBackup")
	}
	if opts.entraTenantBackup {
		types = append(types, "EntraIDTenantBackup")
	}
	if opts.entraAuditLogBackup {
		types = append(types, "EntraIDAuditLogBackup")
	}
	if opts.fileBackupCopy {
		types = append(types, "FileBackupCopy")
	}
	if opts.legacyBackupCopy {
		types = append(types, "LegacyBackupCopy")
	}
	if opts.backupCopy {
		types = append(types, "BackupCopy")
	}
	if opts.windowsAgentBackup {
		types = append(types, "WindowsAgentBackup")
	}
	if opts.linuxAgentBackup {
		types = append(types, "LinuxAgentBackup")
	}
	if opts.entraTenantBackupCopy {
		types = append(types, "EntraIDTenantBackupCopy")
	}

	return uniqueStrings(types)
}

func jobGetCmd() *cobra.Command {
	opts := jobFilterOptions{}

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List jobs with current state",
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
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "Filter by job type (repeatable)")
	cmd.Flags().BoolVar(&opts.vsphereBackup, "vsphere-backup", false, "Only include VMware vSphere backup jobs")
	cmd.Flags().BoolVar(&opts.hyperVBackup, "hyperv-backup", false, "Only include Hyper-V backup jobs")
	cmd.Flags().BoolVar(&opts.vsphereReplica, "vsphere-replica", false, "Only include VMware vSphere replica jobs")
	cmd.Flags().BoolVar(&opts.cloudDirectorBackup, "cloud-director-backup", false, "Only include VMware Cloud Director backup jobs")
	cmd.Flags().BoolVar(&opts.entraTenantBackup, "entra-tenant-backup", false, "Only include Microsoft Entra ID tenant backup jobs")
	cmd.Flags().BoolVar(&opts.entraAuditLogBackup, "entra-auditlog-backup", false, "Only include Microsoft Entra ID audit log backup jobs")
	cmd.Flags().BoolVar(&opts.fileBackupCopy, "file-backup-copy", false, "Only include file backup copy jobs")
	cmd.Flags().BoolVar(&opts.legacyBackupCopy, "legacy-backup-copy", false, "Only include legacy backup copy jobs")
	cmd.Flags().BoolVar(&opts.backupCopy, "backup-copy", false, "Only include backup copy jobs")
	cmd.Flags().BoolVar(&opts.windowsAgentBackup, "windows-agent-backup", false, "Only include Windows agent backup jobs")
	cmd.Flags().BoolVar(&opts.linuxAgentBackup, "linux-agent-backup", false, "Only include Linux agent backup jobs")
	cmd.Flags().BoolVar(&opts.entraTenantBackupCopy, "entra-tenant-backup-copy", false, "Only include Microsoft Entra ID tenant backup copy jobs")
	cmd.Flags().StringVar(&opts.status, "status", "", "Filter by current job status (e.g. Running, Stopped)")
	cmd.Flags().StringVar(&opts.result, "result", "", "Filter by last result (e.g. Success, Warning, Failed)")
	cmd.Flags().StringVar(&opts.workload, "workload", "", "Filter by workload (e.g. Vmware, HyperV)")
	cmd.Flags().StringVar(&opts.repository, "repository", "", "Filter by target repository name")
	cmd.Flags().BoolVar(&opts.highPriority, "high-priority", false, "Only include jobs marked high priority")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only include jobs with last run ≥ RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.before, "before", "", "Only include jobs with last run ≤ RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.afterJob, "after-job", "", "Only include jobs chained after the specified job name")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of jobs to return (default: all)")

	return cmd
}

func jobDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <name>", cmdDescribeUse),
		Short: "Show detailed job configuration and state",
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

func buildJobStatesFilter(ctx context.Context, httpClient *client.Client, opts jobFilterOptions) (client.JobStatesFilter, error) {
	filter := client.JobStatesFilter{
		Name:       opts.name,
		Types:      opts.selectedTypes(),
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

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
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
