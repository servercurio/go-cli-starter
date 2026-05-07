package pool

import (
	"context"
	"errors"
	"time"

	"github.com/panjf2000/ants/v2"
	apperrors "github.com/servercurio/go-cli-starter/internal/errors"
)

// Pool wraps an ants.Pool with project-typed Submit errors and a
// context-aware SubmitWithContext helper. The zero value is unusable —
// construct via New.
type Pool struct {
	inner *ants.Pool
}

// Stats is a point-in-time snapshot of pool utilisation, suitable for
// embedding in a structured log event or surfacing through a health-check
// component result.
type Stats struct {
	Capacity int `json:"capacity"`
	Running  int `json:"running"`
	Free     int `json:"free"`
}

// New constructs a Pool from cfg. A nil cfg is treated as DefaultConfig().
// The returned Pool must be Released by the caller (the Application
// lifecycle handles this in shutdownPool).
func New(cfg *Config) (*Pool, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := []ants.Option{
		ants.WithExpiryDuration(cfg.ExpiryDuration),
		ants.WithPreAlloc(cfg.PreAlloc),
		ants.WithMaxBlockingTasks(cfg.MaxBlockingTasks),
		ants.WithNonblocking(cfg.NonBlocking),
	}

	inner, err := ants.NewPool(cfg.Size, opts...)
	if err != nil {
		return nil, apperrors.SubmitFailed.Wrap(err, "failed to create goroutine pool")
	}

	return &Pool{inner: inner}, nil
}

// Submit queues task for execution. Returns a typed PoolExhausted /
// PoolReleased / SubmitFailed error so callers can branch via errorx
// without importing ants.
func (p *Pool) Submit(task func()) error {
	if p == nil || p.inner == nil {
		return apperrors.PoolReleased.New("pool is not initialised")
	}

	if err := p.inner.Submit(task); err != nil {
		return wrapSubmitErr(err)
	}
	return nil
}

// SubmitWithContext queues task for execution and arranges for ctx to be
// passed in to the worker. ants does not natively accept contexts, so
// the wrapper checks ctx.Err() before invoking task — long-running tasks
// must continue to honour ctx themselves to be cancellable mid-flight.
// Returns the same typed errors as Submit.
func (p *Pool) SubmitWithContext(ctx context.Context, task func(context.Context)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return p.Submit(func() {
		if ctx.Err() != nil {
			return
		}
		task(ctx)
	})
}

// Stats returns the current utilisation snapshot.
func (p *Pool) Stats() Stats {
	if p == nil || p.inner == nil {
		return Stats{}
	}
	return Stats{
		Capacity: p.inner.Cap(),
		Running:  p.inner.Running(),
		Free:     p.inner.Free(),
	}
}

// Release shuts the pool down, refusing further Submit calls. Safe to call
// multiple times — subsequent calls are no-ops.
func (p *Pool) Release() {
	if p == nil || p.inner == nil {
		return
	}
	p.inner.Release()
}

// ReleaseTimeout shuts the pool down and waits up to d for in-flight
// workers to drain. Returns the underlying timeout error (wrapped) when
// any workers are still running after d elapses.
func (p *Pool) ReleaseTimeout(d time.Duration) error {
	if p == nil || p.inner == nil {
		return nil
	}
	if err := p.inner.ReleaseTimeout(d); err != nil {
		return apperrors.SubmitFailed.Wrap(err, "pool release timed out after %s", d)
	}
	return nil
}

// wrapSubmitErr converts an ants Submit error into the project's typed
// error categories. Callers can then use errorx.IsOfType to branch.
func wrapSubmitErr(err error) error {
	switch {
	case errors.Is(err, ants.ErrPoolOverload):
		return apperrors.PoolExhausted.Wrap(err, "pool is at capacity and configured non-blocking")
	case errors.Is(err, ants.ErrPoolClosed):
		return apperrors.PoolReleased.Wrap(err, "pool has been released")
	default:
		return apperrors.SubmitFailed.Wrap(err, "failed to submit task to pool")
	}
}
