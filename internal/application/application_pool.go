package application

import (
	"time"

	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/pool"
)

// shutdownPoolTimeout bounds how long Application.shutdown waits for
// in-flight pool workers to drain before returning. Mirrors the bounded-wait
// pattern previously used for HTTP shutdown.
const shutdownPoolTimeout = 5 * time.Second

// initializePool constructs the shared goroutine pool from app.config.Pool.
// Stashes the result on Application so subcommands can reach it via
// app.Pool(). Errors here are fatal — a CLI that intends to fan out work
// can't usefully proceed without its pool.
func (app *Application) initializePool() error {
	p, err := pool.New(app.config.Pool)
	if err != nil {
		return err
	}
	app.pool = p

	stats := p.Stats()
	logging.Daemon.Info().
		Int("capacity", stats.Capacity).
		Msg("goroutine pool initialized")
	return nil
}

// shutdownPool releases the shared goroutine pool with a bounded wait so a
// stuck worker can't hang the process. Errors are logged but not returned —
// shutdown should always make progress.
func (app *Application) shutdownPool() {
	if app.pool == nil {
		return
	}
	if err := app.pool.ReleaseTimeout(shutdownPoolTimeout); err != nil {
		logging.Daemon.Warn().
			Err(err).
			Dur("timeout", shutdownPoolTimeout).
			Msg("goroutine pool did not drain within timeout")
	} else {
		logging.Daemon.Info().Msg("goroutine pool released")
	}
}

// NotifyPoolConfig emits the resolved pool configuration to the daemon log.
// No-op for a nil config.
func NotifyPoolConfig(cfg *pool.Config) {
	if cfg == nil {
		return
	}
	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("pool configuration")
}
