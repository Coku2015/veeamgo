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

type taskListOptions struct {
	name          string
	sessionID     string
	taskType      string
	sessionType   string
	state         string
	result        string
	scanType      string
	scanResult    string
	scanState     string
	createdAfter  string
	createdBefore string
	endedAfter    string
	endedBefore   string
	sort          string
	desc          bool
	limit         int
}

func taskListCmd() *cobra.Command {
	opts := taskListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List task sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.TaskSessionsFilter{
				Name:        opts.name,
				SessionID:   opts.sessionID,
				Type:        opts.taskType,
				SessionType: opts.sessionType,
				State:       opts.state,
				Result:      opts.result,
				ScanType:    opts.scanType,
				ScanResult:  opts.scanResult,
				ScanState:   opts.scanState,
				MaxItems:    opts.limit,
			}

			if opts.sort != "" {
				filter.OrderColumn = opts.sort
				if opts.desc {
					asc := false
					filter.OrderAscending = &asc
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

			tasks, err := httpClient.TaskSessions(ctx, filter)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, tasks)
			}

			rows := make([]taskRow, 0, len(tasks))
			for _, task := range tasks {
				name := strings.TrimSpace(task.Name)
				if name == "" {
					name = task.ID
				}
				sessionID := strings.TrimSpace(task.SessionID)
				rows = append(rows, taskRow{
					Name:        name,
					Type:        task.Type,
					SessionType: task.SessionType,
					State:       task.State,
					Result:      sessionResultShort(task.Result),
					Progress:    taskProgressString(task.Progress),
					CreatedAt:   formatTimestampValue(task.CreationTime.Time),
					EndedAt:     formatAPITime(task.EndTime),
					SessionID:   sessionID,
					TaskID:      task.ID,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by task name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.sessionID, "session-id", "", "Filter by parent session ID")
	cmd.Flags().StringVar(&opts.taskType, "type", "", "Filter by task type")
	cmd.Flags().StringVar(&opts.sessionType, "session-type", "", "Filter by parent session type")
	cmd.Flags().StringVar(&opts.state, "state", "", "Filter by task state")
	cmd.Flags().StringVar(&opts.result, "result", "", "Filter by task result")
	cmd.Flags().StringVar(&opts.scanType, "scan-type", "", "Filter by scan type")
	cmd.Flags().StringVar(&opts.scanResult, "scan-result", "", "Filter by scan result")
	cmd.Flags().StringVar(&opts.scanState, "scan-state", "", "Filter by scan state")
	cmd.Flags().StringVar(&opts.createdAfter, "created-since", "", "Include tasks created on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.createdBefore, "created-before", "", "Include tasks created on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.endedAfter, "ended-since", "", "Include tasks ended on/after RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.endedBefore, "ended-before", "", "Include tasks ended on/before RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort task sessions by column (e.g. creationTime, endTime)")
	cmd.Flags().BoolVar(&opts.desc, "desc", false, "Sort in descending order (default ascending)")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Maximum number of task sessions to return (default: all)")

	return cmd
}

func taskDescribeCmd() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Show detailed information about a task session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(taskID) == "" {
				return requireFlag("--id", "provide the task session ID to describe")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			detail, err := httpClient.TaskSessionDetail(ctx, taskID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail.Raw)
			}

			name := strings.TrimSpace(detail.Summary.Name)
			if name == "" {
				name = detail.Summary.ID
			}

			view := taskDetailView{
				TaskID:      detail.Summary.ID,
				Name:        name,
				Type:        detail.Summary.Type,
				SessionType: detail.Summary.SessionType,
				SessionID:   detail.Summary.SessionID,
				State:       detail.Summary.State,
				Result:      sessionResult(detail.Summary.Result),
				Progress:    taskProgressString(detail.Summary.Progress),
				CreatedAt:   formatTimestampValue(detail.Summary.CreationTime.Time),
				EndedAt:     formatAPITime(detail.Summary.EndTime),
				Payload:     formatYAMLBlock(detail.Raw),
			}

			return output.Print(format, view)
		},
	}

	cmd.Flags().StringVar(&taskID, "id", "", "Task session ID to describe")
	return cmd
}

func taskLogsCmd() *cobra.Command {
	opts := sessionLogsOptions{}
	var taskID string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show task session log records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(taskID) == "" {
				return requireFlag("--id", "provide the task session ID to inspect")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			records, err := httpClient.TaskSessionLogs(ctx, taskID, client.SessionLogsFilter{Status: opts.status})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, records)
			}

			rows := make([]taskLogRow, 0, len(records))
			for _, rec := range records {
				rows = append(rows, taskLogRow{
					ID:          rec.ID,
					Status:      rec.Status,
					Start:       formatTimestamp(rec.StartTime),
					Updated:     formatTimestamp(rec.UpdateTime),
					Title:       rec.Title,
					Description: rec.Description,
					Info:        rec.AdditionalInfo,
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.status, "status", "", "Filter log records by status")
	cmd.Flags().StringVar(&taskID, "id", "", "Task session ID whose logs to fetch")

	return cmd
}

type taskRow struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	SessionType string `json:"Session Type"`
	State       string `json:"State"`
	Result      string `json:"Result"`
	Progress    string `json:"Progress"`
	CreatedAt   string `json:"Created"`
	EndedAt     string `json:"Ended"`
	SessionID   string `json:"Session ID"`
	TaskID      string `json:"Task ID"`
}

type taskDetailView struct {
	TaskID      string `json:"Task ID"`
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	SessionType string `json:"Session Type"`
	SessionID   string `json:"Session ID"`
	State       string `json:"State"`
	Result      string `json:"Result"`
	Progress    string `json:"Progress"`
	CreatedAt   string `json:"Created"`
	EndedAt     string `json:"Ended"`
	Payload     string `json:"Payload"`
}

type taskLogRow struct {
	ID          int    `json:"ID"`
	Status      string `json:"Status"`
	Start       string `json:"Started"`
	Updated     string `json:"Updated"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Info        string `json:"Additional Info"`
}

func taskProgressString(progress *client.ProgressInfo) string {
	if progress == nil || progress.ProgressPercent == nil {
		return ""
	}
	return fmt.Sprintf("%d%%", *progress.ProgressPercent)
}
