package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/veeamgo/veeamgo/internal/secrets"
)

// Config holds CLI configuration.
type Config struct {
	DefaultProfile string              `yaml:"default_profile"`
	Profiles       map[string]*Profile `yaml:"profiles"`
}

// Profile stores connection settings for a VBR server.
type Profile struct {
	ServerURL string `yaml:"server_url"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password,omitempty"`
	Insecure  bool   `yaml:"insecure"`
}

// Load reads the configuration from disk. If the file does not exist, a default config is returned.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Config{
				Profiles: make(map[string]*Profile),
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	for name, profile := range cfg.Profiles {
		if profile == nil || profile.Password == "" {
			continue
		}
		if strings.HasPrefix(profile.Password, "ENC:") {
			decrypted, err := secrets.DecryptString(path, strings.TrimPrefix(profile.Password, "ENC:"))
			if err != nil {
				return nil, fmt.Errorf("decrypt password for profile %s: %w", name, err)
			}
			profile.Password = decrypted
		}
	}
	return cfg, nil
}

// Save writes the configuration to disk with 0600 permissions.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	encoded, err := encodeSecrets(path, cfg)
	if err != nil {
		return err
	}

	payload, err := yaml.Marshal(encoded)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("commit config: %w", err)
	}
	return nil
}

// SetProfile adds or replaces the profile in the config.
func (c *Config) SetProfile(name string, profile Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	cp := profile
	c.Profiles[name] = &cp
}

// RemoveProfile deletes the profile from the config.
func (c *Config) RemoveProfile(name string) {
	delete(c.Profiles, name)
	if c.DefaultProfile == name {
		c.DefaultProfile = ""
	}
}

// GetProfile returns the profile or nil if missing.
func (c *Config) GetProfile(name string) *Profile {
	if c.Profiles == nil {
		return nil
	}
	return c.Profiles[name]
}

// ListProfiles returns the profile names.
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

func encodeSecrets(path string, cfg *Config) (*Config, error) {
	clone := &Config{
		DefaultProfile: cfg.DefaultProfile,
		Profiles:       make(map[string]*Profile, len(cfg.Profiles)),
	}

	for name, profile := range cfg.Profiles {
		if profile == nil {
			continue
		}
		cp := *profile
		if cp.Password != "" {
			enc, err := secrets.EncryptString(path, cp.Password)
			if err != nil {
				return nil, fmt.Errorf("encrypt password for profile %s: %w", name, err)
			}
			cp.Password = "ENC:" + enc
		}
		clone.Profiles[name] = &cp
	}

	if clone.Profiles == nil {
		clone.Profiles = make(map[string]*Profile)
	}

	return clone, nil
}
