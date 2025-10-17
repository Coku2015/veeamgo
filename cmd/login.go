package cmd

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/internal/config"
	"github.com/veeamgo/veeamgo/internal/session"
)

func loginCmd() *cobra.Command {
	var (
		flagServer     string
		flagUsername   string
		flagPassword   string
		flagInsecure   bool
		flagScope      string
		flagSetDefault bool
		flagSavePass   bool
		flagPort       int
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a Veeam Backup & Replication server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			cfgPath, err := ensureConfigPath(opts.configPath)
			if err != nil {
				return err
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			profileName := activeProfile(cfg.DefaultProfile)
			existing := cfg.GetProfile(profileName)

			if flagServer == "" && existing != nil {
				flagServer = existing.ServerURL
			}
			if flagServer == "" {
				return requireFlag("--server", "provide the VBR server host name or URL")
			}

			if flagUsername == "" && existing != nil {
				flagUsername = existing.Username
			}
			if flagUsername == "" {
				return requireFlag("--username", "supply the account used for REST authentication")
			}

			if flagPassword == "" {
				pass, err := promptPassword("Password: ")
				if err != nil {
					return err
				}
				flagPassword = pass
			}

			if existing != nil && !cmd.Flags().Changed("insecure") {
				flagInsecure = existing.Insecure
			}

			serverURL, err := normalizeServerURL(flagServer, flagPort, cmd.Flags().Changed("port"))
			if err != nil {
				return err
			}

			sess, err := client.Authenticate(ctx, client.Credentials{
				BaseURL:  serverURL,
				Username: flagUsername,
				Password: flagPassword,
				Insecure: flagInsecure,
				Scope:    flagScope,
			})
			if err != nil {
				return err
			}

			manager, err := session.NewManager()
			if err != nil {
				return err
			}
			if err := manager.Store(profileName, *sess); err != nil {
				return err
			}

			passwordToPersist := ""
			if existing != nil {
				passwordToPersist = existing.Password
			}
			if flagSavePass {
				passwordToPersist = flagPassword
			}

			cfg.SetProfile(profileName, config.Profile{
				ServerURL: serverURL,
				Username:  flagUsername,
				Password:  passwordToPersist,
				Insecure:  flagInsecure,
			})
			if flagSetDefault || cfg.DefaultProfile == "" {
				cfg.DefaultProfile = profileName
			}

			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s as %s using profile %q\n", serverURL, flagUsername, profileName)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagServer, "server", "", "Server host name or base URL (e.g. vbr.example.com)")
	cmd.Flags().StringVar(&flagUsername, "username", "", "Username for authentication")
	cmd.Flags().StringVar(&flagPassword, "password", "", "Password (omit to prompt securely)")
	cmd.Flags().BoolVar(&flagInsecure, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().StringVar(&flagScope, "scope", "", "Optional OAuth scopes")
	cmd.Flags().BoolVar(&flagSetDefault, "set-default", false, "Set this profile as the default after login")
	cmd.Flags().BoolVar(&flagSavePass, "save-password", false, "Persist the password in the config file for automatic authentication")
	cmd.Flags().IntVar(&flagPort, "port", 9419, "Server REST API port (default 9419 when host is provided)")

	return cmd
}

func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	passwordBytes, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimSpace(string(passwordBytes)), nil
}

func normalizeServerURL(input string, port int, portOverridden bool) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("server cannot be empty")
	}

	address := trimmed
	fromHostOnly := !strings.Contains(trimmed, "://")
	if fromHostOnly {
		address = "https://" + trimmed
	}

	u, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("parse server address: %w", err)
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("server host is empty")
	}

	finalPort := u.Port()
	if portOverridden {
		if port <= 0 {
			return "", fmt.Errorf("port must be positive")
		}
		finalPort = strconv.Itoa(port)
	} else if fromHostOnly {
		if port <= 0 {
			port = 9419
		}
		finalPort = strconv.Itoa(port)
	}

	hostPort := host
	if finalPort != "" {
		hostPort = net.JoinHostPort(host, finalPort)
	}

	return (&url.URL{Scheme: u.Scheme, Host: hostPort}).String(), nil
}
