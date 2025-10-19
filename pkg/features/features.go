package features

import (
	"fmt"
	"strings"

	"github.com/veeamgo/veeamgo/pkg/apiversion"
)

// Feature enumerates high-level CLI capabilities that may depend on REST revisions.
type Feature string

const (
	// FeatureScaleOutRepositories gates scale-out repository inventory and configuration.
	FeatureScaleOutRepositories Feature = "scale_out_repositories"
)

// Matrix tracks the minimum REST revision required for a feature.
type Matrix struct {
	minVersion map[Feature]string
	labels     map[Feature]string
}

// DefaultMatrix returns the CLI's built-in feature gate configuration.
func DefaultMatrix() Matrix {
	return Matrix{
		minVersion: map[Feature]string{
			FeatureScaleOutRepositories: "1.1-rev0",
		},
		labels: map[Feature]string{
			FeatureScaleOutRepositories: "Scale-out repositories",
		},
	}
}

// Supports reports whether the negotiated REST revision satisfies the feature requirement.
func (m Matrix) Supports(version string, feature Feature) bool {
	min, ok := m.minVersion[feature]
	if !ok {
		return true
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = apiversion.DefaultVersion
	}
	comp, err := apiversion.Compare(version, min)
	if err != nil {
		return false
	}
	return comp >= 0
}

// MinVersion returns the minimum revision for the feature, if known.
func (m Matrix) MinVersion(feature Feature) (string, bool) {
	min, ok := m.minVersion[feature]
	return min, ok
}

// Label returns a human friendly label for error messages.
func (m Matrix) Label(feature Feature) string {
	if label, ok := m.labels[feature]; ok {
		return label
	}
	return string(feature)
}

// UnsupportedError returns a formatted error describing the missing capability.
func (m Matrix) UnsupportedError(feature Feature, negotiated string) error {
	min, _ := m.MinVersion(feature)
	label := m.Label(feature)
	if min == "" {
		return fmt.Errorf("%s is not available for API version %s", label, negotiated)
	}
	return fmt.Errorf("%s is not supported by this server version (requires API %s or later, negotiated %s)", label, min, negotiated)
}
