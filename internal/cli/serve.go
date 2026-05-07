package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/servercurio/go-cli-starter/internal/logging"
)

// newServeCommand returns the long-running daemon example. The body blocks
// until SIGINT/SIGTERM (or whatever shutdownSignals lists for the platform);
// downstream consumers should replace it with their own daemon loop.
func newServeCommand(rc *rootContext) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the long-running daemon (example)",
		Long: `serve starts the example daemon: initializes the database (if configured),
the goroutine pool, and the health registry; logs a "ready" event with pool
stats; then blocks until a shutdown signal arrives. Replace the body with
your own loop, scheduler, or worker dispatch — the lifecycle plumbing
around it is the part worth keeping.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := rc.app
			if err := app.Initialize(); err != nil {
				return err
			}

			stats := app.Pool().Stats()
			logging.Daemon.Info().
				Int("poolCapacity", stats.Capacity).
				Msg("daemon ready; awaiting shutdown signal")

			return app.RunUntilSignal(cmd.Context(), func(ctx context.Context) error {
				<-ctx.Done()
				logging.Daemon.Info().Msg("shutdown signal received; daemon exiting")
				return nil
			})
		},
	}
}
