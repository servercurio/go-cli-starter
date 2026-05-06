package application

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// CsrfConfig captures the cross-site request forgery middleware policy.
//
// Like CorsConfig the starter ships disabled by default — CSRF only makes
// sense once the application uses cookies/sessions, and a wrong default
// either breaks every API client (Enabled with no opt-in) or gives a
// false sense of security (Enabled with permissive lookup). Downstream
// consumers flip Enabled when they introduce session state.
type CsrfConfig struct {
	// Enabled controls whether the middleware is wired into the global
	// chain. When false, the middleware is omitted entirely (no per-
	// request work) — matches the CorsConfig.Enabled() convention.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// TokenLookup follows Echo's "<source>:<name>" syntax (or comma-
	// separated alternatives). Default `header:X-CSRF-Token`. Common
	// alternatives:
	//   - `form:csrf` for HTML form posts
	//   - `header:X-CSRF-Token,query:csrf` for SPAs that may send via either
	TokenLookup string `yaml:"tokenLookup" json:"tokenLookup"`

	// CookieName is the cookie that stores the CSRF token. Default `_csrf`.
	CookieName string `yaml:"cookieName" json:"cookieName"`

	// CookieDomain optionally scopes the cookie to a domain. Empty leaves
	// it as the request host (same-origin default).
	CookieDomain string `yaml:"cookieDomain" json:"cookieDomain"`

	// CookiePath optionally scopes the cookie to a path prefix. Empty
	// leaves it at `/`.
	CookiePath string `yaml:"cookiePath" json:"cookiePath"`

	// CookieMaxAge is the cookie lifetime in seconds. Default 86400 (one
	// day) matches Echo's default. Zero issues a session cookie.
	CookieMaxAge int `yaml:"cookieMaxAge" json:"cookieMaxAge"`

	// CookieSecure restricts the cookie to HTTPS responses. Default false
	// because the starter is also reachable over plain HTTP for local
	// dev. Production deployments behind TLS should flip this to true.
	CookieSecure bool `yaml:"cookieSecure" json:"cookieSecure"`

	// CookieHTTPOnly hides the cookie from JavaScript. Default true —
	// exposing it via document.cookie defeats most of the protection.
	CookieHTTPOnly bool `yaml:"cookieHTTPOnly" json:"cookieHTTPOnly"`

	// CookieSameSite controls the SameSite directive. Accepts (case-
	// insensitive): "default", "lax", "strict", "none". Default "" leaves
	// it as Echo's SameSiteDefaultMode (browser default — typically Lax).
	CookieSameSite string `yaml:"cookieSameSite" json:"cookieSameSite"`
}

// FromEnv hydrates the CSRF fields from environment variables under
// prefix (e.g. <prefix>_ENABLED, <prefix>_TOKEN_LOOKUP).
func (c *CsrfConfig) FromEnv(prefix string) {
	env.SetBoolValue(prefix, "enabled", &c.Enabled)
	env.SetStringValue(prefix, "token_lookup", &c.TokenLookup)
	env.SetStringValue(prefix, "cookie_name", &c.CookieName)
	env.SetStringValue(prefix, "cookie_domain", &c.CookieDomain)
	env.SetStringValue(prefix, "cookie_path", &c.CookiePath)
	env.SetIntValue(prefix, "cookie_max_age", &c.CookieMaxAge)
	env.SetBoolValue(prefix, "cookie_secure", &c.CookieSecure)
	env.SetBoolValue(prefix, "cookie_http_only", &c.CookieHTTPOnly)
	env.SetStringValue(prefix, "cookie_same_site", &c.CookieSameSite)
}

// MarshalZerologObject writes the CSRF configuration into e for the
// startup-log notifier.
func (c *CsrfConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Bool("enabled", c.Enabled).
		Str("tokenLookup", c.TokenLookup).
		Str("cookieName", c.CookieName).
		Str("cookieDomain", c.CookieDomain).
		Str("cookiePath", c.CookiePath).
		Int("cookieMaxAge", c.CookieMaxAge).
		Bool("cookieSecure", c.CookieSecure).
		Bool("cookieHTTPOnly", c.CookieHTTPOnly).
		Str("cookieSameSite", c.CookieSameSite)
}

// Validate rejects unrecognised SameSite strings — the alternative would
// be Echo silently treating an unknown value as the default mode, which
// makes "I set CookieSameSite=stricct and nothing changed" debugging
// painful. Empty is valid (uses Echo's default).
func (c *CsrfConfig) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	if _, err := parseSameSite(c.CookieSameSite); err != nil {
		return err
	}
	return nil
}

// DefaultCsrfConfig returns a disabled-by-default policy whose other
// fields match Echo's DefaultCSRFConfig. Consumers flip Enabled to opt
// in; the rest of the values remain a sensible starting point.
func DefaultCsrfConfig() *CsrfConfig {
	return &CsrfConfig{ //nolint:gosec // G101 false positive: CookieName is the cookie's name, not a credential
		Enabled:        false,
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "_csrf",
		CookieDomain:   "",
		CookiePath:     "",
		CookieMaxAge:   86400,
		CookieSecure:   false,
		CookieHTTPOnly: true,
		CookieSameSite: "",
	}
}

// parseSameSite maps the user-facing SameSite string to the http.SameSite
// constant Echo expects. Empty returns SameSiteDefaultMode so consumers
// who don't set the field inherit Echo's default. Unknown strings return
// an error rather than coercing silently.
func parseSameSite(s string) (http.SameSite, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return http.SameSiteDefaultMode, nil
	case "default":
		return http.SameSiteDefaultMode, nil
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return 0, &sameSiteError{value: s}
	}
}

// sameSiteError carries the original (unrecognised) SameSite string back
// to Validate so the operator-facing error message contains the value
// they actually typed.
type sameSiteError struct{ value string }

// Error implements the error interface.
func (e *sameSiteError) Error() string {
	return "csrf: cookieSameSite must be one of {default, lax, strict, none}, got " + e.value
}
