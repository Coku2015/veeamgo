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

type managedServerDeleteOptions struct {
	id   string
	name string
	wait bool
	yes  bool
}

func managedServerDeleteCmd() *cobra.Command {
	opts := managedServerDeleteOptions{}

	cmd := &cobra.Command{
		Use:   cmdManagedServerUse,
		Short: helptext.ManagedServerDeleteShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManagedServerDelete(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.id, "id", "", "Managed server ID to delete")
	cmd.Flags().StringVar(&opts.name, "name", "", "Managed server name to delete")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the deletion session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required to execute)")

	return cmd
}

func runManagedServerDelete(cmd *cobra.Command, opts *managedServerDeleteOptions) error {
	idTrim := strings.TrimSpace(opts.id)
	nameTrim := strings.TrimSpace(opts.name)
	if idTrim == "" && nameTrim == "" {
		return fmt.Errorf("provide --id or --name")
	}
	if idTrim != "" && nameTrim != "" {
		return fmt.Errorf("provide either --id or --name, not both")
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

	var server *client.ManagedServer
	if nameTrim != "" {
		server, err = resolveManagedServerByName(ctx, httpClient, nameTrim)
		if err != nil {
			return err
		}
		idTrim = server.ID
	} else {
		server, err = httpClient.ManagedServer(ctx, idTrim)
		if err != nil {
			return err
		}
	}

	label := managedServerTypeDisplay(server.Type)
	if label == "" {
		label = "Managed server"
	}
	displayName := strings.TrimSpace(server.Name)
	if displayName == "" {
		displayName = idTrim
	}

	if !opts.yes {
		prompt := fmt.Sprintf("Delete %s %q from the backup infrastructure", label, displayName)
		confirmed, err := promptForConfirmation(cmd, prompt)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled deleting %s %q.\n", label, displayName)
			return nil
		}
	}

	session, err := httpClient.DeleteManagedServer(ctx, idTrim)
	if err != nil {
		return fmt.Errorf("delete managed server %q (%s): %w", displayName, idTrim, err)
	}

	waited := false
	if session != nil && opts.wait {
		final, waitErr := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
		if waitErr != nil {
			return fmt.Errorf("wait for managed server deletion session: %w", waitErr)
		}
		session = final
		waited = true
	}

	return printManagedServerDeletionResult(cmd, session, server, idTrim, waited)
}

func printManagedServerDeletionResult(cmd *cobra.Command, session *client.Session, server *client.ManagedServer, fallbackID string, waited bool) error {
	format := outputFormat()

	name := fallbackID
	serverType := ""
	label := "Managed server"
	if server != nil {
		if display := strings.TrimSpace(server.Name); display != "" {
			name = display
		}
		serverType = server.Type
		if pretty := managedServerTypeDisplay(server.Type); pretty != "" {
			label = pretty
		}
	}
	if strings.TrimSpace(name) == "" {
		name = "(unknown)"
	}

	if format == "json" {
		status := "deletion started"
		result := ""
		var message string
		if session == nil {
			status = "deletion completed"
		} else if waited {
			status = "deletion finished"
			if session.Result != nil {
				result = session.Result.Result
				message = session.Result.Message
			}
		}

		payload := map[string]any{
			"message": fmt.Sprintf("%s %q %s", label, name, status),
			"server": map[string]string{
				"id":   fallbackID,
				"name": name,
				"type": serverType,
			},
			"waited": waited,
		}
		if session != nil {
			payload["session"] = session
		}
		if result != "" {
			payload["result"] = result
		}
		if message != "" {
			payload["details"] = message
		}

		return output.Print(format, payload)
	}

	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q deletion completed.\n", label, name)
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q deletion finished with result %s (session %s).\n", label, name, result, session.ID)
		if session.Result != nil && strings.TrimSpace(session.Result.Message) != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", session.Result.Message)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %q deletion started (session %s).\n", label, name, session.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}
