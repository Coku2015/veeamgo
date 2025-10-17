package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
)

var handshakeCodePattern = regexp.MustCompile(`^\d{6}$`)

func managedServerAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdManagedServerUse,
		Short: "Add managed servers",
	}

	cmd.AddCommand(managedServerAddVsphereCmd())
	cmd.AddCommand(managedServerAddWindowsCmd())
	cmd.AddCommand(managedServerAddLinuxCmd())

	return cmd
}

func managedServerAddVsphereCmd() *cobra.Command {
	opts := struct {
		name        string
		description string
		port        int
		thumbprint  string
		wait        bool
		yes         bool
		credentials managedServerCredentialsOptions
	}{
		credentials: managedServerCredentialsOptions{
			credType: "Standard",
		},
	}

	cmd := &cobra.Command{
		Use:   "vsphere",
		Short: "Add a VMware vSphere managed server",
		Long:  "Registers a VMware vSphere server (vCenter or ESXi) and optionally provisions credentials automatically.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = strings.TrimSpace(opts.name)
			if opts.name == "" {
				return requireFlag("--name", "provide the vSphere server DNS name or IP address")
			}
			if opts.port < 0 || opts.port > 65535 {
				return fmt.Errorf("port must be between 0 and 65535")
			}

			description := strings.TrimSpace(opts.description)
			if description == "" {
				description = autoManagedServerDescription(opts.name, "ViHost")
			}

			serverLabel := managedServerTypeDisplay("ViHost")
			if serverLabel == "" {
				serverLabel = "Managed server"
			}

			if !opts.yes {
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %q", serverLabel, opts.name))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			credentialsID, created, err := opts.credentials.resolve(ctx, cmd, httpClient, opts.name, "ViHost")
			if err != nil {
				return err
			}

			payload := map[string]any{
				"name":          opts.name,
				"description":   description,
				"type":          "ViHost",
				"credentialsId": credentialsID,
			}
			if opts.port > 0 {
				payload["port"] = opts.port
			}
			if thumb := strings.TrimSpace(opts.thumbprint); thumb != "" {
				payload["certificateThumbprint"] = thumb
			}

			session, err := httpClient.CreateManagedServer(ctx, payload)
			if err != nil {
				if created {
					fmt.Fprintf(cmd.ErrOrStderr(), "Managed server creation failed; credentials %s were created and remain available.\n", credentialsID)
				}
				return fmt.Errorf("create managed server: %w", err)
			}

			if opts.wait {
				final, err := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
				if err != nil {
					return fmt.Errorf("wait for provisioning session: %w", err)
				}
				session = final
			}

			printManagedServerSession(cmd, session, opts.name, "ViHost", opts.wait)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "vSphere server DNS name or IP address")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description for the managed server (auto-generated if omitted)")
	cmd.Flags().IntVar(&opts.port, "port", 0, "Port used to communicate with the vSphere server (default 443)")
	cmd.Flags().StringVar(&opts.thumbprint, "thumbprint", "", "TLS thumbprint used to validate the server identity")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to complete")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	cmd.Flags().StringVar(&opts.credentials.credentialsID, "credentials-id", "", "Existing credentials ID to reuse")
	cmd.Flags().StringVar(&opts.credentials.username, "username", "", "Username for creating credentials (mutually exclusive with --credentials-id)")
	cmd.Flags().StringVar(&opts.credentials.password, "password", "", "Password for creating credentials (omit to prompt securely)")
	cmd.Flags().StringVar(&opts.credentials.description, "credential-description", "", "Description for created credentials (auto-generated if omitted)")

	return cmd
}

func managedServerAddWindowsCmd() *cobra.Command {
	opts := struct {
		name        string
		description string
		wait        bool
		yes         bool
		connectMode string
		credentials managedServerCredentialsOptions
	}{
		connectMode: "Credential",
		credentials: managedServerCredentialsOptions{
			credType: "Standard",
		},
	}

	cmd := &cobra.Command{
		Use:   "windows",
		Short: "Add a Microsoft Windows managed server",
		Long:  "Registers a Windows server and can create standard credentials automatically or rely on certificate-based authentication when the deployment kit is installed.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = strings.TrimSpace(opts.name)
			if opts.name == "" {
				return requireFlag("--name", "provide the Windows server DNS name or IP address")
			}

			storageInput := strings.TrimSpace(opts.connectMode)
			switch {
			case storageInput == "":
				opts.connectMode = "Credential"
			case strings.EqualFold(storageInput, "Credential"):
				opts.connectMode = "Credential"
			case strings.EqualFold(storageInput, "Permanent"):
				opts.connectMode = "Credential"
			case strings.EqualFold(storageInput, "Certificate"):
				opts.connectMode = "Certificate"
			default:
				return fmt.Errorf("connect mode %q is not supported; use Credential or Certificate", storageInput)
			}
			apiStorage := "Permanent"
			if opts.connectMode == "Certificate" {
				apiStorage = "Certificate"
			}

			description := strings.TrimSpace(opts.description)
			if description == "" {
				description = autoManagedServerDescription(opts.name, "WindowsHost")
			}

			serverLabel := managedServerTypeDisplay("WindowsHost")
			if serverLabel == "" {
				serverLabel = "Windows server"
			}

			if !opts.yes {
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %q", serverLabel, opts.name))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			var (
				credentialsID string
				created       bool
			)
			if opts.connectMode == "Certificate" {
				if opts.credentials.credentialsID != "" || opts.credentials.username != "" || opts.credentials.password != "" || opts.credentials.description != "" {
					return fmt.Errorf("credential flags cannot be combined with --connect-mode Certificate")
				}
			} else {
				var err error
				credentialsID, created, err = opts.credentials.resolve(ctx, cmd, httpClient, opts.name, "WindowsHost")
				if err != nil {
					return err
				}
			}

			payload := map[string]any{
				"name":                   opts.name,
				"description":            description,
				"type":                   "WindowsHost",
				"credentialsStorageType": apiStorage,
			}
			if credentialsID != "" {
				payload["credentialsId"] = credentialsID
			}

			session, err := httpClient.CreateManagedServer(ctx, payload)
			if err != nil {
				if created && credentialsID != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Managed server creation failed; credentials %s were created and remain available.\n", credentialsID)
				}
				return fmt.Errorf("create managed server: %w", err)
			}

			if opts.wait {
				final, err := httpClient.WaitForSession(ctx, session.ID, 5*time.Second)
				if err != nil {
					return fmt.Errorf("wait for provisioning session: %w", err)
				}
				session = final
			}

			printManagedServerSession(cmd, session, opts.name, "WindowsHost", opts.wait)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Windows server DNS name or IP address")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description for the managed server (auto-generated if omitted)")
	cmd.Flags().StringVar(&opts.connectMode, "connect-mode", "Credential", "Connection mode (Credential or Certificate)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to complete")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	cmd.Flags().StringVar(&opts.credentials.credentialsID, "credentials-id", "", "Existing credentials ID to reuse")
	cmd.Flags().StringVar(&opts.credentials.username, "username", "", "Username for creating credentials (mutually exclusive with --credentials-id)")
	cmd.Flags().StringVar(&opts.credentials.password, "password", "", "Password for creating credentials (omit to prompt securely)")
	cmd.Flags().StringVar(&opts.credentials.description, "credential-description", "", "Description for created credentials (auto-generated if omitted)")

	return cmd
}

type managedServerCredentialsOptions struct {
	credentialsID string
	username      string
	password      string
	description   string
	credType      string
}

func (opts managedServerCredentialsOptions) hasAnyValues() bool {
	return strings.TrimSpace(opts.credentialsID) != "" ||
		strings.TrimSpace(opts.username) != "" ||
		strings.TrimSpace(opts.password) != "" ||
		strings.TrimSpace(opts.description) != ""
}

func (opts *managedServerCredentialsOptions) resolve(ctx context.Context, cmd *cobra.Command, httpClient *client.Client, serverName, serverType string) (string, bool, error) {
	if opts == nil {
		return "", false, fmt.Errorf("credentials options cannot be nil")
	}

	id := strings.TrimSpace(opts.credentialsID)
	if id != "" {
		if opts.username != "" || opts.password != "" || opts.description != "" {
			return "", false, fmt.Errorf("provide either --credentials-id or the username/password flags, not both")
		}
		return id, false, nil
	}

	username := strings.TrimSpace(opts.username)
	if username == "" {
		return "", false, requireFlag("--username", "supply credentials username or use --credentials-id")
	}

	password := opts.password
	if strings.TrimSpace(password) == "" {
		secret, err := promptPassword("Password: ")
		if err != nil {
			return "", false, err
		}
		password = secret
	}
	if password == "" {
		return "", false, fmt.Errorf("password cannot be empty")
	}

	credDescription := strings.TrimSpace(opts.description)
	if credDescription == "" {
		credDescription = autoCredentialDescription(serverName, username, serverType)
	}

	spec := map[string]any{
		"username":    username,
		"description": credDescription,
		"type":        opts.credType,
	}
	switch opts.credType {
	case "Standard":
		spec["password"] = password
	default:
		return "", false, fmt.Errorf("credential type %q is not supported in this command", opts.credType)
	}

	created, err := httpClient.CreateCredentials(ctx, spec)
	if err != nil {
		return "", false, fmt.Errorf("create credentials: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created credentials %q (%s).\n", created.Description, created.ID)
	return created.ID, true, nil
}

func autoManagedServerDescription(name, serverType string) string {
	label := managedServerTypeDisplay(serverType)
	if strings.TrimSpace(label) == "" {
		label = "Managed server"
	}
	return fmt.Sprintf("%s %s (added via veeamgo on %s)", label, name, time.Now().UTC().Format("2006-01-02"))
}

func autoCredentialDescription(serverName, username, serverType string) string {
	label := managedServerTypeDisplay(serverType)
	if strings.TrimSpace(label) == "" {
		label = "Managed server"
	}
	return fmt.Sprintf("%s credentials for %s (%s, created %s)", label, serverName, username, time.Now().UTC().Format("2006-01-02"))
}

func printManagedServerSession(cmd *cobra.Command, session *client.Session, serverName, serverType string, waited bool) {
	label := managedServerTypeDisplay(serverType)
	if label == "" {
		label = "Managed server"
	}

	if session == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %q provisioning completed.\n", label, serverName)
		return
	}

	if waited {
		result := ""
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		if result == "" {
			result = session.State
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %q provisioning finished with result %s (session %s).\n", label, serverName, result, session.ID)
		if session.ResourceID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Managed server ID: %s\n", session.ResourceID)
		}
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Started provisioning %s %q (session %s).\n", label, serverName, session.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
}

func managedServerAddLinuxCmd() *cobra.Command {
	opts := struct {
		name           string
		description    string
		wait           bool
		yes            bool
		connectMode    string
		sshFingerprint string
		handshakeCode  string
		credentials    managedServerCredentialsOptions
		singleUse      linuxSingleUseOptions
	}{
		connectMode: "Credential",
		credentials: managedServerCredentialsOptions{
			credType: "Standard",
		},
	}

	cmd := &cobra.Command{
		Use:   "linux",
		Short: "Add a Linux managed server",
		Long:  "Registers a Linux managed server. Supports permanent credentials, single-use SSH credentials, or certificate-based pairing when the deployment kit is installed.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = strings.TrimSpace(opts.name)
			if opts.name == "" {
				return requireFlag("--name", "provide the Linux server DNS name or IP address")
			}
			modeInput := strings.TrimSpace(opts.connectMode)
			switch {
			case modeInput == "":
				opts.connectMode = "Credential"
			case strings.EqualFold(modeInput, "Credential"):
				opts.connectMode = "Credential"
			case strings.EqualFold(modeInput, "Permanent"):
				opts.connectMode = "Credential"
			case strings.EqualFold(modeInput, "SingleUse"):
				opts.connectMode = "SingleUse"
			case strings.EqualFold(modeInput, "Certificate"):
				opts.connectMode = "Certificate"
			default:
				return fmt.Errorf("connect mode %q is not supported; use Credential, SingleUse, or Certificate", modeInput)
			}

			description := strings.TrimSpace(opts.description)
			if description == "" {
				description = autoManagedServerDescription(opts.name, "LinuxHost")
			}

			serverLabel := managedServerTypeDisplay("LinuxHost")
			if serverLabel == "" {
				serverLabel = "Linux server"
			}

			if !opts.yes {
				confirmed, err := promptForConfirmation(cmd, fmt.Sprintf("Add %s %q", serverLabel, opts.name))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			timeout := 5 * time.Minute
			if opts.wait {
				timeout = 15 * time.Minute
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			opts.sshFingerprint = strings.TrimSpace(opts.sshFingerprint)
			opts.handshakeCode = strings.TrimSpace(opts.handshakeCode)
			handshakeRequired := false

			if opts.connectMode == "Certificate" {
				if opts.sshFingerprint == "" {
					autoCtx, autoCancel := context.WithTimeout(ctx, 30*time.Second)
					defer autoCancel()

					cert, err := httpClient.ConnectionCertificate(autoCtx, client.ConnectionCertificateRequest{
						ServerName:             opts.name,
						Type:                   "LinuxHost",
						CredentialsStorageType: linuxConnectModeToAPI("Credential"),
					})
					if err != nil {
						lower := strings.ToLower(err.Error())
						if strings.Contains(lower, "handshake") || strings.Contains(lower, "pairing") || strings.Contains(lower, "credential") {
							handshakeRequired = true
						} else {
							fmt.Fprintf(cmd.OutOrStderr(), "Fingerprint auto-fetch unavailable: %v\n", err)
						}
					} else {
						fingerprint := strings.TrimSpace(cert.Fingerprint)
						if fingerprint != "" {
							if !opts.yes {
								confirm, confirmErr := promptForConfirmation(cmd, fmt.Sprintf("Accept SSH fingerprint %q for %s", fingerprint, opts.name))
								if confirmErr != nil {
									return confirmErr
								}
								if !confirm {
									return fmt.Errorf("fingerprint rejected; aborting managed server creation")
								}
							}
							fmt.Fprintf(cmd.OutOrStdout(), "Accepted SSH fingerprint %q for %s.\n", fingerprint, opts.name)
							opts.sshFingerprint = fingerprint
						}
					}
				}
			} else {
				if opts.sshFingerprint == "" {
					if opts.connectMode == "Credential" {
						autoCtx, autoCancel := context.WithTimeout(ctx, 30*time.Second)
						defer autoCancel()

						connectionMode := linuxConnectModeToAPI(opts.connectMode)
						cert, err := httpClient.ConnectionCertificate(autoCtx, client.ConnectionCertificateRequest{
							ServerName:             opts.name,
							Type:                   "LinuxHost",
							CredentialsStorageType: connectionMode,
						})
						if err != nil {
							msg := err.Error()
							if strings.Contains(msg, "SingleUseCredentials") || strings.Contains(msg, "CredentialsStorageType") {
								return fmt.Errorf("retrieve SSH fingerprint: %w (provide --ssh-fingerprint or required credential flags to bypass auto-fetch)", err)
							}
							return fmt.Errorf("retrieve SSH fingerprint: %w (specify --ssh-fingerprint to bypass auto-fetch)", err)
						}
						fingerprint := strings.TrimSpace(cert.Fingerprint)
						if fingerprint == "" {
							return fmt.Errorf("retrieved SSH fingerprint is empty; specify --ssh-fingerprint manually")
						}

						if !opts.yes {
							confirm, err := promptForConfirmation(cmd, fmt.Sprintf("Accept SSH fingerprint %q for %s", fingerprint, opts.name))
							if err != nil {
								return err
							}
							if !confirm {
								return fmt.Errorf("fingerprint rejected; aborting managed server creation")
							}
						}
						fmt.Fprintf(cmd.OutOrStdout(), "Accepted SSH fingerprint %q for %s.\n", fingerprint, opts.name)
						opts.sshFingerprint = fingerprint
					} else {
						return requireFlag("--ssh-fingerprint", "provide the SSH fingerprint obtained from the server")
					}
				}
			}

			if handshakeRequired && opts.handshakeCode == "" {
				code, err := promptForInput(cmd, "Handshake code (6 digits): ")
				if err != nil {
					return err
				}
				opts.handshakeCode = strings.TrimSpace(code)
			}
			if handshakeRequired {
				if !handshakeCodePattern.MatchString(opts.handshakeCode) {
					return fmt.Errorf("invalid handshake code; expected 6 digits")
				}
				if opts.sshFingerprint == "" {
					handshakeCtx, handshakeCancel := context.WithTimeout(ctx, 30*time.Second)
					cert, err := httpClient.ConnectionCertificate(handshakeCtx, client.ConnectionCertificateRequest{
						ServerName:             opts.name,
						Type:                   "LinuxHost",
						CredentialsStorageType: linuxConnectModeToAPI(opts.connectMode),
						HandshakeCode:          opts.handshakeCode,
					})
					handshakeCancel()
					if err == nil {
						if fingerprint := strings.TrimSpace(cert.Fingerprint); fingerprint != "" {
							fmt.Fprintf(cmd.OutOrStdout(), "Accepted SSH fingerprint %q for %s via handshake.\n", fingerprint, opts.name)
							opts.sshFingerprint = fingerprint
						}
					}
					if opts.sshFingerprint == "" {
						if err != nil {
							fmt.Fprintf(cmd.OutOrStderr(), "Unable to retrieve fingerprint with handshake code: %v\n", err)
						}
						if opts.yes {
							return requireFlag("--ssh-fingerprint", "provide the SSH fingerprint obtained from the server")
						}
						manual, inputErr := promptForInput(cmd, "SSH fingerprint: ")
						if inputErr != nil {
							return inputErr
						}
						opts.sshFingerprint = strings.TrimSpace(manual)
						if opts.sshFingerprint == "" {
							return requireFlag("--ssh-fingerprint", "provide the SSH fingerprint obtained from the server")
						}
					}
				}
			}

			var (
				credentialsID string
				created       bool
				singleUseSpec map[string]any
			)

			switch opts.connectMode {
			case "Certificate":
				if opts.credentials.hasAnyValues() {
					return fmt.Errorf("credential flags cannot be combined with --connect-mode Certificate")
				}
				if opts.singleUse.hasAnyValues() {
					return fmt.Errorf("single-use flags cannot be combined with --connect-mode Certificate")
				}
			case "Credential":
				if opts.singleUse.hasAnyValues() {
					return fmt.Errorf("single-use flags cannot be combined with --connect-mode Credential")
				}
				var resolveErr error
				credentialsID, created, resolveErr = opts.credentials.resolve(ctx, cmd, httpClient, opts.name, "LinuxHost")
				if resolveErr != nil {
					return resolveErr
				}
			case "SingleUse":
				if opts.credentials.hasAnyValues() {
					return fmt.Errorf("standard credential flags cannot be combined with --connect-mode SingleUse")
				}
				var buildErr error
				singleUseSpec, buildErr = opts.singleUse.build()
				if buildErr != nil {
					return buildErr
				}
			}

			payload := map[string]any{
				"name":                   opts.name,
				"description":            description,
				"type":                   "LinuxHost",
				"credentialsStorageType": linuxConnectModeToAPI(opts.connectMode),
			}
			if opts.sshFingerprint != "" {
				payload["sshFingerprint"] = opts.sshFingerprint
			}
			if credentialsID != "" {
				payload["credentialsId"] = credentialsID
			}
			if opts.connectMode == "SingleUse" && singleUseSpec != nil {
				payload["singleUseCredentials"] = singleUseSpec
			}
			if strings.TrimSpace(opts.handshakeCode) != "" {
				payload["handshakeCode"] = strings.TrimSpace(opts.handshakeCode)
			}

			session, err := httpClient.CreateManagedServer(ctx, payload)
			if err != nil {
				if created && credentialsID != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Managed server creation failed; credentials %s were created and remain available.\n", credentialsID)
				}
				return fmt.Errorf("create managed server: %w", err)
			}

			if opts.wait {
				waitCtx, waitCancel := context.WithTimeout(ctx, timeout)
				defer waitCancel()
				finalSession, err := httpClient.WaitForSession(waitCtx, session.ID, 5*time.Second)
				if err != nil {
					return fmt.Errorf("wait for provisioning session: %w", err)
				}
				session = finalSession
			}

			printManagedServerSession(cmd, session, opts.name, "LinuxHost", opts.wait)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Linux server DNS name or IP address")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description for the managed server (auto-generated if omitted)")
	cmd.Flags().StringVar(&opts.connectMode, "connect-mode", "Credential", "Connection mode (Credential, SingleUse, or Certificate)")
	cmd.Flags().StringVar(&opts.sshFingerprint, "ssh-fingerprint", "", "SSH fingerprint used to verify the server identity")
	cmd.Flags().StringVar(&opts.handshakeCode, "handshake-code", "", "Handshake code for certificate-based pairing (optional)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to complete")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required for non-interactive use)")

	cmd.Flags().StringVar(&opts.credentials.credentialsID, "credentials-id", "", "Existing credentials ID to reuse (Credential mode)")
	cmd.Flags().StringVar(&opts.credentials.username, "username", "", "Username for creating credentials (Credential mode)")
	cmd.Flags().StringVar(&opts.credentials.password, "password", "", "Password for creating credentials (omit to prompt securely)")
	cmd.Flags().StringVar(&opts.credentials.description, "credential-description", "", "Description for created credentials (auto-generated if omitted)")

	cmd.Flags().StringVar(&opts.singleUse.username, "single-use-username", "", "Username for single-use credentials (SingleUse mode)")
	cmd.Flags().StringVar(&opts.singleUse.password, "single-use-password", "", "Password for single-use credentials")
	cmd.Flags().StringVar(&opts.singleUse.privateKey, "single-use-private-key", "", "Private key for single-use credentials (PEM)")
	cmd.Flags().StringVar(&opts.singleUse.passphrase, "single-use-passphrase", "", "Passphrase protecting the private key")
	cmd.Flags().StringVar(&opts.singleUse.rootPassword, "single-use-root-password", "", "Root password used when elevating privileges")
	cmd.Flags().StringVar(&opts.singleUse.authType, "single-use-auth-type", "", "Authentication type for single-use credentials (Password, PrivateKey, or Auto)")
	cmd.Flags().IntVar(&opts.singleUse.sshPort, "single-use-ssh-port", 0, "SSH port for single-use credentials (default 22)")
	cmd.Flags().BoolVar(&opts.singleUse.elevateToRoot, "single-use-elevate", false, "Elevate permissions to root for single-use credentials")
	cmd.Flags().BoolVar(&opts.singleUse.addToSudoers, "single-use-add-sudo", false, "Automatically add account to sudoers in single-use mode")
	cmd.Flags().BoolVar(&opts.singleUse.useSu, "single-use-use-su", false, "Use su instead of sudo in single-use mode")

	return cmd
}

func linuxConnectModeToAPI(connectMode string) string {
	switch connectMode {
	case "Credential":
		return "Permanent"
	case "SingleUse":
		return "SingleUse"
	case "Certificate":
		return "Certificate"
	default:
		return connectMode
	}
}

type linuxSingleUseOptions struct {
	username      string
	password      string
	privateKey    string
	passphrase    string
	rootPassword  string
	authType      string
	sshPort       int
	elevateToRoot bool
	addToSudoers  bool
	useSu         bool
}

func (opts linuxSingleUseOptions) hasAnyValues() bool {
	return strings.TrimSpace(opts.username) != "" ||
		strings.TrimSpace(opts.password) != "" ||
		strings.TrimSpace(opts.privateKey) != "" ||
		strings.TrimSpace(opts.passphrase) != "" ||
		strings.TrimSpace(opts.rootPassword) != "" ||
		strings.TrimSpace(opts.authType) != "" ||
		opts.sshPort != 0 ||
		opts.elevateToRoot ||
		opts.addToSudoers ||
		opts.useSu
}

func (opts linuxSingleUseOptions) build() (map[string]any, error) {
	username := strings.TrimSpace(opts.username)
	if username == "" {
		return nil, requireFlag("--single-use-username", "supply username in SingleUse mode")
	}

	authType := strings.TrimSpace(opts.authType)
	if authType == "" {
		if strings.TrimSpace(opts.privateKey) != "" {
			authType = "PrivateKey"
		} else {
			authType = "Password"
		}
	}

	payload := map[string]any{
		"username":           username,
		"authenticationType": authType,
	}

	if strings.TrimSpace(opts.password) != "" {
		payload["password"] = opts.password
	}
	if strings.TrimSpace(opts.privateKey) != "" {
		payload["privateKey"] = opts.privateKey
	}
	if strings.TrimSpace(opts.passphrase) != "" {
		payload["passphrase"] = opts.passphrase
	}
	if strings.TrimSpace(opts.rootPassword) != "" {
		payload["rootPassword"] = opts.rootPassword
	}
	if opts.sshPort > 0 {
		payload["SSHPort"] = opts.sshPort
	}
	if opts.elevateToRoot {
		payload["elevateToRoot"] = true
	}
	if opts.addToSudoers {
		payload["addToSudoers"] = true
	}
	if opts.useSu {
		payload["useSu"] = true
	}

	if _, hasPassword := payload["password"]; !hasPassword && strings.EqualFold(authType, "Password") {
		return nil, fmt.Errorf("single-use password is required when authentication type is Password")
	}
	if _, hasKey := payload["privateKey"]; !hasKey && strings.EqualFold(authType, "PrivateKey") {
		return nil, fmt.Errorf("single-use private key is required when authentication type is PrivateKey")
	}

	return payload, nil
}
