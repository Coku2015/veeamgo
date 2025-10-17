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

func replicaCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdReplicaUse,
		Short: "Replica restore point inventory",
	}
	root.AddCommand(replicaGetCmd())
	return root
}

type replicaGetOptions struct {
	jobName          string
	replicaName      string
	replicaPointName string
	platformName     string
	platformID       string
	malwareStatus    string
	createdAfter     string
	createdBefore    string
	sort             string
	desc             bool
	limit            int
}

func replicaGetCmd() *cobra.Command {
	opts := replicaGetOptions{}

	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "List replica restore points for a replication job",
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

			replica, err := selectReplica(ctx, httpClient, job, opts.replicaName)
			if err != nil {
				return err
			}

			filter := client.ReplicaPointsFilter{
				Name:          opts.replicaPointName,
				ReplicaID:     replica.ID,
				PlatformName:  opts.platformName,
				PlatformID:    opts.platformID,
				MalwareStatus: opts.malwareStatus,
				MaxItems:      opts.limit,
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

			result, err := httpClient.ReplicaPoints(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]replicaPointRow, 0, len(result.ReplicaPoints))
			for _, point := range result.ReplicaPoints {
				rows = append(rows, replicaPointRow{
					VMName:   point.Name,
					State:    point.State,
					Platform: point.PlatformName,
					Created:  formatTimestampValue(point.CreationTime),
					JobName:  job.Name,
					Malware:  point.MalwareStatus,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.jobName, "name", "", "Replication job name (required)")
	cmd.Flags().StringVar(&opts.replicaName, "replica", "", "Replica name (optional if job has a single replica)")
	cmd.Flags().StringVar(&opts.replicaPointName, "replica-point", "", "Filter by replica restore point name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.platformName, "platform", "", "Filter by platform name (e.g. VMware)")
	cmd.Flags().StringVar(&opts.platformID, "platform-id", "", "Filter by platform ID")
	cmd.Flags().StringVar(&opts.malwareStatus, "malware", "", "Filter by malware status")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Include restore points created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Include restore points created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by column (e.g. creationTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of restore points to return")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func selectReplica(ctx context.Context, httpClient *client.Client, job *client.JobState, replicaName string) (*client.Replica, error) {
	if replicaName != "" {
		return resolveReplicaByName(ctx, httpClient, replicaName, job.ID)
	}

	replicas, err := httpClient.Replicas(ctx, client.ReplicasFilter{
		JobID:    job.ID,
		MaxItems: 2,
	})
	if err != nil {
		return nil, err
	}
	if len(replicas) == 0 {
		return nil, fmt.Errorf("no replicas found for job %q", job.Name)
	}
	if len(replicas) > 1 {
		names := make([]string, 0, len(replicas))
		for _, rp := range replicas {
			names = append(names, rp.Name)
			if len(names) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple replicas found for job %q; specify --replica to disambiguate (found: %s)", job.Name, strings.Join(names, ", "))
	}
	return &replicas[0], nil
}

func resolveReplicaByName(ctx context.Context, httpClient *client.Client, name, jobID string) (*client.Replica, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("replica name cannot be empty")
	}

	replicas, err := httpClient.Replicas(ctx, client.ReplicasFilter{
		Name:  clean,
		JobID: jobID,
	})
	if err != nil {
		return nil, err
	}

	exact := make([]client.Replica, 0)
	for _, rp := range replicas {
		if strings.EqualFold(rp.Name, clean) {
			exact = append(exact, rp)
		}
	}

	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		if len(replicas) == 0 {
			return nil, fmt.Errorf("replica named %q not found", clean)
		}
		suggestions := make([]string, 0, len(replicas))
		for _, rp := range replicas {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", rp.Name, rp.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("replica named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		ids := make([]string, 0, len(exact))
		for _, rp := range exact {
			ids = append(ids, rp.ID)
			if len(ids) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple replicas named %q found (ids: %s)", clean, strings.Join(ids, ", "))
	}
}

type replicaPointRow struct {
	VMName   string `json:"VM Name"`
	State    string `json:"State"`
	Platform string `json:"Platform"`
	Created  string `json:"Created"`
	JobName  string `json:"Job Name"`
	Malware  string `json:"Malware Status"`
}
