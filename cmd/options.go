package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/pkg/output"
)

func optionsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "options",
		Short: "Global platform settings",
	}
	root.AddCommand(optionsGetCmd())
	root.AddCommand(newConfigurationBackupDescribeCmd("config-backup", true, []string{cmdDescribeUse}))
	root.AddCommand(newTrafficRuleGetCmd("traffic", true, []string{cmdGetUse}))
	return root
}

func optionsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "Show general options",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			opts, err := httpClient.GeneralOptions(ctx)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, opts.Raw)
			}

			row := optionsSummaryRow{
				NotificationEnabled:         yesNo(opts.NotificationEnabled),
				StorageSpaceAlertsEnabled:   yesNo(opts.StorageSpaceThresholdEnabled),
				DatastoreSpaceAlertsEnabled: yesNo(opts.DatastoreSpaceThresholdEnabled),
				SkipVmSpaceThresholdEnabled: yesNo(opts.SkipVMSpaceThresholdEnabled),
				NotifySupportExpiration:     yesNo(opts.NotifyOnSupportExpiration),
				NotifyUpdates:               yesNo(opts.NotifyOnUpdates),
				SIEMSNMPEnabled:             yesNo(opts.SIEMSNMPEnabled),
				SIEMSyslogEnabled:           yesNo(opts.SIEMSyslogEnabled),
			}

			return output.Print(format, []optionsSummaryRow{row})
		},
	}
	return cmd
}

type optionsSummaryRow struct {
	NotificationEnabled         string `json:"notificationEnabled"`
	StorageSpaceAlertsEnabled   string `json:"storageSpaceAlertsEnabled"`
	DatastoreSpaceAlertsEnabled string `json:"datastoreSpaceAlertsEnabled"`
	SkipVmSpaceThresholdEnabled string `json:"skipVmSpaceThresholdEnabled"`
	NotifySupportExpiration     string `json:"notifySupportExpiration"`
	NotifyUpdates               string `json:"notifyUpdates"`
	SIEMSNMPEnabled             string `json:"siemSnmpEnabled"`
	SIEMSyslogEnabled           string `json:"siemSyslogEnabled"`
}
