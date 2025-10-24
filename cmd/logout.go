package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/config"
	"github.com/Coku2015/veeamgo/internal/session"
	"github.com/Coku2015/veeamgo/pkg/helptext"
)

func logoutCmd() *cobra.Command {
	var clearAll bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: helptext.LogoutShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := ensureConfigPath(opts.configPath)
			if err != nil {
				return err
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			manager, err := session.NewManager()
			if err != nil {
				return err
			}

			if clearAll {
				if err := manager.DeleteAll(); err != nil {
					return err
				}
				if cfg.DefaultProfile != "" {
					cfg.DefaultProfile = ""
					if err := config.Save(cfgPath, cfg); err != nil {
						return err
					}
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cleared all cached sessions.")
				return nil
			}

			profileName := activeProfile(cfg.DefaultProfile)
			if err := manager.Delete(profileName); err != nil {
				if errors.Is(err, session.ErrNotFound) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No cached session for profile %q\n", profileName)
					return nil
				}
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cleared session for profile %q\n", profileName)
			return nil
		},
	}

	cmd.Flags().BoolVar(&clearAll, "all", false, "Clear sessions for all profiles")
	return cmd
}
