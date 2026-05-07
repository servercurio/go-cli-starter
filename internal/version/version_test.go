package version

import (
	"strings"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestCommit_TrimmedAndLowerCased(t *testing.T) {
	c := Commit()
	asrt.Equal(t, strings.TrimSpace(strings.ToLower(c)), c,
		"Commit must be trimmed and lower-cased")
}

func TestSemVer_NeverNil(t *testing.T) {
	v := SemVer()
	asrt.NotNil(t, v, "SemVer must always return a non-nil version (falls back to 0.0.0)")
}

func TestNumber_NonEmpty(t *testing.T) {
	asrt.NotEmpty(t, Number(), "Number must produce a dotted semver string")
}

func TestTag_HasVPrefix(t *testing.T) {
	tag := Tag()
	asrt.True(t, strings.HasPrefix(tag, "v"), "Tag must be prefixed with 'v'; got %q", tag)
	asrt.Equal(t, "v"+Number(), tag, "Tag must equal 'v' + Number")
}
