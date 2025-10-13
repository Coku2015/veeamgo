package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/pkg/output"
)

func configurationBackupCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "configurationbackup",
		Short: "Configuration backup policies",
	}
	root.AddCommand(newConfigurationBackupDescribeCmd(cmdDescribeUse, false, nil))
	return root
}

func newConfigurationBackupDescribeCmd(use string, hidden bool, aliases []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Describe configuration backup settings",
		Hidden:  hidden,
		Args:    cobra.NoArgs,
		RunE:    runConfigurationBackupDescribe,
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
