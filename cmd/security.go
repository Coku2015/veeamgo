package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/pkg/output"
)

func securityCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "security",
		Short: "Security analyzer configuration and results",
	}
	root.AddCommand(securityAnalyzerCmd())
	return root
}

func securityAnalyzerCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "analyzer",
		Short: "Security & Compliance Analyzer insights",
	}
	root.AddCommand(securityAnalyzerScheduleCmd())
	root.AddCommand(securityAnalyzerSendResultsCmd())
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

func securityAnalyzerSendResultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "send-results",
		Aliases: []string{"sendResults"},
		Short:   "List analyzer send-results compliance status",
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

			rows := make([]securityAnalyzerResultRow, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, securityAnalyzerResultRow{
					BestPractice: item.BestPractice,
					Status:       item.Status,
					Note:         item.Note,
				})
			}

			return output.Print(format, rows)
		},
	}
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
