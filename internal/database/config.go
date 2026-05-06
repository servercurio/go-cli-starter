package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-echo-starter/internal/env"
	"github.com/servercurio/go-echo-starter/internal/obfusicate"
)

// Config holds the database connection parameters and pool sizing.
type Config struct {
	// Driver is the database/sql driver name to register (e.g. "pgx").
	Driver string `yaml:"driver" json:"driver"`

	// DSN is the connection string passed to sql.Open. An empty DSN disables
	// the database subsystem entirely: Connect/Migrate become no-ops and
	// readiness checks skip the database probe.
	DSN string `yaml:"dsn" json:"dsn"`

	// MaxOpenConns is the maximum number of open connections to the
	// database, including those in use and idle. Zero or negative means
	// unlimited (matches database/sql default). Defaults to 25.
	MaxOpenConns int `yaml:"maxOpenConns" json:"maxOpenConns"`

	// MaxIdleConns is the maximum number of idle connections held in the
	// pool. database/sql default is 2. Defaults to 5.
	MaxIdleConns int `yaml:"maxIdleConns" json:"maxIdleConns"`

	// ConnMaxLifetime is how long a connection may live before being
	// recycled. Useful for rotating credentials or working around proxies
	// that close idle connections. Zero means no maximum. Defaults to 1h.
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"`

	// ConnMaxIdleTime is how long an idle connection may sit in the pool
	// before being closed. Zero means no maximum. Defaults to 5m.
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" json:"connMaxIdleTime"`
}

// Enabled reports whether the database subsystem should be initialised. Returns
// true when a non-empty DSN has been configured.
func (c *Config) Enabled() bool {
	return c != nil && c.DSN != ""
}

// FromEnv overlays Config fields with values from environment variables under
// the given prefix (e.g. APP_DATABASE_DRIVER, APP_DATABASE_DSN).
func (c *Config) FromEnv(prefix string) {
	env.SetStringValue(prefix, "driver", &c.Driver)
	env.SetStringValue(prefix, "dsn", &c.DSN)
	env.SetIntValue(prefix, "max_open_conns", &c.MaxOpenConns)
	env.SetIntValue(prefix, "max_idle_conns", &c.MaxIdleConns)
	env.SetDurationValue(prefix, "conn_max_lifetime", &c.ConnMaxLifetime)
	env.SetDurationValue(prefix, "conn_max_idle_time", &c.ConnMaxIdleTime)
}

// MarshalZerologObject writes the database configuration to a zerolog event.
// The DSN is obfuscated to keep credentials out of logs.
func (c *Config) MarshalZerologObject(e *zerolog.Event) {
	e.Str("driver", c.Driver).
		Str("dsn", obfusicate.ConcealUriCredential(c.DSN)).
		Int("maxOpenConns", c.MaxOpenConns).
		Int("maxIdleConns", c.MaxIdleConns).
		Str("connMaxLifetime", c.ConnMaxLifetime.String()).
		Str("connMaxIdleTime", c.ConnMaxIdleTime.String()).
		Bool("enabled", c.Enabled())
}

// Validate enforces the rules sql.Open and the pool configuration would
// surface later: enabled (DSN-set) configs need a driver name; pool-size
// fields and connection-lifetime fields can't go negative. Disabled
// configs (DSN empty) skip validation entirely.
func (c *Config) Validate() error {
	if c == nil || !c.Enabled() {
		return nil
	}
	var errs []error
	if strings.TrimSpace(c.Driver) == "" {
		errs = append(errs, errors.New("database: driver is required when DSN is set"))
	}
	if c.MaxOpenConns < 0 {
		errs = append(errs, fmt.Errorf("database: maxOpenConns must be non-negative, got %d", c.MaxOpenConns))
	}
	if c.MaxIdleConns < 0 {
		errs = append(errs, fmt.Errorf("database: maxIdleConns must be non-negative, got %d", c.MaxIdleConns))
	}
	if c.ConnMaxLifetime < 0 {
		errs = append(errs, fmt.Errorf("database: connMaxLifetime must be non-negative, got %s", c.ConnMaxLifetime))
	}
	if c.ConnMaxIdleTime < 0 {
		errs = append(errs, fmt.Errorf("database: connMaxIdleTime must be non-negative, got %s", c.ConnMaxIdleTime))
	}
	return errors.Join(errs...)
}

// DefaultConfig returns a disabled-by-default Config. Set DSN (via config file
// or env var) to enable the database subsystem.
func DefaultConfig() *Config {
	return &Config{
		Driver:          "pgx",
		DSN:             "",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}
