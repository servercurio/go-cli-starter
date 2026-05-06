package errors

import e "github.com/joomcode/errorx"

var (
	// DatabaseErrors is the errorx namespace for failures originating in the
	// database subsystem (connect, migrate, ORM configuration). Wrap upstream
	// driver / goose / bun errors with one of the typed members below so the
	// caller can branch on category without string-matching.
	DatabaseErrors = e.NewNamespace("database")

	// ConnectionFailed marks failure to open or ping the database. Wrap
	// errors returned by sql.Open / db.Ping with this type.
	ConnectionFailed = DatabaseErrors.NewType("connection_failed")

	// MigrationFailed marks failure to apply schema migrations. Wrap errors
	// returned by goose.Up (and related) with this type so deploy automation
	// can distinguish "schema couldn't be applied" from "couldn't connect".
	MigrationFailed = DatabaseErrors.NewType("migration_failed")

	// ORMConfigurationFailed marks failure to wire the ORM (e.g. bun) on top
	// of an already-established sql.DB. Wrap errors returned by the ORM
	// initialization path with this type.
	ORMConfigurationFailed = DatabaseErrors.NewType("orm_configuration_failed")
)
