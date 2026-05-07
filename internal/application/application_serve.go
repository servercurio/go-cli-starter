package application

import (
	"context"

	"github.com/servercurio/go-cli-starter/internal/logging"
)

// Serve is the canonical daemon subcommand entry point: it owns the full
// CLI lifecycle (Initialize → RunUntilSignal → shutdown), logs a "ready"
// event with pool stats, and blocks until a shutdown signal arrives.
// Pulled onto Application so non-Cobra callers can drive the same daemon
// loop without going through cobra. Replace the body for a real workload.
func (app *Application) Serve(ctx context.Context) error {
	if err := app.Initialize(); err != nil {
		return err
	}

	stats := app.Pool().Stats()
	logging.Daemon.Info().
		Int("poolCapacity", stats.Capacity).
		Msg("daemon ready; awaiting shutdown signal")

	return app.RunUntilSignal(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		logging.Daemon.Info().Msg("shutdown signal received; daemon exiting")
		return nil
	})
}
