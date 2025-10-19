package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")

	cfg := &Config{
		DefaultProfile: "default",
	}
	cfg.SetProfile("default", Profile{
		ServerURL:  "https://example",
		Username:   "admin",
		Password:   "secret",
		Insecure:   true,
		APIVersion: "1.2-rev1",
	})

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected permissions 0600, got %v", info.Mode().Perm())
	}

	keyInfo, err := os.Stat(filepath.Join(filepath.Dir(path), "secret.key"))
	if err != nil {
		t.Fatalf("expected secret key to be created: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected secret key permissions 0600, got %v", keyInfo.Mode().Perm())
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("expected config file to hide plaintext password")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if loaded.DefaultProfile != "default" {
		t.Fatalf("expected default profile to be default, got %q", loaded.DefaultProfile)
	}
	profile := loaded.GetProfile("default")
	if profile == nil {
		t.Fatalf("expected profile to exist")
	}
	if profile.ServerURL != "https://example" {
		t.Fatalf("unexpected server URL: %s", profile.ServerURL)
	}
	if profile.Password != "secret" {
		t.Fatalf("unexpected password: %s", profile.Password)
	}
	if !profile.Insecure {
		t.Fatalf("expected insecure flag to be true")
	}
	if profile.APIVersion != "1.2-rev1" {
		t.Fatalf("unexpected api version: %s", profile.APIVersion)
	}
}
