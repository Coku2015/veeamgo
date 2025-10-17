package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectCacheDirUsesEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VEEAMGO_CACHE_DIR", tmp)

	dir, err := ProjectCacheDir()
	if err != nil {
		t.Fatalf("ProjectCacheDir returned error: %v", err)
	}
	if dir != tmp {
		t.Fatalf("expected cache dir %s, got %s", tmp, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat cache dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("cache dir is not a directory")
	}
}

func TestJobTemplatePathSanitize(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VEEAMGO_CACHE_DIR", tmp)

	path, err := JobTemplatePath(" Job ID 123 ")
	if err != nil {
		t.Fatalf("JobTemplatePath returned error: %v", err)
	}
	expected := filepath.Join(tmp, "jobs", "job-id-123.yaml")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("jobs directory not created: %v", err)
	}
}
