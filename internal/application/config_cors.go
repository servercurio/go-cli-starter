package application

import (
	"errors"
	"strings"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// CorsConfig captures the cross-origin policy applied to both the HTTP and
// HTTPS servers.
//
// The starter ships with empty AllowOrigins by default, which disables CORS
// entirely. Downstream consumers must opt in explicitly. This is deliberate:
// once the server starts handling auth/cookies, a permissive CORS default
// becomes a cross-origin leakage vector.
type CorsConfig struct {
	// AllowOrigins is the list of origins allowed to issue cross-origin
	// requests. When empty, the CORS middleware adds no headers and the
	// browser will block cross-origin requests.
	AllowOrigins []string `yaml:"allowOrigins" json:"allowOrigins"`

	// AllowMethods is the list of HTTP methods the browser may use in a
	// cross-origin request. When empty, defaults to GET, HEAD, PUT, PATCH,
	// POST, DELETE.
	AllowMethods []string `yaml:"allowMethods" json:"allowMethods"`

	// AllowHeaders is the list of request headers the browser may include in
	// a cross-origin request.
	AllowHeaders []string `yaml:"allowHeaders" json:"allowHeaders"`

	// AllowCredentials indicates whether the browser should expose response
	// to frontend JavaScript when the request includes credentials. Pairs
	// with explicit AllowOrigins (wildcard cannot be used with credentials).
	AllowCredentials bool `yaml:"allowCredentials" json:"allowCredentials"`

	// MaxAge is the number of seconds a browser may cache the preflight
	// response. Zero leaves it unset.
	MaxAge int `yaml:"maxAge" json:"maxAge"`
}

// Enabled reports whether at least one origin has been configured. When false,
// the CORS middleware should be skipped entirely.
func (c *CorsConfig) Enabled() bool {
	return c != nil && len(c.AllowOrigins) > 0
}

// FromEnv hydrates the CORS fields from environment variables under
// prefix. List-valued fields (origins, methods, headers) are accepted as
// comma-separated strings and split on the wire.
func (c *CorsConfig) FromEnv(prefix string) {
	var allowOrigins string
	env.SetStringValue(prefix, "allow_origins", &allowOrigins)
	if allowOrigins = strings.TrimSpace(allowOrigins); allowOrigins != "" {
		c.AllowOrigins = splitAndTrim(allowOrigins)
	}

	var allowMethods string
	env.SetStringValue(prefix, "allow_methods", &allowMethods)
	if allowMethods = strings.TrimSpace(allowMethods); allowMethods != "" {
		c.AllowMethods = splitAndTrim(allowMethods)
	}

	var allowHeaders string
	env.SetStringValue(prefix, "allow_headers", &allowHeaders)
	if allowHeaders = strings.TrimSpace(allowHeaders); allowHeaders != "" {
		c.AllowHeaders = splitAndTrim(allowHeaders)
	}

	env.SetBoolValue(prefix, "allow_credentials", &c.AllowCredentials)
	env.SetIntValue(prefix, "max_age", &c.MaxAge)
}

// MarshalZerologObject writes the CORS configuration into e for the
// startup-log notifier.
func (c *CorsConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Strs("allowOrigins", c.AllowOrigins).
		Strs("allowMethods", c.AllowMethods).
		Strs("allowHeaders", c.AllowHeaders).
		Bool("allowCredentials", c.AllowCredentials).
		Int("maxAge", c.MaxAge).
		Bool("enabled", c.Enabled())
}

// Validate rejects the spec-illegal combination of `allowCredentials=true`
// with a wildcard origin. Browsers refuse this combination at request time,
// so a daemon configured this way would silently fail every cross-origin
// request that needed credentials. Refuse at startup so operators see the
// problem in the boot log instead of a UI bug ticket.
func (c *CorsConfig) Validate() error {
	if c == nil || !c.Enabled() {
		return nil
	}
	if c.AllowCredentials {
		for _, o := range c.AllowOrigins {
			if strings.TrimSpace(o) == "*" {
				return errors.New("cors: allowCredentials cannot be combined with wildcard '*' origin (browsers will block such responses)")
			}
		}
	}
	return nil
}

// DefaultCorsConfig returns an empty CORS policy — no origins, no
// methods, no headers, credentials off. Effectively disables the CORS
// middleware until the consumer opts in by populating AllowOrigins.
func DefaultCorsConfig() *CorsConfig {
	return &CorsConfig{
		AllowOrigins:     []string{},
		AllowMethods:     []string{},
		AllowHeaders:     []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}
}

// splitAndTrim splits s on commas, trims whitespace from each element, and
// drops empty entries. Used to convert comma-separated env-var lists into
// the slice-of-string form CORS uses internally.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
