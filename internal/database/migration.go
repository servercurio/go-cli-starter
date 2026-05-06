package database

import (
	"embed"

	"github.com/joomcode/errorx"
	"github.com/pressly/goose/v3"
	"github.com/servercurio/go-echo-starter/internal/logging"
)

// migrationFS holds the Goose SQL migration files embedded at build time.
// New migrations are picked up automatically — drop a YYYYMMDDHHMMSS_*.sql
// file into migrations/sql/ and the next build rebundles it.
//
//go:embed migrations/sql/*.sql
var migrationFS embed.FS

// Migrate applies all pending Goose SQL migrations from the embedded
// migrations/sql directory against the established database connection.
//
// Returns nil (a no-op) when the database subsystem is disabled (cfg has
// empty DSN). Returns an error if Connect was supposed to have been called
// but no connection exists, or if any migration fails.
func Migrate(cfg *Config) error {
	if !cfg.Enabled() {
		return nil
	}

	conn := Connection()
	if conn == nil {
		return errorx.InitializationFailed.New("database connection not established; call Connect before Migrate")
	}

	goose.SetLogger(logging.AsStdLogger(logging.Daemon))
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return errorx.InitializationFailed.Wrap(err, "failed to set goose dialect")
	}

	if err := goose.Up(conn, "migrations/sql"); err != nil {
		return errorx.ExternalError.Wrap(err, "failed to migrate database schema")
	}

	return nil
}
