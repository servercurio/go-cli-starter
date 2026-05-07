package cli

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestVersionCommand_PrintsTagAndCommit(t *testing.T) {
	assert := asrt.New(t)

	cmd := newVersionCommand()
	out := &writeBuf{}
	cmd.SetOut(out)
	assert.NoError(cmd.Execute())
	assert.Contains(out.String(), "v")
}

func TestNewRootCommand_HelpRunsWithoutPanic(t *testing.T) {
	// Help avoids PreRunE side-effects (config loading, logger init).
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--help"})
	out := &writeBuf{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	asrt.NotPanics(t, func() { _ = cmd.Execute() })
	asrt.Contains(t, out.String(), "Available Commands")
	// Ensure all expected subcommands are listed.
	for _, name := range []string{"serve", "copy", "version"} {
		asrt.Contains(t, out.String(), name)
	}
}

// writeBuf is a tiny in-memory io.Writer the subcommand tests can read
// back. Using bytes.Buffer would work too; this avoids one import.
type writeBuf struct{ b []byte }

func (w *writeBuf) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
func (w *writeBuf) String() string { return string(w.b) }
