package logging

import (
	"log"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/version"
)

// AsStdLogger wraps a zerolog.Logger so it can be passed to APIs that expect
// the standard library *log.Logger (e.g. goose's SetLogger). Each Write call
// from the std logger is emitted as a single zerolog event at the wrapped
// logger's current level.
func AsStdLogger(logger zerolog.Logger) *log.Logger {
	return log.New(logger, "", 0)
}

// NotifyDaemonStartup (re)initializes the logging system from cfg and emits
// the daemon's startup banner with embedded version and commit information.
// Called once early in main, before any other logging.
func NotifyDaemonStartup(name string, cfg *Config) {
	Initialize(cfg)

	Daemon.Info().
		Str("version", version.Number()).
		Str("commit", version.Commit()).
		Msgf("%s daemon started", name)
}

// NotifyDaemonLoggingStartup (re)initializes the logging system from cfg and
// emits a structured event describing the resolved daemon-logger settings.
// Called after Configure once env-var overrides have been applied so
// operators see the effective configuration.
func NotifyDaemonLoggingStartup(cfg *Config) {
	Initialize(cfg)

	Daemon.Info().
		EmbedObject(cfg.Daemon).
		Msg("daemon logging")
}

// NotifyHttpLoggingStartup (re)initializes the logging system from cfg and
// emits a structured event describing the resolved HTTP-access-logger
// settings.
func NotifyHttpLoggingStartup(cfg *Config) {
	Initialize(cfg)

	Daemon.Info().
		EmbedObject(cfg.HttpAccess).
		Msg("http access logging")
}
