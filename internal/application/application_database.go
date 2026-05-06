package application

import (
	"context"

	"github.com/servercurio/go-cli-starter/internal/database"
	"github.com/servercurio/go-cli-starter/internal/database/orm"
	apperrors "github.com/servercurio/go-cli-starter/internal/errors"
	"github.com/servercurio/go-cli-starter/internal/logging"
)

// IsDatabaseHealthy reports whether the configured database is currently
// reachable. Returns true when the database subsystem is disabled (no DSN
// configured) so callers can compose readiness checks without special-casing
// the disabled state. The context bounds the underlying PingContext —
// readiness handlers pass the per-check budget so a hung pool can't stall
// /readyz past kubelet's probe deadline.
func (app *Application) IsDatabaseHealthy(ctx context.Context) bool {
	if !app.config.Database.Enabled() {
		return true
	}
	return database.IsHealthy(ctx)
}

// initializeDatabase opens the database connection, runs any pending Goose
// migrations, and configures the Bun ORM singleton. It is a no-op when the
// database subsystem is disabled (empty DSN), so the daemon can run as a
// pure HTTP server with no database backing.
func (app *Application) initializeDatabase() error {
	if !app.config.Database.Enabled() {
		logging.Daemon.Info().Msg("database subsystem disabled (no DSN configured)")
		return nil
	}

	if err := database.Connect(app.config.Database); err != nil {
		return apperrors.ConnectionFailed.Wrap(err, "database connect failed")
	}

	if err := database.Migrate(app.config.Database); err != nil {
		return apperrors.MigrationFailed.Wrap(err, "database migration failed")
	}

	if err := orm.Configure(); err != nil {
		return apperrors.ORMConfigurationFailed.Wrap(err, "ORM configuration failed")
	}

	logging.Daemon.Info().Msg("database initialized")
	return nil
}

// shutdownDatabase closes the database connection pool. Mirrors the shape of
// shutdownHttpServer / shutdownTlsServer so Application.Start can fan
// shutdown across subsystems uniformly. Errors are logged but not returned —
// shutdown should always make progress.
func (app *Application) shutdownDatabase() {
	if err := database.Disconnect(); err != nil {
		logging.Daemon.Warn().Err(err).Msg("error closing database connection")
	}
}
