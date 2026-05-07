package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"

	"github.com/servercurio/go-cli-starter/internal/logging"
)

// copyFlags holds the per-invocation flags for the copy subcommand.
type copyFlags struct {
	recursive bool
}

// newCopyCommand returns the one-shot example: copies SRC to DST, fanning
// recursive copies through app.Pool() so consumers see the canonical
// pool-backed worker pattern.
func newCopyCommand(rc *rootContext) *cobra.Command {
	flags := &copyFlags{}

	cmd := &cobra.Command{
		Use:   "copy SRC DST",
		Short: "Copy a file or directory (example one-shot)",
		Long: `copy is a one-shot example demonstrating positional args, optional
flags, structured logging, and pool-backed parallelism.

Without --recursive, SRC must be a regular file and is copied to DST.
With --recursive, SRC must be a directory; every file under it is
submitted to the shared goroutine pool for parallel copy.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := rc.app
			if err := app.Initialize(); err != nil {
				return err
			}

			src, dst := args[0], args[1]
			return app.Run(cmd.Context(), func(ctx context.Context) error {
				if flags.recursive {
					return copyTree(ctx, app.Pool().SubmitWithContext, src, dst)
				}
				return copyFile(src, dst)
			})
		},
	}

	cmd.Flags().BoolVarP(&flags.recursive, "recursive", "r", false, "recursively copy a directory tree using the goroutine pool")

	return cmd
}

// submitFunc matches *pool.Pool.SubmitWithContext. Accepted as a parameter
// so copyTree stays unit-testable without spinning up a real pool.
type submitFunc func(ctx context.Context, task func(context.Context)) error

// copyTree walks src, mirrors its directory structure under dst, and
// submits one file copy per regular file through submit. Returns the first
// error any worker observed, having waited for in-flight workers to drain.
func copyTree(ctx context.Context, submit submitFunc, src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("--recursive requires a directory source, got %s", src)
	}

	var (
		wg        sync.WaitGroup
		firstErr  atomic.Pointer[error]
		filesDone atomic.Int64
	)

	walkErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !d.Type().IsRegular() {
			return nil
		}

		wg.Add(1)
		submitErr := submit(ctx, func(_ context.Context) {
			defer wg.Done()
			if err := copyFile(path, target); err != nil {
				e := fmt.Errorf("copy %s: %w", path, err)
				firstErr.CompareAndSwap(nil, &e)
				return
			}
			filesDone.Add(1)
		})
		if submitErr != nil {
			wg.Done()
			return submitErr
		}
		return nil
	})

	wg.Wait()

	if walkErr != nil {
		return walkErr
	}
	if e := firstErr.Load(); e != nil {
		return *e
	}

	logging.Daemon.Info().
		Int64("filesCopied", filesDone.Load()).
		Str("src", src).
		Str("dst", dst).
		Msg("recursive copy complete")
	return nil
}

// copyFile copies a single regular file, creating dst's parent directory
// if needed. Permission bits on dst match the source.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	if mkErr := os.MkdirAll(filepath.Dir(dst), 0o750); mkErr != nil {
		return mkErr
	}

	// Paths come from CLI args; reading user-supplied paths is the
	// entire point of the copy subcommand.
	in, err := os.Open(src) //nolint:gosec // G304: user-supplied path is intentional
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode().Perm()) //nolint:gosec // G304: user-supplied path is intentional
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
