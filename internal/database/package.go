// Package database manages the application's optional SQL database connection
// pool, schema migrations, and ORM singleton. The starter ships PostgreSQL
// (pgx) bindings by default; replace the driver and dialect to swap engines.
//
// The database is opt-in: an empty DSN at startup means Connect/Migrate are
// skipped and the readiness probe ignores DB state. This keeps the starter
// usable as a pure HTTP server with no database.
package database
