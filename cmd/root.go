package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/paths"
)

type rootOptions struct {
	configPath   string
	profile      string
	outputFormat string
	apiVersion   string
}

var (
	buildVersion = "dev"
	rootCmd      = &cobra.Command{
		Use:   "veeamgo",
		Short: "VeeamGo CLI connects to Veeam Backup & Replication REST API",
	}
	opts = rootOptions{
		outputFormat: "table",
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = buildVersion
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Path to configuration file")
	rootCmd.PersistentFlags().StringVar(&opts.profile, "profile", "", "Profile name to use")
	rootCmd.PersistentFlags().StringVar(&opts.outputFormat, "output", opts.outputFormat, "Output format (table|json)")
	rootCmd.PersistentFlags().StringVar(&opts.apiVersion, "api-version", "", "Override API version used for REST calls (for example 1.2-rev1)")

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(logoutCmd())
	rootCmd.AddCommand(jobCmd())

}

func ensureConfigPath(path string) (string, error) {
	resolved, err := paths.ResolveConfigPath(path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("ensure config directory: %w", err)
	}

	return resolved, nil
}

func activeProfile(defaultProfile string) string {
	if opts.profile != "" {
		return opts.profile
	}
	if defaultProfile != "" {
		return defaultProfile
	}
	return "default"
}

func outputFormat() string {
	switch opts.outputFormat {
	case "json", "table":
		return opts.outputFormat
	default:
		return "table"
	}
}
