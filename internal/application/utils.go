package application

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joomcode/errorx"
	"github.com/labstack/echo/v5"
	mw "github.com/labstack/echo/v5/middleware"
	"github.com/servercurio/go-cli-starter/internal/database"
	"github.com/servercurio/go-cli-starter/internal/logging"
)

// defaultInsecurePaths lists URI substrings that bypass the
// HTTPS-redirect middleware. Health / readiness endpoints are intentionally
// reachable over plain HTTP so external probes (cloud LBs, kubelet) don't
// have to negotiate TLS just to verify the server is alive.
var defaultInsecurePaths = []string{"api/v1/healthz", "api/v1/readyz"}

// configSearchPaths returns the ordered set of directories scanned for
// config files: /etc/<element> first, then the user's ~/.config/<element>
// (when reachable), then the working directory. Later paths override
// earlier ones, so a per-checkout config beats system defaults.
func configSearchPaths() []string {
	search := []string{fmt.Sprintf("/etc/%s", defaultConfigPathElement)}

	if home, err := os.UserHomeDir(); err == nil {
		search = append(search, absPath(filepath.Join(home, ".config", defaultConfigPathElement)))
	}

	search = append(search, absPath("."))
	return search
}

// absPath returns the absolute form of path, or path unchanged if the
// system can't resolve it (e.g. CWD missing). Used to normalise config
// search paths so they show up unambiguously in the boot log.
func absPath(path string) string {
	if p, err := filepath.Abs(path); err == nil {
		return p
	}

	return path
}

// NotifyHttpServerConfig emits the resolved HTTP listener configuration to
// the daemon log so operators see exactly what's running.
func NotifyHttpServerConfig(cfg *HttpConfig) {
	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("http server configuration")
}

// NotifyHttpsServerConfig emits the resolved TLS listener configuration to
// the daemon log when TLS is enabled. No-op for disabled or nil configs.
func NotifyHttpsServerConfig(cfg *TlsConfig) {
	if cfg == nil || !cfg.Enabled {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("https server configuration")
}

// NotifyCorsConfig emits the resolved CORS policy to the daemon log.
// No-op for a nil config.
func NotifyCorsConfig(cfg *CorsConfig) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("cors configuration")
}

// NotifySecurityConfig emits the resolved security-headers policy to the
// daemon log. No-op for a nil config.
func NotifySecurityConfig(cfg *SecurityConfig) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("security headers configuration")
}

// NotifyCsrfConfig emits the resolved CSRF policy to the daemon log. No-op
// for a nil config.
func NotifyCsrfConfig(cfg *CsrfConfig) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("csrf configuration")
}

// NotifyRateLimitConfig emits the resolved rate-limit policy to the daemon
// log when either the per-IP limiter or the connection cap is active.
// No-op when both knobs are at their disabled defaults.
func NotifyRateLimitConfig(cfg *RateLimitConfig) {
	if cfg == nil || !cfg.Configured() {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("rate limit configuration")
}

// NotifyProxySupportConfig emits the resolved proxy IP-extraction
// configuration to the daemon log. No-op for a nil config.
func NotifyProxySupportConfig(cfg *ProxyConfig) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("proxy support configuration")
}

// NotifyDatabaseConfig emits the resolved database configuration (with the
// DSN credential masked) to the daemon log. No-op for a nil config.
func NotifyDatabaseConfig(cfg *database.Config) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("database configuration")
}

// NotifyOpenAPIConfig emits the resolved OpenAPI / Swagger UI configuration
// to the daemon log. No-op for a nil config.
func NotifyOpenAPIConfig(cfg *OpenAPIConfig) {
	if cfg == nil {
		return
	}

	logging.Daemon.Info().
		EmbedObject(cfg).
		Msg("openapi configuration")
}

// CorsMiddleware returns CORS middleware configured from cfg, or nil when CORS
// is disabled (no AllowOrigins). Callers should skip Use() when nil is
// returned so the middleware chain doesn't include a no-op handler.
func CorsMiddleware(cfg *CorsConfig) echo.MiddlewareFunc {
	if !cfg.Enabled() {
		return nil
	}
	return mw.CORSWithConfig(mw.CORSConfig{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}

// RateLimitMiddleware returns the Echo rate-limiter wired from cfg, or
// nil when disabled. Keys by RealIP() (Echo's default extractor) so the
// limit applies to the actual client even behind a trusted proxy — the
// proxy support added in #17 sets up the IPExtractor upstream of this
// middleware. Echo's memory store falls back to ceil(Rate) when Burst
// is zero, so a one-knob config (just Rate) Just Works.
func RateLimitMiddleware(cfg *RateLimitConfig) echo.MiddlewareFunc {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	store := mw.NewRateLimiterMemoryStoreWithConfig(mw.RateLimiterMemoryStoreConfig{
		Rate:      cfg.Rate,
		Burst:     cfg.Burst,
		ExpiresIn: cfg.ExpiresIn,
	})
	return mw.RateLimiterWithConfig(mw.RateLimiterConfig{
		Store: store,
	})
}

// CsrfMiddleware returns the Echo CSRF middleware wired from cfg, or nil
// when disabled. Mirrors CorsMiddleware so buildMiddleware can skip the
// no-op case by checking for nil. SameSite is parsed via the validated
// helper — Validate() already rejected unknown values at Configure time,
// so any error here is a programming bug; we fall back to the default
// mode rather than panic.
func CsrfMiddleware(cfg *CsrfConfig) echo.MiddlewareFunc {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	sameSite, _ := parseSameSite(cfg.CookieSameSite)
	return mw.CSRFWithConfig(mw.CSRFConfig{
		TokenLookup:    cfg.TokenLookup,
		CookieName:     cfg.CookieName,
		CookieDomain:   cfg.CookieDomain,
		CookiePath:     cfg.CookiePath,
		CookieMaxAge:   cfg.CookieMaxAge,
		CookieSecure:   cfg.CookieSecure,
		CookieHTTPOnly: cfg.CookieHTTPOnly,
		CookieSameSite: sameSite,
	})
}

// parseByteSize parses a human-readable byte-size string (e.g. "1MB", "500KB",
// "2GB", "1024") into a byte count. The suffix is case-insensitive; bare
// numbers are treated as bytes. Returns an error for unrecognised suffixes or
// non-numeric magnitudes.
func parseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errorx.IllegalArgument.New("empty byte size")
	}

	upper := strings.ToUpper(s)
	mult := int64(1)
	switch {
	case strings.HasSuffix(upper, "GB"):
		mult = 1024 * 1024 * 1024
		upper = strings.TrimSuffix(upper, "GB")
	case strings.HasSuffix(upper, "MB"):
		mult = 1024 * 1024
		upper = strings.TrimSuffix(upper, "MB")
	case strings.HasSuffix(upper, "KB"):
		mult = 1024
		upper = strings.TrimSuffix(upper, "KB")
	case strings.HasSuffix(upper, "B"):
		upper = strings.TrimSuffix(upper, "B")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(upper), 10, 64)
	if err != nil {
		return 0, errorx.IllegalArgument.Wrap(err, "invalid byte size %q", s)
	}
	if n < 0 {
		return 0, errorx.IllegalArgument.New("byte size must be non-negative: %q", s)
	}
	return n * mult, nil
}

// HTTPSRedirectWithConfig returns Echo Pre-middleware that 308-redirects
// plaintext HTTP requests to the equivalent HTTPS URL. Skips requests
// whose path matches defaultInsecurePaths (the health probes), and uses
// the configured Hostname rather than the client-supplied Host header to
// neutralise host-header redirect attacks.
func HTTPSRedirectWithConfig(cfg *TlsConfig) echo.MiddlewareFunc {
	portSpec := ""
	if cfg.Port != 443 {
		portSpec = fmt.Sprintf(":%d", cfg.Port)
	}

	configuredHost := strings.TrimSpace(cfg.Hostname)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if c.Request().TLS != nil {
				return next(c)
			}

			for _, path := range defaultInsecurePaths {
				if strings.Contains(c.Request().URL.Path, path) {
					return next(c)
				}
			}

			// Prefer the configured hostname over the client-supplied Host
			// header. A malicious client can spoof Host to redirect victims
			// to attacker-controlled domains; using the configured hostname
			// pins the redirect target to a value the operator chose.
			hostNoPort := configuredHost
			if hostNoPort == "" {
				hostNoPort = c.Request().Host
				if i := strings.Index(hostNoPort, ":"); i >= 0 {
					hostNoPort = hostNoPort[:i]
				}
			}

			redirectUrl := fmt.Sprintf("https://%s%s%s",
				hostNoPort, portSpec, c.Request().URL.Path)

			if err := c.Redirect(http.StatusPermanentRedirect, redirectUrl); err != nil {
				return err
			}

			return nil
		}
	}
}
