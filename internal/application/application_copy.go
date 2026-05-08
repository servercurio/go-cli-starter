package application

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/servercurio/go-cli-starter/internal/logging"
)

// Copy is the canonical one-shot subcommand entry point: it owns the full
// CLI lifecycle (Initialize → Run → shutdown) and dispatches to a single-file
// or recursive directory copy depending on the recursive flag. Pulled onto
// Application so non-Cobra callers (tests, library use, alternate runtimes)
// can invoke it without going through cobra.
func (app *Application) Copy(ctx context.Context, src, dst string, recursive bool) error {
	if err := app.Initialize(); err != nil {
		return err
	}

	return app.Run(ctx, func(ctx context.Context) error {
		if recursive {
			return copyTree(ctx, app.Pool().SubmitWithContext, src, dst)
		}
		return copyFile(src, dst)
	})
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
