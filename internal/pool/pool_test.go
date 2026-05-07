package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joomcode/errorx"
	asrt "github.com/stretchr/testify/assert"

	apperrors "github.com/servercurio/go-cli-starter/internal/errors"
)

func TestNew_NilConfigUsesDefaults(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(nil)
	assert.NoError(err)
	assert.NotNil(p)
	defer p.Release()

	// DefaultConfig sizes the pool to NumCPU()*2; just confirm capacity > 0.
	assert.Greater(p.Stats().Capacity, 0)
}

func TestNew_ExplicitConfigHonoured(t *testing.T) {
	assert := asrt.New(t)

	cfg := DefaultConfig()
	cfg.Size = 4
	p, err := New(cfg)
	assert.NoError(err)
	defer p.Release()

	assert.Equal(4, p.Stats().Capacity)
	assert.Equal(0, p.Stats().Running)
	assert.Equal(4, p.Stats().Free)
}

func TestSubmit_ExecutesTask(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 2, ExpiryDuration: time.Second})
	assert.NoError(err)
	defer p.Release()

	var ran atomic.Bool
	done := make(chan struct{})
	assert.NoError(p.Submit(func() {
		ran.Store(true)
		close(done)
	}))

	select {
	case <-done:
		assert.True(ran.Load())
	case <-time.After(time.Second):
		t.Fatal("submitted task did not execute within 1s")
	}
}

func TestSubmit_NilReceiverReturnsPoolReleased(t *testing.T) {
	assert := asrt.New(t)

	var p *Pool // nil
	err := p.Submit(func() {})
	assert.True(errorx.IsOfType(err, apperrors.PoolReleased),
		"submit on nil pool must return PoolReleased; got %v", err)
}

func TestSubmit_AfterReleaseReturnsPoolReleased(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 1, ExpiryDuration: time.Second})
	assert.NoError(err)

	p.Release()
	err = p.Submit(func() {})
	assert.True(errorx.IsOfType(err, apperrors.PoolReleased),
		"submit after release must wrap as PoolReleased; got %v", err)
}

func TestSubmit_NonBlockingExhaustionReturnsPoolExhausted(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 1, NonBlocking: true, ExpiryDuration: time.Second})
	assert.NoError(err)
	defer p.Release()

	// Park a worker so the pool is at capacity.
	hold := make(chan struct{})
	assert.NoError(p.Submit(func() { <-hold }))
	defer close(hold)

	// Give the worker a moment to register as running before submitting again.
	for i := 0; i < 50 && p.Stats().Running == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}

	err = p.Submit(func() {})
	assert.True(errorx.IsOfType(err, apperrors.PoolExhausted),
		"non-blocking submit at capacity must return PoolExhausted; got %v", err)
}

func TestSubmitWithContext_RunsTaskWhenCtxLive(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 2, ExpiryDuration: time.Second})
	assert.NoError(err)
	defer p.Release()

	var ran atomic.Bool
	done := make(chan struct{})
	assert.NoError(p.SubmitWithContext(context.Background(), func(_ context.Context) {
		ran.Store(true)
		close(done)
	}))

	<-done
	assert.True(ran.Load())
}

func TestSubmitWithContext_SkipsTaskWhenCtxAlreadyCancelled(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 2, ExpiryDuration: time.Second})
	assert.NoError(err)
	defer p.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var ran atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	assert.NoError(p.SubmitWithContext(ctx, func(_ context.Context) {
		defer wg.Done()
		ran.Store(true)
	}))

	// The ants worker still calls into our wrapper, but the wrapper
	// returns early without invoking the user task. wg.Done is called
	// from the user task — never; so use a small timeout instead.
	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()

	select {
	case <-doneCh:
		t.Fatal("task body should have been skipped when ctx is already cancelled")
	case <-time.After(150 * time.Millisecond):
		assert.False(ran.Load(), "task body must not run when ctx is cancelled")
	}
}

func TestSubmitWithContext_NilCtxTreatedAsBackground(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 1, ExpiryDuration: time.Second})
	assert.NoError(err)
	defer p.Release()

	done := make(chan struct{})
	//lint:ignore SA1012 nil context is intentional — wrapper should default to Background
	assert.NoError(p.SubmitWithContext(nil, func(_ context.Context) { close(done) })) //nolint:staticcheck

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task did not run with nil ctx (should default to Background)")
	}
}

func TestStats_NilReceiverIsZeroValue(t *testing.T) {
	assert := asrt.New(t)

	var p *Pool
	assert.Equal(Stats{}, p.Stats())
}

func TestRelease_IdempotentAndSafeOnNil(t *testing.T) {
	assert := asrt.New(t)

	var p *Pool
	assert.NotPanics(func() { p.Release() })

	p2, err := New(&Config{Size: 1, ExpiryDuration: time.Second})
	assert.NoError(err)
	assert.NotPanics(func() { p2.Release() })
	assert.NotPanics(func() { p2.Release() }, "double-release must not panic")
}

func TestReleaseTimeout_NilReceiverReturnsNil(t *testing.T) {
	assert := asrt.New(t)

	var p *Pool
	assert.NoError(p.ReleaseTimeout(10 * time.Millisecond))
}

func TestReleaseTimeout_DrainsCleanly(t *testing.T) {
	assert := asrt.New(t)

	p, err := New(&Config{Size: 1, ExpiryDuration: time.Second})
	assert.NoError(err)

	// No outstanding work — should drain immediately.
	assert.NoError(p.ReleaseTimeout(time.Second))
}

func TestWrapSubmitErr_FallthroughIsSubmitFailed(t *testing.T) {
	assert := asrt.New(t)

	wrapped := wrapSubmitErr(errors.New("some other ants error"))
	assert.True(errorx.IsOfType(wrapped, apperrors.SubmitFailed),
		"unrecognised error must wrap as SubmitFailed; got %v", wrapped)
}
