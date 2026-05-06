package version

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Commit returns the lower-cased, whitespace-trimmed git commit hash baked
// into the binary at build time. Returns an empty string when commit.txt
// was empty (e.g. building from a tarball without git history).
func Commit() string {
	return strings.TrimSpace(strings.ToLower(commit))
}

// SemVer returns the semantic version baked into the binary, parsed via the
// Masterminds semver package. Falls back to 0.0.0 when version.txt is empty
// or unparseable so callers can rely on a non-nil result.
func SemVer() *semver.Version {
	if v, err := semver.NewVersion(version); err == nil {
		return v
	}

	return semver.New(0, 0, 0, "", "")
}

// Number returns the dotted semver string (e.g. "1.2.3") with no leading "v"
// prefix.
func Number() string {
	return SemVer().String()
}

// Tag returns the git-tag form of the version, prefixed with "v"
// (e.g. "v1.2.3"), matching the tag scheme used by semantic-release.
func Tag() string {
	return "v" + SemVer().String()
}
