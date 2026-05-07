package application

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"

	"github.com/servercurio/go-cli-starter/internal/config"
	"github.com/servercurio/go-cli-starter/internal/health"
	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/pool"
)

// Default identity strings. Embedded into Application by NewApplication;
// callers (typically internal/cli's root command) can override on the
// returned value when the downstream binary needs a different name or
// env-var prefix.
const (
	defaultName              = "appcli"
	defaultEnvPrefix         = "APP"
	defaultConfigName        = "appcli"
	defaultConfigPathElement = "appcli"
)

// Application is the CLI's top-level lifecycle owner: it holds the loaded
// configuration, the optional database pool, the goroutine pool, the shared
// health registry, and the readiness flag toggled by RunUntilSignal. The
// zero value is not usable — construct via NewApplication.
type Application struct {
	Name              string
	ConfigFileName    string
	EnvVariablePrefix string

	config         *Config
	pool           *pool.Pool
	healthRegistry *health.Registry

	ready atomic.Bool
}

// IsReady reports whether the daemon is currently in its serving phase.
// True between the moment RunUntilSignal flips the flag and the moment
// shutdown begins. One-shot commands (Run) leave the flag at false.
func (app *Application) IsReady() bool {
	return app.ready.Load()
}

// HealthRegistry returns the per-Application health registry. Subcommands
// (or downstream consumer code) read it to render a health snapshot.
func (app *Application) HealthRegistry() *health.Registry {
	return app.healthRegistry
}

// Pool returns the shared goroutine pool. nil before Initialize completes;
// non-nil afterwards until Shutdown releases it. Subcommands fan out
// concurrent work through this pool so a single --workers / APP_POOL_SIZE
// knob bounds total in-flight goroutines across every subsystem.
func (app *Application) Pool() *pool.Pool {
	return app.pool
}

// NewApplication returns an Application initialised with sensible defaults
// (name, env prefix, empty health registry). Logging is brought up early
// using env-var-resolved settings so subsequent boot steps can emit
// structured logs immediately.
func NewApplication(cfg *Config) *Application {
	app := &Application{
		Name:              defaultName,
		ConfigFileName:    defaultConfigName,
		EnvVariablePrefix: defaultEnvPrefix,
		config:            cfg,
		healthRegistry:    health.NewRegistry(),
	}

	loggingCfg := logging.NewConfigFromEnv(app.EnvVariablePrefix)
	logging.NotifyDaemonStartup(app.Name, loggingCfg)

	return app
}

// Configure loads configuration from /etc, the user's home directory, and
// the working directory (in that order, so later sources override earlier
// ones), applies env-var overrides, emits the resolved config to the daemon
// log, and runs Validate. Returns the joined validation error so callers
// can refuse to start with a single message listing every issue.
func (app *Application) Configure() error {
	configLocations := configSearchPaths()

	logging.Daemon.
		Trace().
		Strs("paths", configLocations).
		Strs("fileNames", config.FileNameVariants(app.ConfigFileName)).
		Msg("searching for config files")

	if err := config.FromPaths(app.config, app.ConfigFileName, configLocations...); err != nil {
		logging.Daemon.
			Warn().
			Err(err).
			Strs("paths", configLocations).
			Strs("fileNames", config.FileNameVariants(app.ConfigFileName)).
			Msg("error loading config")
	}

	app.config.FromEnv(app.EnvVariablePrefix)

	logging.NotifyDaemonLoggingStartup(app.config.Logging)
	NotifyDatabaseConfig(app.config.Database)
	NotifyPoolConfig(app.config.Pool)

	if err := app.config.Validate(); err != nil {
		logging.Daemon.Error().Err(err).Msg("invalid configuration; refusing to start")
		return err
	}

	return nil
}

// Initialize stands up every subsystem in the correct order: database (when
// enabled), goroutine pool, then health checks. Each step logs and returns
// on hard failure.
func (app *Application) Initialize() error {
	if err := app.initializeDatabase(); err != nil {
		return err
	}

	if err := app.initializePool(); err != nil {
		return err
	}

	app.registerHealthChecks()

	return nil
}

// Run executes a one-shot command body: it flips ready=true, invokes body,
// flips ready=false, and unwinds every subsystem. Use for subcommands that
// do their work and return (e.g. `copy`).
func (app *Application) Run(ctx context.Context, body func(ctx context.Context) error) error {
	app.ready.Store(true)
	defer func() {
		app.ready.Store(false)
		app.shutdown()
	}()

	if body == nil {
		return nil
	}
	return body(ctx)
}

// RunUntilSignal flips ready=true, invokes body in the foreground (typically
// a daemon's long-running loop), and blocks until either body returns or one
// of the configured shutdown signals arrives. body receives a context that is
// cancelled on the first signal so it can unwind cooperatively. After body
// returns, every subsystem is shut down. Use for daemon subcommands.
func (app *Application) RunUntilSignal(ctx context.Context, body func(ctx context.Context) error) error {
	signalCtx, signalCancel := signal.NotifyContext(ctx, shutdownSignals...)
	defer signalCancel()

	app.ready.Store(true)
	defer func() {
		app.ready.Store(false)
		app.shutdown()
	}()

	if body == nil {
		<-signalCtx.Done()
		logging.Daemon.Info().Msg("shutdown signal received; daemon exiting")
		return nil
	}

	err := body(signalCtx)
	if err != nil {
		logging.Daemon.Error().Err(err).Msg("daemon body returned error")
	} else {
		logging.Daemon.Info().Msg("daemon body returned; exiting")
	}
	return err
}

// shutdown unwinds every subsystem in reverse-initialise order. Errors are
// logged but never returned — shutdown should always make progress.
func (app *Application) shutdown() {
	app.shutdownPool()
	app.shutdownDatabase()
}

// configSearchPaths returns the ordered set of directories scanned for
// config files: /etc/<element> first, then the user's
// ~/.config/<element> (when reachable), then the working directory.
// Later paths override earlier ones, so a per-checkout config beats
// system defaults.
func configSearchPaths() []string {
	search := []string{fmt.Sprintf("/etc/%s", defaultConfigPathElement)}

	if home, err := os.UserHomeDir(); err == nil {
		search = append(search, absPath(filepath.Join(home, ".config", defaultConfigPathElement)))
	}

	search = append(search, absPath("."))
	return search
}

// absPath returns the absolute form of path, or path unchanged if the system
// can't resolve it. Used to normalise config search paths so they show up
// unambiguously in the boot log.
func absPath(path string) string {
	if p, err := filepath.Abs(path); err == nil {
		return p
	}
	return path
}
