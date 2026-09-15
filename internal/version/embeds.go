// SPDX-License-Identifier: Apache-2.0

package version

import (
	_ "embed"
)

// commit holds the git commit hash embedded at build time from commit.txt.
//
//go:embed commit.txt
var commit string

// version holds the semver string embedded at build time from version.txt.
//
//go:embed version.txt
var version string
