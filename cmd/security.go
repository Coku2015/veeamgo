package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func securityAnalyzerGetCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "securityanalyzer",
		Short: "Security & Compliance Analyzer insights",
	}
	root.AddCommand(securityAnalyzerResultsCmd())
	return root
}

func securityAnalyzerDescribeCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "securityanalyzer",
		Short: "Security & Compliance Analyzer insights",
	}
	root.AddCommand(securityAnalyzerScheduleCmd())
	return root
}

func securityAnalyzerScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Show analyzer schedule configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			schedule, err := httpClient.SecurityAnalyzerSchedule(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, schedule.Raw)
			}

			custom := schedule.Settings.CustomNotificationConfig
			view := securityAnalyzerScheduleView{
				DailyScanEnabled: yesNo(schedule.Settings.DailyScanEnabled),
				DailyScanTime:    schedule.Settings.DailyScanLocalTime,
				SendResults:      yesNo(schedule.Settings.SendScanResults),
				Recipients:       schedule.Settings.Recipients,
				NotificationType: schedule.Settings.NotificationType,
			}
			if custom != nil {
				view.CustomSubject = custom.Subject
				view.NotifyOnSuccess = yesNo(custom.NotifyOnSuccess)
				view.NotifyOnWarning = yesNo(custom.NotifyOnWarning)
				view.NotifyOnError = yesNo(custom.NotifyOnError)
			}

			return output.Print(format, view)
		},
	}
	return cmd
}

func securityAnalyzerResultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results",
		Short: "List analyzer compliance results",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			result, err := httpClient.SecurityAnalyzerBestPractices(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			return output.Print(format, securityAnalyzerRows(result.Items))
		},
	}
	return cmd
}

func securityAnalyzerStartCmd() *cobra.Command {
	var (
		assumeYes bool
		wait      bool
	)

	cmd := &cobra.Command{
		Use:   "securityanalyzer",
		Short: "Start Security & Compliance Analyzer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !assumeYes {
				confirmed, err := promptForConfirmation(cmd, "Start Security & Compliance Analyzer")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled start request.")
					return nil
				}
			}

			timeout := 2 * time.Minute
			if wait {
				timeout = 30 * time.Minute
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			session, err := httpClient.StartSecurityAnalyzer(ctx)
			if err != nil {
				return err
			}

			message := "Started Security & Compliance Analyzer."
			if session != nil {
				message = fmt.Sprintf("Started Security & Compliance Analyzer.%s", formatSessionTail(session))
			}

			if wait && session != nil {
				waitCtx, waitCancel := context.WithTimeout(ctx, timeout)
				defer waitCancel()

				finalSession, err := httpClient.WaitForSession(waitCtx, session.ID, 5*time.Second)
				if err != nil {
					return err
				}
				session = finalSession
				message = fmt.Sprintf("Security & Compliance Analyzer finished.%s", formatSessionTail(session))

				bestPractices, err := httpClient.SecurityAnalyzerBestPractices(waitCtx)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to retrieve analyzer results: %v\n", err)
				} else if bestPractices != nil {
					return printSecurityAnalyzerSummaryAndResults(cmd, session, message, bestPractices)
				}
			}

			return printSessionMessage(cmd, session, message)
		},
	}

	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the analyzer session to finish")
	return cmd
}

type securityAnalyzerScheduleView struct {
	DailyScanEnabled string `json:"Daily Scan Enabled"`
	DailyScanTime    string `json:"Daily Scan Time,omitempty"`
	SendResults      string `json:"Send Results"`
	Recipients       string `json:"Recipients,omitempty"`
	NotificationType string `json:"Notification Type"`
	CustomSubject    string `json:"Custom Subject,omitempty"`
	NotifyOnSuccess  string `json:"Notify On Success,omitempty"`
	NotifyOnWarning  string `json:"Notify On Warning,omitempty"`
	NotifyOnError    string `json:"Notify On Error,omitempty"`
}

type securityAnalyzerResultRow struct {
	BestPractice string `json:"Best Practice"`
	Status       string `json:"Status"`
	Note         string `json:"Note,omitempty"`
}

func securityAnalyzerRows(items []client.SecurityBestPractice) []securityAnalyzerResultRow {
	rows := make([]securityAnalyzerResultRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, securityAnalyzerResultRow{
			BestPractice: item.BestPractice,
			Status:       item.Status,
			Note:         item.Note,
		})
	}
	return rows
}

func printSecurityAnalyzerSummaryAndResults(cmd *cobra.Command, session *client.Session, message string, result *client.SecurityAnalyzerBestPracticesResult) error {
	format := outputFormat()
	if format == "json" {
		payload := map[string]any{
			"message": message,
			"session": session,
			"results": result.Raw,
		}
		return output.Print(format, payload)
	}

	fmt.Fprintln(cmd.OutOrStdout(), message)
	fmt.Fprintln(cmd.OutOrStdout())
	return output.Print(format, securityAnalyzerRows(result.Items))
}
