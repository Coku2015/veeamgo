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

func sessionCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "session",
		Short: "Session management and diagnostics",
	}
	root.AddCommand(sessionListCmd())
	root.AddCommand(sessionDescribeCmd())
	root.AddCommand(sessionLogsCmd())
	return root
}

type sessionListOptions struct {
	name          string
	jobID         string
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

func sessionListCmd() *cobra.Command {
	opts := sessionListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Veeam job sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.SessionsFilter{
				Name:     opts.name,
				JobID:    opts.jobID,
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

	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by session name (supports * wildcards)")
	cmd.Flags().StringVar(&opts.jobID, "job", "", "Filter by job ID")
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

func sessionDescribeCmd() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Show detailed information about a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(sessionID) == "" {
				return requireFlag("--id", "provide the session ID to describe")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			detail, err := httpClient.SessionDetail(ctx, sessionID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail.Raw)
			}

			view := sessionDetailView{
				ID:              detail.Session.ID,
				Name:            detail.Session.Name,
				Type:            detail.Session.SessionType,
				State:           detail.Session.State,
				Result:          sessionResult(detail.Session.Result),
				Progress:        fmt.Sprintf("%d%%", detail.Session.ProgressPercent),
				CreatedAt:       formatTimestampValue(detail.Session.CreationTime),
				EndedAt:         formatTimestamp(detail.Session.EndTime),
				JobID:           detail.Session.JobID,
				ParentSessionID: detail.Session.ParentSessionID,
				ResourceID:      detail.Session.ResourceID,
				ResourceRef:     detail.Session.ResourceReference,
				Platform:        detail.Session.PlatformName,
				InitiatedBy:     detail.Session.InitiatedBy,
			}

			return output.Print(format, view)
		},
	}

	cmd.Flags().StringVar(&sessionID, "id", "", "Session ID to describe")
	return cmd
}

type sessionLogsOptions struct {
	status string
}

func sessionLogsCmd() *cobra.Command {
	opts := sessionLogsOptions{}
	var sessionID string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show session log records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionID) == "" {
				return requireFlag("--id", "provide the session ID to inspect")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			records, err := httpClient.SessionLogs(ctx, sessionID, client.SessionLogsFilter{
				Status: opts.status,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, records)
			}

			rows := make([]sessionLogRow, 0, len(records))
			for _, rec := range records {
				rows = append(rows, sessionLogRow{
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
	cmd.Flags().StringVar(&sessionID, "id", "", "Session ID whose logs to fetch")

	return cmd
}

type sessionRow struct {
	Name      string `json:"Name"`
	Type      string `json:"Type"`
	State     string `json:"State"`
	Result    string `json:"Result"`
	Progress  string `json:"Progress"`
	CreatedAt string `json:"Created"`
	EndedAt   string `json:"Ended"`
	SessionID string `json:"Session ID"`
}

type sessionDetailView struct {
	ID              string `json:"ID"`
	Name            string `json:"Name"`
	Type            string `json:"Type"`
	State           string `json:"State"`
	Result          string `json:"Result"`
	Progress        string `json:"Progress"`
	CreatedAt       string `json:"Created"`
	EndedAt         string `json:"Ended"`
	JobID           string `json:"Job ID"`
	ParentSessionID string `json:"Parent Session ID"`
	ResourceID      string `json:"Resource ID"`
	ResourceRef     string `json:"Resource Reference"`
	Platform        string `json:"Platform"`
	InitiatedBy     string `json:"Initiated By"`
}

type sessionLogRow struct {
	ID          int    `json:"ID"`
	Status      string `json:"Status"`
	Start       string `json:"Started"`
	Updated     string `json:"Updated"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Info        string `json:"Additional Info"`
}

func sessionResult(res *client.SessionResult) string {
	if res == nil {
		return ""
	}
	if res.Message != "" {
		return fmt.Sprintf("%s (%s)", res.Result, res.Message)
	}
	return res.Result
}

func sessionResultShort(res *client.SessionResult) string {
	if res == nil {
		return ""
	}
	if value := strings.TrimSpace(res.Result); value != "" {
		parts := strings.Fields(value)
		if len(parts) > 0 {
			return parts[0]
		}
		return value
	}
	if value := strings.TrimSpace(res.Message); value != "" {
		parts := strings.Fields(value)
		if len(parts) > 0 {
			return parts[0]
		}
		return value
	}
	return ""
}
