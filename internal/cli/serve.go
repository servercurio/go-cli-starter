package cli

import (
	"github.com/spf13/cobra"
)

// newServeCommand returns the long-running daemon example. The actual
// daemon lifecycle (Initialize → RunUntilSignal) lives on Application.Serve;
// this file is a thin Cobra shell that delegates to it.
func newServeCommand(rc *rootContext) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the long-running daemon (example)",
		Long: `serve starts the example daemon: initializes the database (if configured),
the goroutine pool, and the health registry; logs a "ready" event with pool
stats; then blocks until a shutdown signal arrives. Replace the body in
internal/application/application_serve.go with your own loop, scheduler, or
worker dispatch — the lifecycle plumbing around it is the part worth keeping.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return rc.app.Serve(cmd.Context())
		},
	}
}
