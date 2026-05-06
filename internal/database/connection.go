package database

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joomcode/errorx"
)

// connectPingTimeout caps the initial reachability probe Connect runs after
// sql.Open. Short enough that a misconfigured DSN fails the daemon's boot
// quickly; long enough to absorb transient cold-start latency.
const connectPingTimeout = 5 * time.Second

// dbConn is the package-wide *sql.DB singleton populated by Connect and
// returned by Connection.
var dbConn *sql.DB

// m guards Connect / Disconnect / Connection so concurrent boot-time and
// shutdown paths don't race on dbConn.
var m sync.Mutex

// Connection returns the shared *sql.DB singleton, or nil if Connect has not
// been called or DSN was empty.
func Connection() *sql.DB {
	m.Lock()
	defer m.Unlock()
	return dbConn
}

// Connect opens a new database connection using the driver and DSN in cfg,
// verifies reachability with a ping, and stores the result in the package
// singleton. Returns nil (a no-op) when cfg.Enabled() is false or when a
// connection is already established. Returns an error if the connection
// cannot be opened or pinged.
func Connect(cfg *Config) error {
	if !cfg.Enabled() {
		return nil
	}

	m.Lock()
	defer m.Unlock()

	if dbConn != nil {
		return nil
	}

	conn, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return errorx.InitializationFailed.Wrap(err, "failed to open database connection")
	}

	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	conn.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(context.Background(), connectPingTimeout)
	defer cancel()
	if err = conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return errorx.InitializationFailed.Wrap(err, "failed to ping database")
	}

	dbConn = conn
	return nil
}

// Disconnect closes the shared database connection and resets the singleton
// to nil. It is a no-op if no connection is established.
func Disconnect() error {
	m.Lock()
	defer m.Unlock()

	if dbConn == nil {
		return nil
	}

	if err := dbConn.Close(); err != nil {
		return errorx.InternalError.Wrap(err, "failed to close database connection")
	}

	dbConn = nil
	return nil
}

// IsHealthy returns true if a connection has been established AND a
// PingContext using the supplied context succeeds. Used by the readiness
// probe to gate /readyz on database availability — callers pass the
// per-check budgeted context from health.Registry.Snapshot, so the
// PingContext deadline is enforced at the registry level rather than
// inside this package. Returns false (without erroring) when the
// singleton hasn't been initialised yet or when the ping fails for any
// reason.
func IsHealthy(ctx context.Context) bool {
	conn := Connection()
	if conn == nil {
		return false
	}

	return conn.PingContext(ctx) == nil
}
