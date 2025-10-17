package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func backupCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdBackupUse,
		Short: "Backup inspection helpers",
	}
	root.AddCommand(backupListCmd())
	root.AddCommand(backupFilesCmd())
	root.AddCommand(backupObjectsCmd())
	return root
}

type backupFilesOptions struct {
	jobName       string
	backupName    string
	fileName      string
	createdAfter  string
	createdBefore string
	gfsPeriod     string
	sort          string
	desc          bool
	limit         int
}

type backupObjectsOptions struct {
	jobName    string
	backupName string
}

type backupListOptions struct {
	backupName    string
	jobName       string
	jobID         string
	jobType       string
	createdAfter  string
	createdBefore string
	sort          string
	desc          bool
	limit         int
}

func backupListCmd() *cobra.Command {
	opts := backupListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.BackupsFilter{
				Name:     strings.TrimSpace(opts.backupName),
				JobID:    strings.TrimSpace(opts.jobID),
				JobType:  strings.TrimSpace(opts.jobType),
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

			var jobLabel string
			if strings.TrimSpace(opts.jobName) != "" {
				job, err := httpClient.JobStateByName(ctx, opts.jobName)
				if err != nil {
					return fmt.Errorf("resolve job %q: %w", opts.jobName, err)
				}
				if filter.JobID != "" && !strings.EqualFold(filter.JobID, job.ID) {
					return fmt.Errorf("--job-id (%s) does not match job %q (%s)", filter.JobID, job.Name, job.ID)
				}
				filter.JobID = job.ID
				jobLabel = job.Name
			}

			backups, err := httpClient.Backups(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, backups)
			}

			rows := make([]backupListRow, 0, len(backups))
			for _, backup := range backups {
				label := jobLabel
				if label == "" {
					label = backup.JobID
					if isZeroUUID(backup.JobID) {
						label = "(deleted job)"
					}
				}
				rows = append(rows, backupListRow{
					Name:       backup.Name,
					BackupID:   backup.ID,
					Job:        label,
					JobType:    backup.JobType,
					Repository: backup.RepositoryName,
					CreatedAt:  formatTimestampValue(backup.CreationTime),
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Filter by backup name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.jobName, "name", "", "Filter by backup job name")
	cmd.Flags().StringVar(&opts.jobID, "job-id", "", "Filter by backup job ID")
	cmd.Flags().StringVar(&opts.jobType, "job-type", "", "Filter by backup job type (e.g. VSphereBackup)")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Include backups created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Include backups created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. creationTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of backups to return")

	return cmd
}

func backupFilesCmd() *cobra.Command {
	opts := backupFilesOptions{}

	cmd := &cobra.Command{
		Use:   "files",
		Short: "List files contained in a backup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			if strings.TrimSpace(opts.jobName) == "" && strings.TrimSpace(opts.backupName) == "" {
				return fmt.Errorf("provide --backup or --name")
			}

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			backup, _, err := resolveBackup(ctx, httpClient, opts.jobName, opts.backupName)
			if err != nil {
				return err
			}

			filter := client.BackupFilesFilter{
				Name:      opts.fileName,
				GFSPeroid: opts.gfsPeriod,
				MaxItems:  opts.limit,
			}
			if opts.sort != "" {
				filter.OrderColumn = opts.sort
				if opts.desc {
					value := false
					filter.OrderAsc = &value
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

			files, err := httpClient.BackupFiles(ctx, backup.ID, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, files)
			}

			rows := make([]backupFileRow, 0, len(files))
			for _, file := range files {
				rows = append(rows, backupFileRow{
					Name:          file.Name,
					BackupSize:    formatBytes(file.BackupSize),
					DataSize:      formatBytes(file.DataSize),
					DedupRatio:    strconv.Itoa(file.DedupRatio),
					CompressRatio: strconv.Itoa(file.CompressRatio),
					CreatedAt:     formatTimestampValue(file.CreationTime),
					GFS:           formatGFSPeroids(file.GFSPeroids),
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Backup job name (optional)")
	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Backup name")
	cmd.Flags().StringVar(&opts.fileName, "file", "", "Filter by backup file name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Only include files created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Only include files created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.gfsPeriod, "gfs", "", "Filter by GFS period (e.g. Weekly, Monthly)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. creationTime, backupSize)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of files to return")

	return cmd
}

func backupObjectsCmd() *cobra.Command {
	opts := backupObjectsOptions{}

	cmd := &cobra.Command{
		Use:   "objects",
		Short: "List objects protected by a backup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			if strings.TrimSpace(opts.jobName) == "" && strings.TrimSpace(opts.backupName) == "" {
				return fmt.Errorf("provide --backup or --name")
			}

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			backup, _, err := resolveBackup(ctx, httpClient, opts.jobName, opts.backupName)
			if err != nil {
				return err
			}

			objects, err := httpClient.BackupObjects(ctx, backup.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, objects)
			}

			rows := make([]backupObjectRow, 0, len(objects))
			for _, obj := range objects {
				lastRunResult := "Success"
				if obj.LastRunFailed {
					lastRunResult = "Failed"
				}

				rows = append(rows, backupObjectRow{
					Name:          obj.Name,
					Type:          obj.Type,
					Platform:      obj.PlatformName,
					RestorePoints: obj.RestorePointsCount,
					LastRunResult: lastRunResult,
					ID:            obj.ID,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Backup job name (optional)")
	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Backup name")

	return cmd
}

type backupFileRow struct {
	Name          string `json:"Name"`
	BackupSize    string `json:"Backup Size"`
	DataSize      string `json:"Data Size"`
	DedupRatio    string `json:"Dedup Ratio"`
	CompressRatio string `json:"Compress Ratio"`
	CreatedAt     string `json:"Created"`
	GFS           string `json:"GFS Periods"`
}

type backupObjectRow struct {
	Name          string `json:"Name"`
	Type          string `json:"Type"`
	Platform      string `json:"Platform"`
	RestorePoints int    `json:"Restore Points"`
	LastRunResult string `json:"Last Run Result"`
	ID            string `json:"ID"`
}

type backupListRow struct {
	Name       string `json:"Name"`
	BackupID   string `json:"Backup ID"`
	Job        string `json:"Job"`
	JobType    string `json:"Job Type"`
	Repository string `json:"Repository"`
	CreatedAt  string `json:"Created"`
}

func resolveBackup(ctx context.Context, httpClient *client.Client, jobName, backupName string) (*client.Backup, *client.JobState, error) {
	jobName = strings.TrimSpace(jobName)
	backupName = strings.TrimSpace(backupName)

	var (
		backup *client.Backup
		job    *client.JobState
		err    error
	)

	if backupName != "" {
		backup, err = httpClient.BackupByName(ctx, backupName)
		if err != nil {
			return nil, nil, err
		}

		if !isZeroUUID(backup.JobID) {
			states, err := httpClient.JobStates(ctx, client.JobStatesFilter{
				ID:       backup.JobID,
				MaxItems: 1,
			})
			if err == nil && len(states) > 0 {
				job = &states[0]
			}
		}

		if jobName != "" {
			if job == nil {
				if candidate, err := httpClient.JobStateByName(ctx, jobName); err == nil {
					if !isZeroUUID(backup.JobID) && !strings.EqualFold(candidate.ID, backup.JobID) {
						return nil, nil, fmt.Errorf("backup %q belongs to job %q", backup.Name, candidate.Name)
					}
					job = candidate
				}
			} else if !strings.EqualFold(job.Name, jobName) {
				return nil, nil, fmt.Errorf("backup %q belongs to job %q (not %q)", backup.Name, job.Name, jobName)
			}
		}

		return backup, job, nil
	}

	if jobName == "" {
		return nil, nil, fmt.Errorf("provide --backup or --name")
	}

	job, err = httpClient.JobStateByName(ctx, jobName)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve job %q: %w", jobName, err)
	}

	backups, err := httpClient.Backups(ctx, client.BackupsFilter{
		JobID:    job.ID,
		MaxItems: 2,
	})
	if err != nil {
		return nil, nil, err
	}
	if len(backups) == 0 {
		return nil, nil, fmt.Errorf("no backups found for job %q", job.Name)
	}
	if len(backups) > 1 {
		names := make([]string, 0, len(backups))
		for _, b := range backups {
			names = append(names, b.Name)
			if len(names) == 5 {
				break
			}
		}
		return nil, nil, fmt.Errorf("multiple backups found for job %q; specify --backup to disambiguate (found: %s)", job.Name, strings.Join(names, ", "))
	}
	backup = &backups[0]

	return backup, job, nil
}

func isZeroUUID(value string) bool {
	clean := strings.TrimSpace(value)
	return clean == "" || strings.EqualFold(clean, "00000000-0000-0000-0000-000000000000")
}
func formatGFSPeroids(periods []string) string {
	if len(periods) == 0 {
		return ""
	}

	cleaned := make([]string, 0, len(periods))
	for _, period := range periods {
		trimmed := strings.TrimSpace(period)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "none") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}

	if len(cleaned) == 0 {
		return ""
	}

	return strings.Join(cleaned, ", ")
}

func objectsInBackupCmd() *cobra.Command {
	cmd := backupObjectsCmd()
	cmd.Use = "objectsinbackup"
	cmd.Short = "List objects contained in a backup"
	return cmd
}
