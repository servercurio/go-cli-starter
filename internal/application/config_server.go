package application

import (
	"errors"

	"github.com/servercurio/go-echo-starter/internal/env"
)

// ServerConfig groups every server-facing configuration block: the HTTP
// and TLS listeners and the cross-cutting policies (CORS, security
// headers, CSRF, rate limiting) layered over them.
type ServerConfig struct {
	Http      *HttpConfig      `yaml:"http" json:"http"`
	Https     *TlsConfig       `yaml:"https" json:"https"`
	Cors      *CorsConfig      `yaml:"cors" json:"cors"`
	Security  *SecurityConfig  `yaml:"security" json:"security"`
	Csrf      *CsrfConfig      `yaml:"csrf" json:"csrf"`
	RateLimit *RateLimitConfig `yaml:"rateLimit" json:"rateLimit"`
}

// FromEnv fans out env-var hydration to each child config under its
// corresponding sub-prefix.
func (c *ServerConfig) FromEnv(prefix string) {
	c.Http.FromEnv(env.AddPrefix(prefix, "http"))
	c.Https.FromEnv(env.AddPrefix(prefix, "https"))
	c.Cors.FromEnv(env.AddPrefix(prefix, "cors"))
	c.Security.FromEnv(env.AddPrefix(prefix, "security"))
	c.Csrf.FromEnv(env.AddPrefix(prefix, "csrf"))
	c.RateLimit.FromEnv(env.AddPrefix(prefix, "rate_limit"))
}

// Validate aggregates the per-subsystem validators with errors.Join so an
// operator sees every config issue at once rather than fixing them one
// boot at a time.
func (c *ServerConfig) Validate() error {
	if c == nil {
		return nil
	}
	return errors.Join(
		c.Http.Validate(),
		c.Https.Validate(),
		c.Cors.Validate(),
		c.Security.Validate(),
		c.Csrf.Validate(),
		c.RateLimit.Validate(),
	)
}

// DefaultServerConfig returns a ServerConfig populated with each child's
// defaults: HTTP enabled, HTTPS off, CORS empty, security headers at
// Mozilla's "A" baseline, CSRF off, rate limit off.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Http:      DefaultHttpConfig(),
		Https:     DefaultTlsConfig(),
		Cors:      DefaultCorsConfig(),
		Security:  DefaultSecurityConfig(),
		Csrf:      DefaultCsrfConfig(),
		RateLimit: DefaultRateLimitConfig(),
	}
}
