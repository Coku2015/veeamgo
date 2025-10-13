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

func backupFilesCmd() *cobra.Command {
	opts := backupFilesOptions{}

	cmd := &cobra.Command{
		Use:   "files",
		Short: "List files contained in a backup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			if strings.TrimSpace(opts.jobName) == "" {
				return fmt.Errorf("--name is required")
			}

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			job, err := httpClient.JobStateByName(ctx, opts.jobName)
			if err != nil {
				return fmt.Errorf("resolve job %q: %w", opts.jobName, err)
			}

			backup, err := selectBackupForJob(ctx, httpClient, job, opts.backupName)
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

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Backup job name (required)")
	cmd.Flags().StringVar(&opts.jobName, "job", "", "Deprecated: use --name to specify the backup job name")
	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Backup name (optional if job has a single backup)")
	cmd.Flags().StringVar(&opts.fileName, "file", "", "Filter by backup file name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Only include files created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Only include files created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.gfsPeriod, "gfs", "", "Filter by GFS period (e.g. Weekly, Monthly)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. creationTime, backupSize)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of files to return")

	if jobFlag := cmd.Flags().Lookup("job"); jobFlag != nil {
		jobFlag.Hidden = true
	}
	_ = cmd.MarkFlagRequired("name")

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

			if strings.TrimSpace(opts.jobName) == "" {
				return fmt.Errorf("--name is required")
			}

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			job, err := httpClient.JobStateByName(ctx, opts.jobName)
			if err != nil {
				return fmt.Errorf("resolve job %q: %w", opts.jobName, err)
			}

			backup, err := selectBackupForJob(ctx, httpClient, job, opts.backupName)
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
				rows = append(rows, backupObjectRow{
					Name:          obj.Name,
					Type:          obj.Type,
					Platform:      obj.PlatformName,
					RestorePoints: obj.RestorePointsCount,
					LastRunFailed: yesNo(obj.LastRunFailed),
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Backup job name (required)")
	cmd.Flags().StringVar(&opts.jobName, "job", "", "Deprecated: use --name to specify the backup job name")
	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Backup name (optional if job has a single backup)")

	if jobFlag := cmd.Flags().Lookup("job"); jobFlag != nil {
		jobFlag.Hidden = true
	}
	_ = cmd.MarkFlagRequired("name")

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
	LastRunFailed string `json:"Last Run Failed"`
	ID            string `json:"ID"`
}

func selectBackupForJob(ctx context.Context, httpClient *client.Client, job *client.JobState, backupName string) (*client.Backup, error) {
	if backupName != "" {
		backup, err := httpClient.BackupByName(ctx, backupName)
		if err != nil {
			return nil, err
		}
		if backup.JobID != job.ID {
			return nil, fmt.Errorf("backup %q does not belong to job %q", backup.Name, job.Name)
		}
		return backup, nil
	}

	backups, err := httpClient.Backups(ctx, client.BackupsFilter{
		JobID:    job.ID,
		MaxItems: 2,
	})
	if err != nil {
		return nil, err
	}
	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found for job %q", job.Name)
	}
	if len(backups) > 1 {
		names := make([]string, 0, len(backups))
		for _, b := range backups {
			names = append(names, b.Name)
			if len(names) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple backups found for job %q; specify --backup to disambiguate (found: %s)", job.Name, strings.Join(names, ", "))
	}
	return &backups[0], nil
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
