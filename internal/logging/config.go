package logging

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// Config represents the logging configuration for the application.
type Config struct {
	Daemon *LoggerConfig `yaml:"daemon" json:"daemon"`
}

// FromEnv hydrates Daemon from environment variables prefixed
// "<prefix>_DAEMON_LOG_*".
func (c *Config) FromEnv(prefix string) {
	c.Daemon.FromEnv(env.AddPrefix(prefix, "daemon_log"))
}

// LoggerConfig represents the configuration for a single logger.
type LoggerConfig struct {
	// Enabled indicates whether the logger is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Level is the verbosity of the logging output.
	Level string `yaml:"level" json:"level"`

	// PrettyPrint enables human-readable output.
	PrettyPrint bool `yaml:"prettyPrint" json:"prettyPrint"`

	// IncludeCaller enables caller information in the log output.
	IncludeCaller bool `yaml:"includeCaller" json:"includeCaller"`
}

// MarshalZerologObject implements zerolog.LogObjectMarshaler so a
// LoggerConfig can be embedded in a structured log event via .EmbedObject(),
// used by the startup notifications.
func (l *LoggerConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Bool("enabled", l.Enabled)
	e.Str("logLevel", strings.ToLower(l.Level))
	e.Bool("prettyPrint", l.PrettyPrint)
	e.Bool("includeCaller", l.IncludeCaller)
}

// FromEnv reads the standard logger keys (enabled, level, pretty_print,
// include_caller) under prefix and applies any present values to the
// receiver.
func (l *LoggerConfig) FromEnv(prefix string) {
	env.SetBoolValue(prefix, "enabled", &l.Enabled)
	env.SetStringValue(prefix, "level", &l.Level)
	env.SetBoolValue(prefix, "pretty_print", &l.PrettyPrint)
	env.SetBoolValue(prefix, "include_caller", &l.IncludeCaller)
}

// Validate rejects logger configs whose Level string isn't a recognised
// zerolog level (trace/debug/info/warn/error/fatal/panic). NewLoggerConfig
// silently coerces empty/invalid values to "info", but configs loaded
// directly from a file or env var bypass that path — refusing here means
// "loglevel: warning" (instead of "warn") fails fast instead of producing
// silently-wrong output.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	return errors.Join(
		c.Daemon.Validate("daemon"),
	)
}

// Validate parses the LoggerConfig's Level via zerolog. Empty Level is
// permitted (NewLoggerConfig coerces to info elsewhere); a non-empty but
// unrecognised value is the configuration mistake worth catching.
func (l *LoggerConfig) Validate(name string) error {
	if l == nil {
		return nil
	}
	level := strings.TrimSpace(l.Level)
	if level == "" {
		return nil
	}
	if _, err := zerolog.ParseLevel(strings.ToLower(level)); err != nil {
		return fmt.Errorf("logging.%s.level: %q is not a recognised zerolog level", name, l.Level)
	}
	return nil
}

// DefaultLoggingConfig returns a Config with daemon logging enabled at info
// with pretty-printing on.
func DefaultLoggingConfig() *Config {
	return &Config{
		Daemon: NewLoggerConfig(zerolog.LevelInfoValue, true, false, true),
	}
}

// NewConfigFromEnv returns a defaults-then-env-override Config: starts from
// DefaultLoggingConfig and applies any matching environment variables under
// prefix.
func NewConfigFromEnv(prefix string) *Config {
	cfg := DefaultLoggingConfig()
	cfg.FromEnv(prefix)

	return cfg
}

// NewLoggerConfig builds a LoggerConfig with the given level (coerced to
// "info" if empty or unrecognised so callers don't have to validate
// upstream), pretty-print, caller-info, and enabled flags.
func NewLoggerConfig(level string, prettyPrint, includeCaller, enabled bool) *LoggerConfig {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" || !isValidLevel(level) {
		level = zerolog.LevelInfoValue
	}

	return &LoggerConfig{
		Enabled:       enabled,
		Level:         level,
		PrettyPrint:   prettyPrint,
		IncludeCaller: includeCaller,
	}
}

// isValidLevel reports whether level is a name zerolog can parse.
func isValidLevel(level string) bool {
	if _, err := zerolog.ParseLevel(level); err == nil {
		return true
	}

	return false
}

// parseLevel returns the zerolog.Level for the given name, or zerolog.NoLevel
// when the name is not recognised.
func parseLevel(level string) zerolog.Level {
	if l, err := zerolog.ParseLevel(level); err == nil {
		return l
	}

	return zerolog.NoLevel
}
