// Package application owns the CLI lifecycle: loading and validating
// configuration, wiring the optional database/ORM, the goroutine pool, and
// the health registry, and unwinding subsystems cleanly on shutdown.
//
// internal/cli constructs an Application via NewApplication and runs the
// Configure → Initialize sequence inside each subcommand's RunE — Run for
// one-shot commands and RunUntilSignal for long-running daemons.
package application
