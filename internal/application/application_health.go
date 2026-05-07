package application

import (
	"context"

	"github.com/servercurio/go-cli-starter/internal/health"
)

// registerHealthChecks populates the application's health registry with one
// entry per active subsystem. Called once during Initialize, after every
// subsystem has had a chance to read its configuration. The closures
// captured here may be invoked frequently (e.g. by a periodic poller or a
// `health` subcommand), so they must stay cheap — long checks belong
// behind a cached background probe.
func (app *Application) registerHealthChecks() {
	// Lifecycle: reflects the atomic.Bool flipped by Run / RunUntilSignal.
	app.healthRegistry.Register("lifecycle", func(_ context.Context) health.ComponentResult {
		if app.IsReady() {
			return health.ComponentResult{Status: health.StatusUp}
		}
		return health.ComponentResult{
			Status:  health.StatusDown,
			Details: map[string]any{"reason": "application has not yet entered ready state, or shutdown has begun"},
		}
	})

	// Database: only registered when configured. Status reflects a live
	// PingContext on the pool (see app.IsDatabaseHealthy).
	dbCfg := app.config.Database
	if dbCfg.Enabled() {
		app.healthRegistry.Register("database", func(ctx context.Context) health.ComponentResult {
			details := map[string]any{
				"driver": dbCfg.Driver,
			}
			if app.IsDatabaseHealthy(ctx) {
				return health.ComponentResult{Status: health.StatusUp, Details: details}
			}
			if ctx.Err() == context.DeadlineExceeded {
				details["reason"] = "ping exceeded check budget"
			} else {
				details["reason"] = "ping failed"
			}
			return health.ComponentResult{Status: health.StatusDown, Details: details}
		})
	}

	// Pool: surfaces capacity / running / free so consumers can spot
	// saturation. UP whenever the pool is constructed and not released.
	app.healthRegistry.Register("pool", func(_ context.Context) health.ComponentResult {
		if app.pool == nil {
			return health.ComponentResult{
				Status:  health.StatusDown,
				Details: map[string]any{"reason": "pool not initialised"},
			}
		}
		stats := app.pool.Stats()
		if stats.Capacity <= 0 {
			return health.ComponentResult{
				Status:  health.StatusDown,
				Details: map[string]any{"reason": "pool released or zero-capacity"},
			}
		}
		return health.ComponentResult{
			Status: health.StatusUp,
			Details: map[string]any{
				"capacity": stats.Capacity,
				"running":  stats.Running,
				"free":     stats.Free,
			},
		}
	})
}
