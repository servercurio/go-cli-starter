package application

import (
	"context"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joomcode/errorx"
	"github.com/labstack/echo/v5"
	mw "github.com/labstack/echo/v5/middleware"
	"github.com/servercurio/go-echo-starter/internal/config"
	"github.com/servercurio/go-echo-starter/internal/health"
	"github.com/servercurio/go-echo-starter/internal/logging"
	"github.com/servercurio/go-echo-starter/internal/router"
)

// Default daemon identity strings. Embedded into Application by
// NewApplication; cmd/daemon overrides them on the returned value when the
// downstream binary needs a different name or env-var prefix.
const (
	defaultName              = "appsvrd"
	defaultEnvPrefix         = "APP"
	defaultConfigName        = "appsvrd"
	defaultConfigPathElement = "appsvr"
)

// Application is the daemon's top-level lifecycle owner: it holds the loaded
// configuration, the HTTP and TLS Echo servers, the registered modules, the
// shared health registry, and the readiness flag toggled by Start /
// shutdown. The zero value is not usable — construct via NewApplication.
type Application struct {
	Name              string
	ConfigFileName    string
	EnvVariablePrefix string

	config     *Config
	middleware []echo.MiddlewareFunc
	httpServer *echo.Echo
	tlsServer  *echo.Echo

	userHomeDirectory string
	certificate       *InMemoryCertificate
	modules           map[string]router.Module
	healthRegistry    *health.Registry

	ready atomic.Bool
}

// IsReady reports whether the application is currently accepting traffic. It
// returns true between the point Start has launched the server goroutines and
// the point shutdown is initiated.
func (app *Application) IsReady() bool {
	return app.ready.Load()
}

// NewApplication returns an Application initialised with sensible defaults
// (name, env prefix, both Echo servers constructed, health registry and
// module map empty). Logging is brought up early using env-var-resolved
// settings so subsequent boot steps can emit structured logs immediately.
func NewApplication(cfg *Config) *Application {
	app := &Application{
		Name:              defaultName,
		ConfigFileName:    defaultConfigName,
		EnvVariablePrefix: defaultEnvPrefix,
		config:            cfg,
		healthRegistry:    health.NewRegistry(),
		httpServer:        echo.New(),
		tlsServer:         echo.New(),
		modules:           make(map[string]router.Module),
	}

	// Initialize the logging configuration early to avoid missing any critical logs
	loggingCfg := logging.NewConfigFromEnv(app.EnvVariablePrefix)
	logging.NotifyDaemonStartup(app.Name, loggingCfg)

	return app
}

// buildMiddleware assembles the global middleware stack from the loaded
// configuration. Called from Initialize() once Configure() has populated
// app.config — must run before configureHttpServer / configureTlsServer
// so the slice is ready when those methods call e.Use(app.middleware...).
//
// Lifted out of NewApplication because the Secure middleware needs the
// SecurityConfig values (HSTS, CSP, Referrer-Policy), which aren't
// available until config files + env vars have been resolved.
func (app *Application) buildMiddleware() {
	sec := app.config.Server.Security
	if sec == nil {
		sec = DefaultSecurityConfig()
	}

	app.middleware = []echo.MiddlewareFunc{
		mw.Recover(),
		mw.RequestID(),
		mw.GzipWithConfig(mw.GzipConfig{
			Level:     5,
			MinLength: 2 * 1024,
		}),
		logging.EchoMiddleware(),
		mw.SecureWithConfig(mw.SecureConfig{
			XSSProtection:         mw.DefaultSecureConfig.XSSProtection,
			ContentTypeNosniff:    mw.DefaultSecureConfig.ContentTypeNosniff,
			XFrameOptions:         mw.DefaultSecureConfig.XFrameOptions,
			HSTSMaxAge:            sec.HSTSMaxAge,
			HSTSExcludeSubdomains: sec.HSTSExcludeSubdomains,
			HSTSPreloadEnabled:    sec.HSTSPreloadEnabled,
			ContentSecurityPolicy: sec.ContentSecurityPolicy,
			ReferrerPolicy:        sec.ReferrerPolicy,
		}),
	}

	// Rate limit goes early so 429s short-circuit before any heavier
	// per-request work. Skipped when disabled — see RateLimitMiddleware.
	if rl := RateLimitMiddleware(app.config.Server.RateLimit); rl != nil {
		app.middleware = append(app.middleware, rl)
	}

	// CSRF rides at the end of the chain so the security headers are
	// already set before any 403 short-circuits the request. Skipped
	// entirely when disabled (the default) — a nil append is cheaper
	// than a no-op middleware and keeps the chain shorter.
	if csrf := CsrfMiddleware(app.config.Server.Csrf); csrf != nil {
		app.middleware = append(app.middleware, csrf)
	}
}

// Configure loads configuration from /etc, the user's home directory, and
// the working directory (in that order, so later sources override earlier
// ones), applies env-var overrides, emits the resolved config to the daemon
// log, and runs Validate. Returns the joined validation error so the daemon
// can refuse to boot with a single message listing every issue.
func (app *Application) Configure() error {
	configLocations := configSearchPaths()

	logging.Daemon.
		Trace().
		Strs("paths", configLocations).
		Strs("fileNames", config.FileNameVariants(app.ConfigFileName)).
		Msg("searching for config files")

	// Load the server configuration from config files
	if err := config.FromPaths(app.config, app.ConfigFileName, configLocations...); err != nil {
		logging.Daemon.
			Warn().
			Err(err).
			Strs("paths", configLocations).
			Strs("fileNames", config.FileNameVariants(app.ConfigFileName)).
			Msg("error loading server config")
	}

	// Load the server configuration from environment variables
	app.config.FromEnv(app.EnvVariablePrefix)

	logging.NotifyDaemonLoggingStartup(app.config.Logging)
	logging.NotifyHttpLoggingStartup(app.config.Logging)

	NotifyHttpServerConfig(app.config.Server.Http)
	NotifyHttpsServerConfig(app.config.Server.Https)
	NotifyCorsConfig(app.config.Server.Cors)
	NotifySecurityConfig(app.config.Server.Security)
	NotifyCsrfConfig(app.config.Server.Csrf)
	NotifyRateLimitConfig(app.config.Server.RateLimit)
	NotifyProxySupportConfig(app.config.Proxy)
	NotifyDatabaseConfig(app.config.Database)
	NotifyOpenAPIConfig(app.config.OpenAPI)

	if err := app.config.Validate(); err != nil {
		logging.Daemon.Error().Err(err).Msg("invalid configuration; daemon will not start")
		return err
	}

	return nil
}

// Initialize stands up every subsystem in the correct order: home dir,
// middleware chain, HTTP/TLS servers, proxy IP-extractor, database (when
// enabled), health checks, OpenAPI module (when enabled), and finally
// routing for every module the caller has registered. Each step logs and
// returns on hard failure; soft failures (proxy, routing) are demoted to
// warnings so the daemon still serves what it can.
func (app *Application) Initialize() error {
	app.resolveHomeDirectory()

	app.buildMiddleware()

	if err := app.configureHttpServer(); err != nil {
		return err
	}

	if err := app.configureTlsServer(); err != nil {
		return err
	}

	if err := app.configureProxySupport(); err != nil {
		logging.Daemon.
			Warn().
			Err(err).
			Msg("invalid proxy support configuration")
	}

	if err := app.initializeDatabase(); err != nil {
		return err
	}

	app.registerHealthChecks()

	if err := app.initializeOpenAPI(); err != nil {
		return err
	}

	if err := app.initializeRouting(); err != nil {
		logging.Daemon.
			Warn().
			Err(err).
			Msg("failed to initialize routing")
	}

	return nil
}

// RegisterModule adds a top-level routing module to the Application. Must be
// called before Initialize attaches modules to Echo (or, for modules
// registered by Initialize itself like openapi/swagger, before the next
// Initialize phase consumes them). Rejects nil and refuses to overwrite an
// existing registration with the same Name.
func (app *Application) RegisterModule(m router.Module) error {
	if m == nil {
		return errorx.IllegalArgument.New("module argument must not be nil")
	}

	if _, exists := app.modules[m.Name()]; exists {
		logging.Daemon.
			Warn().
			Str("name", m.Name()).
			Str("id", m.Id()).
			Msg("application module already registered")
		return errorx.IllegalState.New("application module '%s' already registered", m.Name())
	}

	app.modules[m.Name()] = m
	return nil
}

// Start brings up the HTTP and TLS server goroutines, flips the readiness
// flag, then blocks until one of the configured shutdown signals arrives.
// On shutdown it cancels both servers, waits for them to drain (bounded by
// the larger of the two ShutdownTimeouts plus a five-second grace), closes
// the database connection, and returns. The int return is the suggested
// process exit code; today it's always 0 unless an unrecoverable error
// surfaced upstream.
func (app *Application) Start() (int, error) {
	signalCtx, signalCancel :=
		signal.NotifyContext(context.Background(), shutdownSignals...)
	defer signalCancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		app.startHttpServer(signalCtx)
	}()
	go func() {
		defer wg.Done()
		app.startTlsServer(signalCtx)
	}()

	app.ready.Store(true)

	<-signalCtx.Done()
	app.ready.Store(false)

	// Echo v5 shuts each server down internally when signalCtx cancels (using
	// the GracefulTimeout we set on StartConfig). Wait for the goroutines to
	// finish, but bound the wait so a stalled handler can't hang the process.
	maxShutdown := app.config.Server.Http.ShutdownTimeout
	if app.config.Server.Https != nil && app.config.Server.Https.ShutdownTimeout > maxShutdown {
		maxShutdown = app.config.Server.Https.ShutdownTimeout
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.Daemon.Info().Msg("server goroutines shut down cleanly")
	case <-time.After(maxShutdown + 5*time.Second):
		logging.Daemon.Warn().
			Dur("timeout", maxShutdown+5*time.Second).
			Msg("server goroutines did not return within shutdown timeout")
	}

	app.shutdownDatabase()

	return 0, nil
}
