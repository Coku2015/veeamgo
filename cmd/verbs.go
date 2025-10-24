package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/pkg/helptext"
)

func init() {
	rootCmd.AddCommand(getVerbCmd())
	rootCmd.AddCommand(describeVerbCmd())
	rootCmd.AddCommand(rescanVerbCmd())
	rootCmd.AddCommand(templateVerbCmd())
	rootCmd.AddCommand(addVerbCmd())
	rootCmd.AddCommand(editVerbCmd())
	rootCmd.AddCommand(enableVerbCmd())
	rootCmd.AddCommand(disableVerbCmd())
	rootCmd.AddCommand(cloneVerbCmd())
	rootCmd.AddCommand(deleteVerbCmd())
	rootCmd.AddCommand(startVerbCmd())
	rootCmd.AddCommand(stopVerbCmd())
	rootCmd.AddCommand(retryVerbCmd())
	rootCmd.AddCommand(migrateVerbCmd())
}

func getVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: helptext.GetVerbShort,
	}

	cmd.AddCommand(wrapForVerb(serverGetCmd, cmdServerUse))
	cmd.AddCommand(wrapForVerb(managedServerGetCmd, cmdManagedServerUse))
	cmd.AddCommand(wrapForVerb(repositoryGetCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(objectRepositoryGetCmd, cmdObjectRepositoryUse))
	cmd.AddCommand(wrapForVerb(scaleOutRepositoryGetCmd, cmdScaleOutRepositoryUse))
	cmd.AddCommand(wrapForVerb(wanAcceleratorGetCmd, cmdWanAcceleratorUse))
	cmd.AddCommand(wrapForVerb(generalOptionGetCmd, cmdGeneralOptionUse))
	cmd.AddCommand(wrapForVerb(jobGetCmd, cmdJobUse))
	cmd.AddCommand(wrapForVerb(restorePointGetCmd, cmdRestorePointUse))
	cmd.AddCommand(wrapForVerb(replicaGetCmd, cmdReplicaUse))
	cmd.AddCommand(wrapForVerb(proxyListCmd, "proxy"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newTrafficRuleGetCmd(cmdGetUse, false) }, "trafficrule"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newExclusionVMGetCmd(cmdGetUse, false) }, "exclusionvm"))
	cmd.AddCommand(wrapForVerb(inventoryGetRootCmd, cmdInventoryUse))
	cmd.AddCommand(fingerprintGetCmd())
	cmd.AddCommand(sessionGetVerbCmd())
	cmd.AddCommand(taskGetVerbCmd())
	cmd.AddCommand(backupGetVerbCmd())
	cmd.AddCommand(securityGetVerbCmd())
	cmd.AddCommand(wrapForVerb(malwareDetectionEventGetCmd, cmdMalwareDetectionEventUse))
	cmd.AddCommand(wrapForVerb(yaraRuleGetCmd, cmdYaraRuleUse))
	cmd.AddCommand(wrapForVerb(publishedDiskGetCmd, "publisheddisk"))
	cmd.AddCommand(licenseVerbCmd())

	return cmd
}

func describeVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: helptext.DescribeVerbShort,
	}

	cmd.AddCommand(wrapForVerb(managedServerDescribeCmd, cmdManagedServerUse))
	cmd.AddCommand(wrapForVerb(repositoryDescribeCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(objectRepositoryDescribeCmd, cmdObjectRepositoryUse))
	cmd.AddCommand(wrapForVerb(scaleOutRepositoryDescribeCmd, cmdScaleOutRepositoryUse))
	cmd.AddCommand(wrapForVerb(wanAcceleratorDescribeCmd, cmdWanAcceleratorUse))
	cmd.AddCommand(wrapForVerb(jobDescribeCmd, cmdJobUse))
	cmd.AddCommand(wrapForVerb(proxyDescribeCmd, "proxy"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newConfigurationBackupDescribeCmd(cmdDescribeUse, false) }, "configurationbackup"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newExclusionVMDescribeCmd(cmdDescribeUse, false) }, "exclusionvm"))
	cmd.AddCommand(wrapForVerb(inventoryDescribeRootCmd, cmdInventoryUse))
	cmd.AddCommand(sessionDescribeVerbCmd())
	cmd.AddCommand(taskDescribeVerbCmd())
	cmd.AddCommand(securityDescribeVerbCmd())

	return cmd
}

func rescanVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: helptext.RescanVerbShort,
	}

	cmd.AddCommand(wrapForVerb(repositoryRescanCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(objectRepositoryRescanCmd, cmdObjectRepositoryUse))
	cmd.AddCommand(wrapForVerb(managedServerRescanCmd, cmdManagedServerUse, cmdServerUse))

	return cmd
}

func templateVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdTemplateUse,
		Short: helptext.TemplateVerbShort,
	}
	cmd.AddCommand(templateJobCmd())
	return cmd
}

func addVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdAddUse,
		Short: helptext.AddVerbShort,
	}
	cmd.AddCommand(wrapForVerb(managedServerAddCmd, cmdManagedServerUse))
	cmd.AddCommand(wrapForVerb(proxyAddCmd, cmdProxyUse))
	cmd.AddCommand(wrapForVerb(repositoryAddCmd, cmdRepositoryUse))
	cmd.AddCommand(jobAddVerbCmd())
	cmd.AddCommand(wrapForVerb(objectRepositoryAddCmd, cmdObjectRepositoryUse))
	return cmd
}

func editVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdEditUse,
		Short: helptext.EditVerbShort,
	}
	cmd.AddCommand(jobEditVerbCmd())
	cmd.AddCommand(proxyEditCmd())
	cmd.AddCommand(wrapForVerb(repositoryEditCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(objectRepositoryEditCmd, cmdObjectRepositoryUse))
	return cmd
}

func wrapForVerb(factory func() *cobra.Command, use string, aliases ...string) *cobra.Command {
	cmd := factory()

	original := cmd.Use
	args := ""
	if parts := strings.SplitN(original, " ", 2); len(parts) == 2 {
		args = parts[1]
	}

	if args != "" {
		cmd.Use = strings.TrimSpace(use + " " + args)
	} else {
		cmd.Use = use
	}

	if len(aliases) > 0 {
		cmd.Aliases = append(cmd.Aliases, aliases...)
	}

	return cmd
}

func sessionGetVerbCmd() *cobra.Command {
	list := sessionListCmd()
	list.Use = "session"

	logs := sessionLogsCmd()
	logs.Use = "logs"

	list.AddCommand(logs)
	return list
}

func sessionDescribeVerbCmd() *cobra.Command {
	describe := sessionDescribeCmd()
	describe.Use = "session"

	return describe
}

func taskGetVerbCmd() *cobra.Command {
	list := taskListCmd()
	list.Use = "task"

	logs := taskLogsCmd()
	logs.Use = "logs"
	list.AddCommand(logs)

	return list
}

func taskDescribeVerbCmd() *cobra.Command {
	describe := taskDescribeCmd()
	describe.Use = "task"
	return describe
}

func backupGetVerbCmd() *cobra.Command {
	list := backupListCmd()

	root := &cobra.Command{
		Use:   "backup",
		Short: helptext.BackupListShort,
		RunE:  list.RunE,
	}

	root.Flags().AddFlagSet(list.Flags())
	root.AddCommand(list)
	root.AddCommand(backupFilesCmd())
	root.AddCommand(backupObjectsCmd())
	return root
}

func securityGetVerbCmd() *cobra.Command {
	return securityAnalyzerGetCmd()
}

func securityDescribeVerbCmd() *cobra.Command {
	return securityAnalyzerDescribeCmd()
}

func enableVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdEnableUse,
		Short: "Enable resources",
	}
	cmd.AddCommand(jobEnableCmd())
	cmd.AddCommand(proxyEnableCmd())
	return cmd
}

func disableVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdDisableUse,
		Short: "Disable resources",
	}
	cmd.AddCommand(jobDisableCmd())
	cmd.AddCommand(proxyDisableCmd())
	return cmd
}

func cloneVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdCloneUse,
		Short: "Clone resources",
	}
	cmd.AddCommand(jobCloneCmd())
	return cmd
}

func deleteVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdDeleteUse,
		Short: "Delete resources",
	}
	cmd.AddCommand(jobDeleteCmd())
	cmd.AddCommand(proxyDeleteCmd())
	cmd.AddCommand(wrapForVerb(managedServerDeleteCmd, cmdManagedServerUse))
	cmd.AddCommand(wrapForVerb(repositoryDeleteCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(objectRepositoryDeleteCmd, cmdObjectRepositoryUse))
	return cmd
}

func startVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdStartUse,
		Short: helptext.StartVerbShort,
	}
	cmd.AddCommand(jobStartCmd())
	cmd.AddCommand(jobQuickBackupCmd())
	cmd.AddCommand(securityAnalyzerStartCmd())
	cmd.AddCommand(configBackupStartCmd())
	cmd.AddCommand(publishDiskStartCmd())
	return cmd
}

func stopVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdStopUse,
		Short: helptext.StopVerbShort,
	}
	cmd.AddCommand(jobStopCmd())
	cmd.AddCommand(publishDiskStopCmd())
	return cmd
}

func retryVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdRetryUse,
		Short: helptext.RetryVerbShort,
	}
	cmd.AddCommand(jobRetryCmd())
	return cmd
}

func migrateVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdMigrateUse,
		Short: helptext.MigrateVerbShort,
	}
	return cmd
}
