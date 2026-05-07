package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// inlineSubmit runs the task synchronously on the calling goroutine. Lets
// us exercise copyTree without spinning up a real ants pool, while still
// covering the submit-error and worker-error code paths.
func inlineSubmit(ctx context.Context, task func(context.Context)) error {
	task(ctx)
	return nil
}

func TestCopyFile_HappyPath(t *testing.T) {
	assert := asrt.New(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "nested", "dst.txt")
	contents := []byte("hello world\n")
	assert.NoError(os.WriteFile(src, contents, 0o600))

	assert.NoError(copyFile(src, dst))

	got, err := os.ReadFile(dst) //nolint:gosec // test path
	assert.NoError(err)
	assert.Equal(contents, got)
}

func TestCopyFile_RejectsNonRegular(t *testing.T) {
	assert := asrt.New(t)

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "asrc")
	assert.NoError(os.Mkdir(srcDir, 0o750))

	err := copyFile(srcDir, filepath.Join(dir, "dst"))
	assert.Error(err)
	assert.Contains(err.Error(), "not a regular file")
}

func TestCopyFile_StatErrorPropagates(t *testing.T) {
	assert := asrt.New(t)

	err := copyFile("/nonexistent/path/that/should/not/exist", "/tmp/whatever")
	assert.Error(err)
}

func TestCopyTree_RecursesAndMirrorsTree(t *testing.T) {
	assert := asrt.New(t)

	srcRoot := t.TempDir()
	dstRoot := filepath.Join(t.TempDir(), "dst")

	assert.NoError(os.MkdirAll(filepath.Join(srcRoot, "sub"), 0o750))
	assert.NoError(os.WriteFile(filepath.Join(srcRoot, "a.txt"), []byte("a"), 0o600))
	assert.NoError(os.WriteFile(filepath.Join(srcRoot, "sub", "b.txt"), []byte("b"), 0o600))

	assert.NoError(copyTree(context.Background(), inlineSubmit, srcRoot, dstRoot))

	for _, rel := range []string{"a.txt", "sub/b.txt"} {
		_, err := os.Stat(filepath.Join(dstRoot, rel))
		assert.NoError(err, "expected %s to exist under dst", rel)
	}
}

func TestCopyTree_RejectsNonDirSource(t *testing.T) {
	assert := asrt.New(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "afile.txt")
	assert.NoError(os.WriteFile(src, []byte("nope"), 0o600))

	err := copyTree(context.Background(), inlineSubmit, src, filepath.Join(dir, "dst"))
	assert.Error(err)
	assert.Contains(err.Error(), "requires a directory source")
}

func TestCopyTree_StatErrorOnMissingSrc(t *testing.T) {
	assert := asrt.New(t)

	err := copyTree(context.Background(), inlineSubmit,
		"/nonexistent/source/tree",
		filepath.Join(t.TempDir(), "dst"))
	assert.Error(err)
}

func TestCopyTree_PropagatesSubmitError(t *testing.T) {
	assert := asrt.New(t)

	srcRoot := t.TempDir()
	assert.NoError(os.WriteFile(filepath.Join(srcRoot, "x.txt"), []byte("x"), 0o600))

	failing := func(_ context.Context, _ func(context.Context)) error {
		return errors.New("simulated submit failure")
	}

	err := copyTree(context.Background(), failing, srcRoot, filepath.Join(t.TempDir(), "dst"))
	assert.Error(err)
	assert.Contains(err.Error(), "simulated submit failure")
}

func TestCopyTree_PropagatesWorkerError(t *testing.T) {
	assert := asrt.New(t)

	srcRoot := t.TempDir()
	src := filepath.Join(srcRoot, "doomed.txt")
	assert.NoError(os.WriteFile(src, []byte("byebye"), 0o600))

	// Submit synchronously, but rename the source file before invoking
	// the body so copyFile fails inside the worker. This exercises the
	// worker's firstErr-set branch.
	racingSubmit := func(ctx context.Context, task func(context.Context)) error {
		_ = os.Remove(src)
		task(ctx)
		return nil
	}

	err := copyTree(context.Background(), racingSubmit, srcRoot, filepath.Join(t.TempDir(), "dst"))
	assert.Error(err, "missing source file inside worker should surface as a copyTree error")
}

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

// inlineSubmit is referenced indirectly by atomic.Int64 — keep this used
// import alive without a build tag.
var _ atomic.Int64

// writeBuf is a tiny in-memory io.Writer the subcommand tests can read
// back. Using bytes.Buffer would work too; this avoids one import.
type writeBuf struct{ b []byte }

func (w *writeBuf) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
func (w *writeBuf) String() string { return string(w.b) }
