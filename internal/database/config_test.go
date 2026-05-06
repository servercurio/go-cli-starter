package database

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestConfig_EnabledRequiresNonEmptyDSN(t *testing.T) {
	assert := asrt.New(t)

	assert.False((*Config)(nil).Enabled(), "nil config must not be enabled")
	assert.False((&Config{Driver: "pgx", DSN: ""}).Enabled(), "empty DSN means disabled")
	assert.True((&Config{Driver: "pgx", DSN: "postgres://localhost/app"}).Enabled())
}

func TestConfig_FromEnvOverlaysDriverAndDSN(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_DATABASE_DRIVER", "pgx")
	t.Setenv("APP_DATABASE_DSN", "postgres://app@db.example.com:5432/app")

	cfg := DefaultConfig()
	cfg.FromEnv("APP_DATABASE")

	assert.Equal("pgx", cfg.Driver)
	assert.Equal("postgres://app@db.example.com:5432/app", cfg.DSN)
	assert.True(cfg.Enabled())
}

func TestDefaultConfig_IsDisabledByDefault(t *testing.T) {
	assert := asrt.New(t)
	cfg := DefaultConfig()

	assert.False(cfg.Enabled(), "starter must default to no database so it runs without external dependencies")
	assert.NotEmpty(cfg.Driver, "default driver should be populated so enabling the DB only requires setting a DSN")
}

// TestConfig_Validate exercises the rules that previously deferred to the
// database/sql layer at Connect time. Disabled configs (empty DSN) skip
// validation; enabled configs need a driver name and non-negative pool
// sizes.
func TestConfig_Validate(t *testing.T) {
	assert := asrt.New(t)

	t.Run("disabled config skips validation", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.NoError(cfg.Validate(), "default (disabled) config must validate")
	})

	t.Run("enabled with driver", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DSN = "postgres://localhost/app"
		assert.NoError(cfg.Validate())
	})

	t.Run("enabled without driver", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Driver = ""
		cfg.DSN = "postgres://localhost/app"
		err := cfg.Validate()
		assert.Error(err)
		assert.Contains(err.Error(), "driver is required")
	})

	t.Run("negative MaxOpenConns", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DSN = "postgres://localhost/app"
		cfg.MaxOpenConns = -1
		err := cfg.Validate()
		assert.Error(err)
		assert.Contains(err.Error(), "maxOpenConns")
	})
}
