package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func fingerprintGetCmd() *cobra.Command {
	var opts struct {
		serverName         string
		serverType         string
		credentialsStorage string
		credentialsHint    string
		port               int
	}

	cmd := &cobra.Command{
		Use:   "fingerprint",
		Short: "Request a managed server certificate or SSH fingerprint",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.serverName = strings.TrimSpace(opts.serverName)
			if opts.serverName == "" {
				return requireFlag("--server", "provide the host DNS name or IP address")
			}

			opts.serverType = strings.TrimSpace(opts.serverType)
			if opts.serverType == "" {
				return requireFlag("--type", "provide the managed server type (e.g. LinuxHost, WindowsHost, ViHost)")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			req := client.ConnectionCertificateRequest{
				ServerName: opts.serverName,
				Type:       opts.serverType,
			}
			if storage := strings.TrimSpace(opts.credentialsStorage); storage != "" {
				req.CredentialsStorageType = storage
			}
			if hint := strings.TrimSpace(opts.credentialsHint); hint != "" {
				req.HandshakeCode = hint
			}
			if opts.port > 0 {
				req.Port = opts.port
			}

			result, err := httpClient.ConnectionCertificate(ctx, req)
			if err != nil {
				return fmt.Errorf("retrieve fingerprint: %w", err)
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result)
			}

			rows := connectionCertificateRows(result)
			if len(rows) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No fingerprint or certificate returned.")
				return nil
			}
			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&opts.serverName, "server", "", "Host DNS name or IP address to query")
	cmd.Flags().StringVar(&opts.serverType, "type", "", "Managed server type (e.g. LinuxHost, WindowsHost)")
	cmd.Flags().StringVar(&opts.credentialsStorage, "credentials-storage", "", "Credentials storage type hint (Permanent, SingleUse, Certificate)")
	cmd.Flags().StringVar(&opts.credentialsHint, "credentials", "", "Handshake code or credential identifier (if required by the server type)")
	cmd.Flags().StringVar(&opts.credentialsHint, "handshake-code", "", "Alias of --credentials")
	cmd.Flags().IntVar(&opts.port, "port", 0, "Override port when requesting the certificate or fingerprint")

	return cmd
}

type connectionCertificateRow struct {
	Property string `json:"Property"`
	Value    string `json:"Value"`
}

func connectionCertificateRows(model *client.ConnectionCertificateModel) []connectionCertificateRow {
	if model == nil {
		return nil
	}

	rows := make([]connectionCertificateRow, 0, 2)
	if fingerprint := strings.TrimSpace(model.Fingerprint); fingerprint != "" {
		rows = append(rows, connectionCertificateRow{
			Property: "Fingerprint",
			Value:    fingerprint,
		})
	}
	if summary := formatCertificateSummary(model.Certificate); summary != "" {
		rows = append(rows, connectionCertificateRow{
			Property: "Certificate",
			Value:    summary,
		})
	}
	return rows
}

func formatCertificateSummary(certificate map[string]any) string {
	if len(certificate) == 0 {
		return ""
	}

	priorityKeys := []string{
		"commonName",
		"subject",
		"issuer",
		"thumbprint",
		"notAfter",
	}
	parts := make([]string, 0, len(priorityKeys))
	for _, key := range priorityKeys {
		if value, ok := certificate[key]; ok {
			if str := strings.TrimSpace(fmt.Sprint(value)); str != "" {
				parts = append(parts, fmt.Sprintf("%s=%s", key, str))
			}
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}

	data, err := json.Marshal(certificate)
	if err != nil {
		return fmt.Sprint(certificate)
	}
	return string(data)
}
