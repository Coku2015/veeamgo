package features

import "testing"

func TestMatrixSupports(t *testing.T) {
	matrix := DefaultMatrix()
	if !matrix.Supports("1.3-rev0", FeatureScaleOutRepositories) {
		t.Fatalf("expected latest version to support scale-out repositories")
	}

	if matrix.Supports("1.0-rev2", FeatureScaleOutRepositories) {
		t.Fatalf("expected 1.0-rev2 to be gated")
	}

	if err := matrix.UnsupportedError(FeatureScaleOutRepositories, "1.0-rev2"); err == nil {
		t.Fatalf("expected UnsupportedError to include context")
	}
}
