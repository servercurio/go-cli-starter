package logging

import (
	"io"
	"testing"

	"github.com/rs/zerolog"
	asrt "github.com/stretchr/testify/assert"
)

func TestDefaultLoggingConfig(t *testing.T) {
	assert := asrt.New(t)

	c := DefaultLoggingConfig()
	assert.NotNil(c)
	assert.NotNil(c.Daemon)
	assert.True(c.Daemon.Enabled)
	assert.Equal(zerolog.LevelInfoValue, c.Daemon.Level)
	assert.True(c.Daemon.PrettyPrint)
	assert.False(c.Daemon.IncludeCaller)
}

func TestNewConfigFromEnv_AppliesEnvOverrides(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_DAEMON_LOG_LEVEL", "debug")
	t.Setenv("APP_DAEMON_LOG_PRETTY_PRINT", "false")
	t.Setenv("APP_DAEMON_LOG_INCLUDE_CALLER", "true")

	c := NewConfigFromEnv("APP")
	assert.Equal("debug", c.Daemon.Level)
	assert.False(c.Daemon.PrettyPrint)
	assert.True(c.Daemon.IncludeCaller)
}

func TestNewLoggerConfig_CoercesEmptyAndInvalidLevels(t *testing.T) {
	assert := asrt.New(t)

	c := NewLoggerConfig("", true, false, true)
	assert.Equal(zerolog.LevelInfoValue, c.Level, "empty level coerced to info")

	c2 := NewLoggerConfig("not-a-level", false, false, true)
	assert.Equal(zerolog.LevelInfoValue, c2.Level, "unrecognised level coerced to info")

	c3 := NewLoggerConfig("DEBUG", false, false, true)
	assert.Equal("debug", c3.Level, "level lower-cased and trimmed")
}

func TestConfig_Validate(t *testing.T) {
	assert := asrt.New(t)

	assert.NoError((*Config)(nil).Validate(), "nil config validates")
	assert.NoError(DefaultLoggingConfig().Validate(), "defaults validate")

	// Direct construction bypasses NewLoggerConfig's coercion, so a bad
	// level survives until Validate catches it.
	bad := &Config{Daemon: &LoggerConfig{Enabled: true, Level: "warning"}}
	err := bad.Validate()
	assert.Error(err)
	assert.Contains(err.Error(), "warning")

	// Empty level is permitted (NewLoggerConfig coerces elsewhere).
	emptyLevel := &Config{Daemon: &LoggerConfig{Enabled: true, Level: ""}}
	assert.NoError(emptyLevel.Validate())

	// Nil sub-logger is permitted.
	emptySub := &Config{Daemon: nil}
	assert.NoError(emptySub.Validate())
}

func TestLoggerConfig_FromEnv(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_DAEMON_LOG_ENABLED", "true")
	t.Setenv("APP_DAEMON_LOG_LEVEL", "trace")
	t.Setenv("APP_DAEMON_LOG_PRETTY_PRINT", "true")
	t.Setenv("APP_DAEMON_LOG_INCLUDE_CALLER", "true")

	lc := &LoggerConfig{}
	lc.FromEnv("APP_DAEMON_LOG")

	assert.True(lc.Enabled)
	assert.Equal("trace", lc.Level)
	assert.True(lc.PrettyPrint)
	assert.True(lc.IncludeCaller)
}

func TestParseLevelAndIsValidLevel(t *testing.T) {
	assert := asrt.New(t)

	assert.True(isValidLevel("info"))
	assert.True(isValidLevel("trace"))
	assert.False(isValidLevel("loud"))

	assert.Equal(zerolog.InfoLevel, parseLevel("info"))
	assert.Equal(zerolog.NoLevel, parseLevel("loud"), "unrecognised level falls back to NoLevel")
}

func TestLoggerConfig_MarshalZerologObject_Smoke(t *testing.T) {
	cfg := DefaultLoggingConfig().Daemon
	logger := zerolog.New(io.Discard)
	asrt.NotPanics(t, func() {
		logger.Info().EmbedObject(cfg).Send()
	})
}

func TestInitialize_DoesNotPanic(t *testing.T) {
	// Initialize sets package-level loggers; just confirm it runs cleanly
	// for both default and disabled configs.
	asrt.NotPanics(t, func() {
		Initialize(DefaultLoggingConfig())
	})

	disabled := &Config{Daemon: &LoggerConfig{Enabled: false}}
	asrt.NotPanics(t, func() {
		Initialize(disabled)
	})
}
