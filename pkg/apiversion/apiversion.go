package apiversion

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// HeaderName is the HTTP header used by Veeam REST API to negotiate the revision.
const HeaderName = "x-api-version"

// DefaultVersion is the highest API revision bundled with this CLI.
const DefaultVersion = "1.3-rev0"

var (
	versionPattern = regexp.MustCompile(`^(\d+)\.(\d+)-rev(\d+)$`)

	supportedVersions = []string{
		"1.3-rev0",
		"1.2-rev1",
		"1.2-rev0",
		"1.1-rev2",
		"1.1-rev1",
		"1.1-rev0",
		"1.0-rev2",
		"1.0-rev1",
	}

	buildThresholds = []struct {
		min Build
		ver string
	}{
		{min: Build{Major: 13, Minor: 0, Patch: 0}, ver: "1.3-rev0"},
		{min: Build{Major: 12, Minor: 3, Patch: 1}, ver: "1.2-rev1"},
		{min: Build{Major: 12, Minor: 3, Patch: 0}, ver: "1.2-rev0"},
		{min: Build{Major: 12, Minor: 2, Patch: 0}, ver: "1.1-rev2"},
		{min: Build{Major: 12, Minor: 1, Patch: 0}, ver: "1.1-rev1"},
		{min: Build{Major: 12, Minor: 0, Patch: 0}, ver: "1.1-rev0"},
		{min: Build{Major: 11, Minor: 0, Patch: 1}, ver: "1.0-rev2"},
		{min: Build{Major: 11, Minor: 0, Patch: 0}, ver: "1.0-rev1"},
	}
)

// Version holds the parsed semantic parts of a REST revision (for example 1.3-rev0).
type Version struct {
	Major    int
	Minor    int
	Revision int
}

// Parse converts a REST revision string to a Version.
func Parse(value string) (Version, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Version{}, fmt.Errorf("api version is empty")
	}
	matches := versionPattern.FindStringSubmatch(value)
	if len(matches) != 4 {
		return Version{}, fmt.Errorf("invalid api version %q", value)
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	rev, _ := strconv.Atoi(matches[3])
	return Version{Major: major, Minor: minor, Revision: rev}, nil
}

// Compare reports how a relates to b. Returns -1 when a<b, 0 when a==b, 1 when a>b.
func Compare(a, b string) (int, error) {
	left, err := Parse(a)
	if err != nil {
		return 0, err
	}
	right, err := Parse(b)
	if err != nil {
		return 0, err
	}
	switch {
	case left.Major != right.Major:
		if left.Major < right.Major {
			return -1, nil
		}
		return 1, nil
	case left.Minor != right.Minor:
		if left.Minor < right.Minor {
			return -1, nil
		}
		return 1, nil
	case left.Revision != right.Revision:
		if left.Revision < right.Revision {
			return -1, nil
		}
		return 1, nil
	default:
		return 0, nil
	}
}

// Supported returns the list of REST revisions recognised by the CLI in descending order.
func Supported() []string {
	out := make([]string, len(supportedVersions))
	copy(out, supportedVersions)
	return out
}

// Highest returns the most recent revision known to the CLI.
func Highest() string {
	return supportedVersions[0]
}

// IsSupported reports whether the CLI knows about the given revision.
func IsSupported(version string) bool {
	version = strings.TrimSpace(strings.ToLower(version))
	for _, candidate := range supportedVersions {
		if strings.EqualFold(candidate, version) {
			return true
		}
	}
	return false
}

// Normalize trims whitespace and lowercases the revision for comparison purposes.
func Normalize(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

// Build holds the leading components of a VBR build string (major.minor.patch).
type Build struct {
	Major int
	Minor int
	Patch int
}

// ParseBuild extracts a Build triple from a dotted build string (e.g. 13.0.0.4967).
func ParseBuild(value string) (Build, error) {
	fields := strings.Split(strings.TrimSpace(value), ".")
	if len(fields) < 3 {
		return Build{}, fmt.Errorf("invalid build number %q", value)
	}
	ints := make([]int, 3)
	for i := 0; i < 3; i++ {
		v, err := strconv.Atoi(fields[i])
		if err != nil {
			return Build{}, fmt.Errorf("invalid build number %q", value)
		}
		ints[i] = v
	}
	return Build{Major: ints[0], Minor: ints[1], Patch: ints[2]}, nil
}

// Compare returns -1 when b<other, 0 when b==other, 1 when b>other.
func (b Build) Compare(other Build) int {
	switch {
	case b.Major != other.Major:
		if b.Major < other.Major {
			return -1
		}
		return 1
	case b.Minor != other.Minor:
		if b.Minor < other.Minor {
			return -1
		}
		return 1
	case b.Patch != other.Patch:
		if b.Patch < other.Patch {
			return -1
		}
		return 1
	default:
		return 0
	}
}

// SuggestByBuild returns the closest known REST revision for a server build.
// The second return value reports whether the build could be mapped exactly.
// The third return value reports whether the build is newer than the CLI's known matrix.
func SuggestByBuild(build string) (string, bool, bool) {
	parsed, err := ParseBuild(build)
	if err != nil {
		return supportedVersions[len(supportedVersions)-1], false, false
	}

	if parsed.Compare(buildThresholds[0].min) > 0 {
		return supportedVersions[0], true, true
	}

	for _, candidate := range buildThresholds {
		if parsed.Compare(candidate.min) >= 0 {
			return candidate.ver, true, false
		}
	}

	return supportedVersions[len(supportedVersions)-1], false, false
}

// MergeCandidates merges and de-duplicates version candidates in descending order.
func MergeCandidates(values ...string) []string {
	set := make(map[string]struct{}, len(values))
	normalized := make([]VersionWithRaw, 0, len(values))
	for _, raw := range values {
		trimmed := Normalize(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}

		parsed, err := Parse(trimmed)
		if err != nil {
			// Unknown format: keep as-is but push to the back to honour caller ordering.
			normalized = append(normalized, VersionWithRaw{Raw: trimmed, Unknown: true})
			continue
		}
		normalized = append(normalized, VersionWithRaw{Raw: trimmed, Parsed: parsed})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		ai := normalized[i]
		aj := normalized[j]
		if ai.Unknown && aj.Unknown {
			return i < j
		}
		if ai.Unknown {
			return false
		}
		if aj.Unknown {
			return true
		}
		if ai.Parsed.Major != aj.Parsed.Major {
			return ai.Parsed.Major > aj.Parsed.Major
		}
		if ai.Parsed.Minor != aj.Parsed.Minor {
			return ai.Parsed.Minor > aj.Parsed.Minor
		}
		return ai.Parsed.Revision > aj.Parsed.Revision
	})

	out := make([]string, len(normalized))
	for i, v := range normalized {
		out[i] = v.Raw
	}
	return out
}

// VersionWithRaw tracks parsed and raw values for ordering purposes.
type VersionWithRaw struct {
	Raw     string
	Parsed  Version
	Unknown bool
}
