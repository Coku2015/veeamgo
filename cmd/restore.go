package cmd

import "github.com/spf13/cobra"

func restoreCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   cmdRestoreUse,
		Short: "Instant recovery utilities",
	}
	root.AddCommand(restoreMountCmd())
	return root
}
