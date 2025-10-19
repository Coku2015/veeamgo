package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// BaseDir returns the root directory for VeeamGo configuration (~/.veeamgo by default).
func BaseDir() (string, error) {
	if envHome := os.Getenv("VEEAMGO_HOME"); envHome != "" {
		dir := filepath.Clean(envHome)
		if err := ensureDir(dir, 0o700); err != nil {
			return "", err
		}
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	dir := filepath.Join(home, ".veeamgo")
	if err := ensureDir(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// ResolveConfigPath determines the configuration file path, migrating legacy paths when present.
func ResolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}

	if env := os.Getenv("VEEAMGO_CONFIG"); env != "" {
		return filepath.Clean(env), nil
	}

	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	primary := filepath.Join(base, "config")

	if exists(primary) {
		return primary, nil
	}

	legacyCandidates := legacyConfigCandidates()
	for _, candidate := range legacyCandidates {
		if exists(candidate) {
			return candidate, nil
		}
	}

	return primary, nil
}

func ConfigKeyPath(configPath string) string {
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, "secret.key")
}

func SessionsPath(baseDir string) string {
	return filepath.Join(baseDir, "sessions.json")
}

func ensureDir(path string, perm os.FileMode) error {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return os.MkdirAll(path, perm)
	case err != nil:
		return fmt.Errorf("stat %s: %w", path, err)
	case !info.IsDir():
		return fmt.Errorf("%s exists and is not a directory", path)
	default:
		_ = os.Chmod(path, perm) // best effort; ignore errors on read-only filesystems
		return nil
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func legacyConfigCandidates() []string {
	candidates := make([]string, 0)

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".veeamgo.config"),
			filepath.Join(home, ".config", "veeamgo", "config.yaml"),
		)
	}

	if cfgDir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(cfgDir, "veeamgo", "config.yaml"))
	}

	return candidates
}
