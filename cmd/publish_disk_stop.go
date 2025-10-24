package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

type publishDiskStopOptions struct {
	ids  []string
	wait bool
}

func publishDiskStopCmd() *cobra.Command {
	opts := publishDiskStopOptions{}

	cmd := &cobra.Command{
		Use:   "publishdisk",
		Short: helptext.PublishDiskStopShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublishDiskStop(cmd, &opts)
		},
	}

	cmd.Flags().StringSliceVar(&opts.ids, "id", nil, "Published disk mount ID to stop (repeatable)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the unpublish session to finish")

	return cmd
}

func runPublishDiskStop(cmd *cobra.Command, opts *publishDiskStopOptions) error {
	ids := trimSlice(opts.ids)
	if len(ids) == 0 {
		return requireFlag("--id", "provide at least one mount ID")
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

	type stopResult struct {
		ID      string          `json:"id"`
		Session *client.Session `json:"session,omitempty"`
		Waited  bool            `json:"waited"`
		Error   string          `json:"error,omitempty"`
		Result  string          `json:"result,omitempty"`
		Message string          `json:"message,omitempty"`
	}

	format := outputFormat()
	results := make([]stopResult, 0, len(ids))

	for _, id := range ids {
		result := stopResult{ID: id}

		session, err := httpClient.UnpublishBackupContent(ctx, id)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		if opts.wait {
			final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
			if waitErr != nil {
				result.Error = waitErr.Error()
			} else {
				session = final
				result.Waited = true
				if session.Result != nil {
					result.Result = session.Result.Result
					result.Message = session.Result.Message
				}
			}
		}

		result.Session = session
		results = append(results, result)
	}

	if format == "json" {
		return output.Print(format, results)
	}

	for _, res := range results {
		if res.Error != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to stop published disk %s: %s\n", res.ID, res.Error)
			continue
		}

		if res.Session == nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No session returned when stopping published disk %s.\n", res.ID)
			continue
		}

		if res.Waited {
			resultText := res.Session.State
			if res.Session.Result != nil && res.Session.Result.Result != "" {
				resultText = res.Session.Result.Result
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped published disk %s with result %s (session %s).\n", res.ID, resultText, res.Session.ID)
			if res.Message != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", res.Message)
			}
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopping published disk %s (session %s).\n", res.ID, res.Session.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", res.Session.ID)
		}
	}

	return nil
}
