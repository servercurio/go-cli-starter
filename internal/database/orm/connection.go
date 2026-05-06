package orm

import (
	"sync"

	"github.com/joomcode/errorx"
	"github.com/servercurio/go-cli-starter/internal/database"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// db is the package-wide *bun.DB singleton populated by Configure and
// returned by Database.
var db *bun.DB

// m guards Configure / Reset / Database so concurrent boot-time and shutdown
// paths don't race on db.
var m sync.Mutex

// Configure initialises the Bun ORM singleton by wrapping the established
// *sql.DB connection with a PostgreSQL dialect. It is a no-op if the
// singleton has already been configured. Returns an error if the underlying
// database connection has not been established yet.
func Configure() error {
	m.Lock()
	defer m.Unlock()

	conn := database.Connection()
	if conn == nil {
		return errorx.InitializationFailed.New("database connection is not initialized")
	}

	if db != nil {
		return nil
	}

	db = bun.NewDB(conn, pgdialect.New())

	return nil
}

// Reset clears the Bun ORM singleton, allowing Configure to be called again.
// Intended for use in tests that need a fresh ORM state between runs.
func Reset() {
	m.Lock()
	defer m.Unlock()
	db = nil
}

// Database returns the shared *bun.DB singleton. Returns nil if Configure
// has not been called yet.
func Database() *bun.DB {
	m.Lock()
	defer m.Unlock()
	return db
}
