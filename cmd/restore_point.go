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

func restorePointCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdRestorePointUse,
		Short: "Restore point inventory",
	}
	root.AddCommand(restorePointGetCmd())
	return root
}

type restorePointGetOptions struct {
	jobName          string
	backupName       string
	restorePointName string
	objectID         string
	platformName     string
	platformID       string
	malwareStatus    string
	createdAfter     string
	createdBefore    string
	sort             string
	desc             bool
	limit            int
}

func restorePointGetCmd() *cobra.Command {
	opts := restorePointGetOptions{}

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List restore points for a backup job",
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

			var backup *client.Backup
			if opts.backupName != "" {
				backup, err = httpClient.BackupByName(ctx, opts.backupName)
				if err != nil {
					return err
				}
				if backup.JobID != job.ID {
					return fmt.Errorf("backup %q does not belong to job %q", backup.Name, job.Name)
				}
			} else {
				backups, err := httpClient.Backups(ctx, client.BackupsFilter{
					JobID:    job.ID,
					MaxItems: 2,
				})
				if err != nil {
					return err
				}
				if len(backups) == 0 {
					return fmt.Errorf("no backups found for job %q", job.Name)
				}
				if len(backups) > 1 {
					names := make([]string, 0, len(backups))
					for _, b := range backups {
						names = append(names, b.Name)
						if len(names) == 5 {
							break
						}
					}
					return fmt.Errorf("multiple backups found for job %q; specify --backup to disambiguate (found: %s)", job.Name, strings.Join(names, ", "))
				}
				backup = &backups[0]
			}

			filter := client.RestorePointsFilter{
				Name:           opts.restorePointName,
				BackupID:       backup.ID,
				BackupObjectID: opts.objectID,
				PlatformName:   opts.platformName,
				PlatformID:     opts.platformID,
				MalwareStatus:  opts.malwareStatus,
				MaxItems:       opts.limit,
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

			result, err := httpClient.RestorePoints(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]restorePointRow, 0, len(result.RestorePoints))
			for _, rp := range result.RestorePoints {
				rows = append(rows, restorePointRow{
					Name:      rp.Name,
					Type:      rp.Type,
					Platform:  rp.PlatformName,
					CreatedAt: formatTimestampValue(rp.CreationTime),
					JobName:   job.Name,
					Malware:   rp.MalwareStatus,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Backup job name (required)")
	cmd.Flags().StringVar(&opts.jobName, "job", "", "Deprecated: use --name to specify the backup job name")
	cmd.Flags().StringVar(&opts.backupName, "backup", "", "Backup name (optional if job has a single backup)")
	cmd.Flags().StringVar(&opts.restorePointName, "restorepoint", "", "Filter by restore point name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.restorePointName, "restorepoint-name", "", "Filter by restore point name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.objectID, "object", "", "Filter by backup object ID")
	cmd.Flags().StringVar(&opts.platformName, "platform", "", "Filter by platform name (e.g. VMware)")
	cmd.Flags().StringVar(&opts.platformID, "platform-id", "", "Filter by platform ID")
	cmd.Flags().StringVar(&opts.malwareStatus, "malware", "", "Filter by malware status")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Include restore points created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Include restore points created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. creationTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of restore points to return")

	if jobFlag := cmd.Flags().Lookup("job"); jobFlag != nil {
		jobFlag.Hidden = true
	}

	return cmd
}

type restorePointRow struct {
	Name      string `json:"Name"`
	Type      string `json:"Type"`
	Platform  string `json:"Platform"`
	CreatedAt string `json:"Created At"`
	JobName   string `json:"Job Name"`
	Malware   string `json:"Malware Status"`
}
