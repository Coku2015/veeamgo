package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectCacheDir returns a writable cache location scoped to the current project.
func ProjectCacheDir() (string, error) {
	if env := os.Getenv("VEEAMGO_CACHE_DIR"); strings.TrimSpace(env) != "" {
		dir := filepath.Clean(env)
		if err := ensureDir(dir, 0o700); err != nil {
			return "", err
		}
		return dir, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	dir := filepath.Join(cwd, ".veeamgo", "cache")
	if err := ensureDir(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// JobTemplatePath returns the default path for storing a job blueprint template.
func JobTemplatePath(jobID string) (string, error) {
	cacheDir, err := ProjectCacheDir()
	if err != nil {
		return "", err
	}

	jobsDir := filepath.Join(cacheDir, "jobs")
	if err := ensureDir(jobsDir, 0o700); err != nil {
		return "", err
	}

	name := strings.TrimSpace(jobID)
	if name == "" {
		name = "job"
	}

	return filepath.Join(jobsDir, sanitizeFilename(name)+".yaml"), nil
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "job"
	}

	builder := strings.Builder{}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + 32)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		case r == ' ':
			builder.WriteRune('-')
		default:
			builder.WriteRune('-')
		}
	}

	slug := strings.Trim(builder.String(), "-.")
	if slug == "" {
		return "job"
	}
	return slug
}
