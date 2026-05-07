package application

import (
	"errors"

	"github.com/servercurio/go-cli-starter/internal/database"
	"github.com/servercurio/go-cli-starter/internal/env"
	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/pool"
)

// Config is the CLI's top-level configuration aggregate. Each field is owned
// by the corresponding subsystem, with this type acting as the wiring node
// that fans environment-variable hydration and validation out to each.
type Config struct {
	Logging  *logging.Config  `yaml:"logging" json:"logging"`
	Database *database.Config `yaml:"database" json:"database"`
	Pool     *pool.Config     `yaml:"pool" json:"pool"`
}

// FromEnv hydrates each subsystem config from environment variables under
// the corresponding child prefix (e.g. <prefix>_DATABASE_*, <prefix>_POOL_*).
func (c *Config) FromEnv(prefix string) {
	c.Logging.FromEnv(prefix)
	c.Database.FromEnv(env.AddPrefix(prefix, "database"))
	c.Pool.FromEnv(env.AddPrefix(prefix, "pool"))
}

// Validate fans out to every sub-config's Validate and joins the results so
// Configure can return a single error containing every issue found. Each
// subcommand surfaces a non-nil Configure error to Cobra, which prints it
// and returns a non-zero exit code; surfacing all issues at once means the
// operator does not have to iterate boot-fix-boot per problem.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	return errors.Join(
		c.Logging.Validate(),
		c.Database.Validate(),
		c.Pool.Validate(),
	)
}

// DefaultConfig returns a Config populated with each subsystem's defaults.
// The result is suitable for handing straight to NewApplication when no
// config files or env vars are available.
func DefaultConfig() *Config {
	return &Config{
		Logging:  logging.DefaultLoggingConfig(),
		Database: database.DefaultConfig(),
		Pool:     pool.DefaultConfig(),
	}
}
