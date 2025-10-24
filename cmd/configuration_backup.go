package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

func configBackupStartCmd() *cobra.Command {
	var (
		assumeYes bool
		wait      bool
	)

	cmd := &cobra.Command{
		Use:   "configurationbackup",
		Short: helptext.ConfigurationBackupStartShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !assumeYes {
				confirmed, err := promptForConfirmation(cmd, "Start configuration backup")
				if err != nil {
					return err
				}
				if !confirmed {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled start request.")
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

			session, err := httpClient.StartConfigBackup(ctx)
			if err != nil {
				return err
			}

			message := "Started configuration backup."
			if session != nil {
				message = fmt.Sprintf("Started configuration backup.%s", formatSessionTail(session))
			}

			if wait && session != nil {
				waitCtx, waitCancel := context.WithTimeout(ctx, timeout)
				defer waitCancel()

				finalSession, err := httpClient.WaitForSession(waitCtx, session.ID, 5*time.Second)
				if err != nil {
					return err
				}
				session = finalSession
				message = fmt.Sprintf("Configuration backup finished.%s", formatSessionTail(session))
			}

			return printSessionMessage(cmd, session, message)
		},
	}

	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Confirm without prompting (required to execute)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the backup session to finish")
	return cmd
}

func newConfigurationBackupDescribeCmd(use string, hidden bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:    use,
		Short:  helptext.ConfigurationBackupDescribeShort,
		Hidden: hidden,
		Args:   cobra.NoArgs,
		RunE:   runConfigurationBackupDescribe,
	}
	return cmd
}

func runConfigurationBackupDescribe(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	cfg, err := httpClient.ConfigBackup(ctx)
	if err != nil {
		return err
	}

	format := outputFormat()
	if format == "json" {
		return output.Print(format, cfg.Raw)
	}

	view := configBackupView{
		Enabled:              yesNo(cfg.IsEnabled),
		RepositoryID:         cfg.BackupRepositoryID,
		RestorePointsKeep:    cfg.RestorePointsToKeep,
		EncryptionEnabled:    yesNo(cfg.EncryptionEnabled),
		EncryptionPasswordID: cfg.EncryptionPassword,
		LastRunAt:            formatAPITime(cfg.LastRunTime),
		LastSessionID:        cfg.LastSessionID,
	}

	return output.Print(format, view)
}

type configBackupView struct {
	Enabled              string `json:"Enabled"`
	RepositoryID         string `json:"Repository ID"`
	RestorePointsKeep    int    `json:"Restore Points To Keep"`
	EncryptionEnabled    string `json:"Encryption Enabled"`
	EncryptionPasswordID string `json:"Encryption Password ID,omitempty"`
	LastRunAt            string `json:"Last Successful Run,omitempty"`
	LastSessionID        string `json:"Last Session ID,omitempty"`
}
