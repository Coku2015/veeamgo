package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/pkg/output"
)

func generalOptionGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "Show global notification and SIEM options",
		Args:  cobra.NoArgs,
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

			view := generalOptionsView{
				NotificationsEnabled:    yesNo(opts.NotificationEnabled),
				StorageSpaceThreshold:   yesNo(opts.StorageSpaceThresholdEnabled),
				DatastoreSpaceThreshold: yesNo(opts.DatastoreSpaceThresholdEnabled),
				SkipVMSpaceThreshold:    yesNo(opts.SkipVMSpaceThresholdEnabled),
				SupportExpirationAlerts: yesNo(opts.NotifyOnSupportExpiration),
				UpdateNotifications:     yesNo(opts.NotifyOnUpdates),
				SIEMSNMPEvents:          yesNo(opts.SIEMSNMPEnabled),
				SIEMSyslogEvents:        yesNo(opts.SIEMSyslogEnabled),
				Config:                  formatYAMLBlock(opts.Raw),
			}

			return output.Print(format, view)
		},
	}

	return cmd
}

type generalOptionsView struct {
	NotificationsEnabled    string `json:"Notifications Enabled"`
	StorageSpaceThreshold   string `json:"Storage Space Threshold Alert"`
	DatastoreSpaceThreshold string `json:"Datastore Space Threshold Alert"`
	SkipVMSpaceThreshold    string `json:"Skip VM Space Threshold Alert"`
	SupportExpirationAlerts string `json:"Support Expiration Alerts"`
	UpdateNotifications     string `json:"Update Notifications"`
	SIEMSNMPEvents          string `json:"SIEM SNMP Events"`
	SIEMSyslogEvents        string `json:"SIEM Syslog Events"`
	Config                  string `json:"Config,omitempty"`
}
