package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func mustSetFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("set flag %s: %v", name, err)
	}
}

func TestWasabiCloudConfiguratorBuildBase(t *testing.T) {
	provider := newWasabiCloudProvider()
	cfg := provider.newConfigurator()

	cmd := &cobra.Command{Use: "test"}
	cfg.BindFlags(cmd)

	mustSetFlag(t, cmd, "credentials-id", "550e8400-e29b-41d4-a716-446655440000")
	mustSetFlag(t, cmd, "region-id", "us-east-1")
	mustSetFlag(t, cmd, "bucket-name", "veeam-backups")
	mustSetFlag(t, cmd, "folder", "prod")
	mustSetFlag(t, cmd, "mount-server-id", "775c1d5f-3fa3-4b21-96a1-2c2c3c76a6e5")
	mustSetFlag(t, cmd, "mount-server-cache", `D:\Cache`)

	payload, err := cfg.BuildBase(cmd, &objectRepositoryAddOptions{})
	if err != nil {
		t.Fatalf("BuildBase returned error: %v", err)
	}

	if got := nestedString(payload, "type"); got != "WasabiCloud" {
		t.Fatalf("expected type WasabiCloud, got %q", got)
	}
	if got := nestedString(payload, "account", "credentialsId"); got == "" {
		t.Fatalf("expected credentialsId to be set")
	}
	if got := nestedString(payload, "bucket", "bucketName"); got != "veeam-backups" {
		t.Fatalf("expected bucketName veeam-backups, got %q", got)
	}
	if got := nestedString(payload, "bucket", "folderName"); got != "prod" {
		t.Fatalf("expected folderName prod, got %q", got)
	}
	if got := nestedString(payload, "mountServer", "windows", "mountServerId"); got == "" {
		t.Fatalf("expected mount server ID to be set")
	}
}

func TestWasabiCloudConfiguratorValidation(t *testing.T) {
	provider := newWasabiCloudProvider()
	cfg := provider.newConfigurator()

	cmd := &cobra.Command{Use: "test"}
	cfg.BindFlags(cmd)

	// Intentionally omit bucket-name to trigger validation failure.
	mustSetFlag(t, cmd, "credentials-id", "550e8400-e29b-41d4-a716-446655440000")
	mustSetFlag(t, cmd, "region-id", "us-east-1")
	mustSetFlag(t, cmd, "folder", "prod")
	mustSetFlag(t, cmd, "mount-server-id", "775c1d5f-3fa3-4b21-96a1-2c2c3c76a6e5")
	mustSetFlag(t, cmd, "mount-server-cache", `D:\Cache`)

	if _, err := cfg.BuildBase(cmd, &objectRepositoryAddOptions{}); err == nil {
		t.Fatalf("expected BuildBase to fail when bucket-name is missing")
	}
}

func TestAzureArchiveConfiguratorBuildBase(t *testing.T) {
	provider := newAzureArchiveProvider()
	cfg := provider.newConfigurator()

	cmd := &cobra.Command{Use: "test"}
	cfg.BindFlags(cmd)

	mustSetFlag(t, cmd, "credentials-id", "7aa17809-0e52-4f23-992f-3a7420530c21")
	mustSetFlag(t, cmd, "container-name", "archive-container")
	mustSetFlag(t, cmd, "folder", "vault")
	mustSetFlag(t, cmd, "proxy-subscription-id", "514c0221-5a0e-4b43-b49a-946f2a6d4872")
	mustSetFlag(t, cmd, "mount-server-id", "9ce5bfa1-7f73-4102-9641-66980b9e8d7f")
	mustSetFlag(t, cmd, "mount-server-cache", `D:\Cache`)

	payload, err := cfg.BuildBase(cmd, &objectRepositoryAddOptions{})
	if err != nil {
		t.Fatalf("BuildBase returned error: %v", err)
	}

	if got := nestedString(payload, "type"); got != "AzureArchive" {
		t.Fatalf("expected type AzureArchive, got %q", got)
	}
	if got := nestedString(payload, "account", "credentialsId"); got == "" {
		t.Fatalf("expected credentialsId to be set")
	}
	if got := nestedString(payload, "container", "containerName"); got != "archive-container" {
		t.Fatalf("expected containerName archive-container, got %q", got)
	}
	if got := nestedString(payload, "proxyAppliance", "subscriptionId"); got == "" {
		t.Fatalf("expected proxy subscription ID to be set")
	}
	if got := nestedString(payload, "mountServer", "windows", "mountServerId"); got == "" {
		t.Fatalf("expected mount server ID to be set")
	}
}
