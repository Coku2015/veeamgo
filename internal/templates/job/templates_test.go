package jobtemplates

import (
	"strings"
	"testing"
)

func TestSupportedJobTypesContainCoreTemplates(t *testing.T) {
	types := SupportedJobTypes()
	expected := []string{
		"VSphereBackup",
		"HyperVBackup",
		"CloudDirectorBackup",
		"VSphereReplica",
		"BackupCopy",
		"FileBackupCopy",
		"WindowsAgentBackup",
		"LinuxAgentBackup",
		"EntraIDTenantBackup",
		"EntraIDAuditLogBackup",
		"EntraIDTenantBackupCopy",
	}
	for _, want := range expected {
		if !contains(types, want) {
			t.Fatalf("expected %q in supported types, got %v", want, types)
		}
	}

	variants := VariantsFor("VSphereBackup")
	if !containsVariant(variants, VariantMinimal) || !containsVariant(variants, VariantFull) {
		t.Fatalf("vsphere variants missing minimal/full, got %v", variants)
	}
}

func TestRenderStripsComments(t *testing.T) {
	withComments, err := Render("VSphereBackup", VariantMinimal, true)
	if err != nil {
		t.Fatalf("Render with comments failed: %v", err)
	}
	if len(withComments) == 0 {
		t.Fatalf("Render with comments returned empty payload")
	}

	withoutComments, err := Render("VSphereBackup", VariantMinimal, false)
	if err != nil {
		t.Fatalf("Render without comments failed: %v", err)
	}
	for _, line := range splitLines(withoutComments) {
		if trimmed := trimSpace(line); trimmed != "" && trimmed[0] == '#' {
			t.Fatalf("expected no comment lines, found %q", line)
		}
	}
}

func TestSpecReturnsMap(t *testing.T) {
	spec, err := Spec("VSphereBackup", VariantMinimal)
	if err != nil {
		t.Fatalf("Spec failed: %v", err)
	}
	if spec["type"] != "VSphereBackup" {
		t.Fatalf("unexpected type field: %v", spec["type"])
	}
	storage, ok := spec["storage"].(map[string]any)
	if !ok {
		t.Fatalf("storage not decoded as map: %#v", spec["storage"])
	}
	if _, ok := storage["backupRepositoryId"]; !ok {
		t.Fatalf("backupRepositoryId missing in storage")
	}
}

func TestWindowsAgentTemplateHasComputers(t *testing.T) {
	spec, err := Spec("WindowsAgentBackup", VariantMinimal)
	if err != nil {
		t.Fatalf("Spec failed: %v", err)
	}
	if spec["type"] != "WindowsAgentBackup" {
		t.Fatalf("unexpected type %v", spec["type"])
	}
	computers, ok := spec["computers"].([]any)
	if !ok || len(computers) == 0 {
		t.Fatalf("computers list missing or empty: %#v", spec["computers"])
	}
	entry, ok := computers[0].(map[string]any)
	if !ok {
		t.Fatalf("computers entry not a map: %#v", computers[0])
	}
	if entry["type"] != "WindowsComputer" {
		t.Fatalf("unexpected computer type: %v", entry["type"])
	}
}

func TestFullTemplateDecodes(t *testing.T) {
	spec, err := Spec("BackupCopy", VariantFull)
	if err != nil {
		t.Fatalf("Spec full failed: %v", err)
	}
	if spec["mode"] != "Periodic" {
		t.Fatalf("unexpected mode: %v", spec["mode"])
	}
	target, ok := spec["target"].(map[string]any)
	if !ok {
		t.Fatalf("target not a map: %#v", spec["target"])
	}
	if _, ok := target["backupRepositoryId"]; !ok {
		t.Fatalf("backupRepositoryId missing in full template target")
	}
}

func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(string(data), "\n")
}

func trimSpace(value string) string {
	return strings.TrimSpace(value)
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func containsVariant(values []Variant, target Variant) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
