package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getVerbCmd())
	rootCmd.AddCommand(describeVerbCmd())
	rootCmd.AddCommand(rescanVerbCmd())
}

func getVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdGetUse,
		Short: "Retrieve resources",
	}

	cmd.AddCommand(wrapForVerb(serverGetCmd, cmdServerUse))
	cmd.AddCommand(wrapForVerb(repositoryGetCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(jobGetCmd, cmdJobUse))
	cmd.AddCommand(wrapForVerb(restorePointGetCmd, cmdRestorePointUse))
	cmd.AddCommand(wrapForVerb(replicaGetCmd, cmdReplicaUse))
	cmd.AddCommand(wrapForVerb(proxyListCmd, "proxy"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newTrafficRuleGetCmd(cmdGetUse, false, nil) }, "trafficrule"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newExclusionVMGetCmd(cmdGetUse, nil, false) }, "exclusionvm"))
	cmd.AddCommand(sessionGetVerbCmd())
	cmd.AddCommand(backupGetVerbCmd())
	cmd.AddCommand(objectsInBackupCmd())
	cmd.AddCommand(securityGetVerbCmd())
	cmd.AddCommand(licenseVerbCmd())

	return cmd
}

func describeVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdDescribeUse,
		Short: "Describe resources",
	}

	cmd.AddCommand(wrapForVerb(serverDescribeCmd, cmdServerUse))
	cmd.AddCommand(wrapForVerb(repositoryDescribeCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(jobDescribeCmd, cmdJobUse))
	cmd.AddCommand(wrapForVerb(proxyDescribeCmd, "proxy"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newConfigurationBackupDescribeCmd(cmdDescribeUse, false, nil) }, "configurationbackup"))
	cmd.AddCommand(wrapForVerb(func() *cobra.Command { return newExclusionVMDescribeCmd(cmdDescribeUse, []string{"show"}, false) }, "exclusionvm"))
	cmd.AddCommand(sessionDescribeVerbCmd())
	cmd.AddCommand(securityDescribeVerbCmd())

	return cmd
}

func rescanVerbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: "Trigger rescans",
	}

	cmd.AddCommand(wrapForVerb(repositoryRescanCmd, cmdRepositoryUse))
	cmd.AddCommand(wrapForVerb(serverRescanCmd, cmdServerUse))

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
	logs.Use = "logs <session-id>"

	list.AddCommand(logs)
	return list
}

func sessionDescribeVerbCmd() *cobra.Command {
	describe := sessionDescribeCmd()
	describe.Use = "session <session-id>"

	current := sessionCurrentCmd()
	current.Use = "current"

	describe.AddCommand(current)
	return describe
}

func backupGetVerbCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "backup",
		Short: "Backup inspection helpers",
	}

	root.AddCommand(backupFilesCmd())
	root.AddCommand(backupObjectsCmd())
	return root
}

func securityGetVerbCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "security",
		Short: "Security analyzer configuration and results",
	}

	analyzer := &cobra.Command{
		Use:   "analyzer",
		Short: "Security & Compliance Analyzer insights",
	}
	analyzer.AddCommand(securityAnalyzerSendResultsCmd())
	root.AddCommand(analyzer)
	return root
}

func securityDescribeVerbCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "security",
		Short: "Security analyzer configuration and results",
	}

	analyzer := &cobra.Command{
		Use:   "analyzer",
		Short: "Security & Compliance Analyzer insights",
	}
	analyzer.AddCommand(securityAnalyzerScheduleCmd())
	root.AddCommand(analyzer)
	return root
}
