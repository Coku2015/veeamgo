package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

type managedServerRescanOptions struct {
	id   string
	name string
	all  bool
	wait bool
}

func managedServerRescanCmd() *cobra.Command {
	opts := managedServerRescanOptions{}

	cmd := &cobra.Command{
		Use:   cmdRescanUse,
		Short: helptext.ManagedServerRescanShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManagedServerRescan(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.id, "id", "", "Managed server ID to rescan")
	cmd.Flags().StringVar(&opts.name, "name", "", "Managed server name to rescan")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Rescan all managed servers")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the rescan session to finish")

	return cmd
}

func runManagedServerRescan(cmd *cobra.Command, opts *managedServerRescanOptions) error {
	idTrim := strings.TrimSpace(opts.id)
	nameTrim := strings.TrimSpace(opts.name)

	if opts.all {
		if idTrim != "" || nameTrim != "" {
			return fmt.Errorf("do not combine --all with --id or --name")
		}
	} else {
		selected := 0
		if idTrim != "" {
			selected++
		}
		if nameTrim != "" {
			selected++
		}
		if selected == 0 {
			return fmt.Errorf("provide --id or --name (or use --all)")
		}
		if selected > 1 {
			return fmt.Errorf("provide either --id or --name, not both")
		}
	}

	timeout := 5 * time.Minute
	if opts.wait {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	var (
		targetSession *client.Session
		targetServer  *client.ManagedServer
	)

	if opts.all {
		targetSession, err = httpClient.RescanAllManagedServers(ctx)
		if err != nil {
			return fmt.Errorf("rescan all managed servers: %w", err)
		}
	} else {
		if nameTrim != "" {
			targetServer, err = resolveManagedServerByName(ctx, httpClient, nameTrim)
			if err != nil {
				return err
			}
			idTrim = targetServer.ID
		} else {
			targetServer, err = httpClient.ManagedServer(ctx, idTrim)
			if err != nil {
				return err
			}
		}

		targetSession, err = httpClient.RescanManagedServer(ctx, idTrim)
		if err != nil {
			return fmt.Errorf("rescan managed server %q (%s): %w", targetServer.Name, idTrim, err)
		}
	}

	waited := false
	if targetSession != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, targetSession.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for managed server rescan session: %w", waitErr)
		}
		targetSession = final
		waited = true
	}

	return printManagedServerRescanResult(cmd, targetSession, targetServer, idTrim, waited, opts.all)
}

func printManagedServerRescanResult(cmd *cobra.Command, session *client.Session, server *client.ManagedServer, fallbackID string, waited bool, all bool) error {
	format := outputFormat()

	if format == "json" {
		scope := "single"
		name := fallbackID
		serverType := ""
		if all {
			scope = "all"
			name = ""
		} else if server != nil {
			if trimmed := strings.TrimSpace(server.Name); trimmed != "" {
				name = trimmed
			}
			serverType = server.Type
		}

		status := "rescan started"
		result := ""
		message := ""
		if session == nil {
			status = "rescan completed"
		} else if waited {
			status = "rescan finished"
			if session.Result != nil {
				result = session.Result.Result
				message = session.Result.Message
			}
		}

		payload := map[string]any{
			"scope":  scope,
			"waited": waited,
			"status": status,
		}
		if scope == "single" {
			payload["server"] = map[string]string{
				"id":   fallbackID,
				"name": name,
				"type": serverType,
			}
		}
		if session != nil {
			payload["session"] = session
		}
		if result != "" {
			payload["result"] = result
		}
		if strings.TrimSpace(message) != "" {
			payload["details"] = message
		}

		return output.Print(format, payload)
	}

	label := "Managed server"
	name := fallbackID
	if all {
		label = "Managed servers"
		name = ""
	} else if server != nil {
		if pretty := managedServerTypeDisplay(server.Type); pretty != "" {
			label = pretty
		}
		if trimmed := strings.TrimSpace(server.Name); trimmed != "" {
			name = trimmed
		}
		if strings.TrimSpace(name) == "" {
			name = fallbackID
		}
	}

	if session == nil {
		if all {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s rescan completed.\n", label)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q rescan completed.\n", label, name)
		}
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		if all {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s rescan finished with result %s (session %s).\n", label, result, session.ID)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q rescan finished with result %s (session %s).\n", label, name, result, session.ID)
		}
		if session.Result != nil && strings.TrimSpace(session.Result.Message) != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", session.Result.Message)
		}
		return nil
	}

	if all {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started rescan for %s (session %s).\n", strings.ToLower(label), session.ID)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started rescan for %s %q (session %s).\n", label, name, session.ID)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}
