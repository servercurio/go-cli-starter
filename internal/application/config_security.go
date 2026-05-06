package application

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// SecurityConfig captures the response-header policy applied by the
// hardening middleware (`echo/v5/middleware.SecureWithConfig`). The
// X-Frame-Options / X-Content-Type-Options / X-XSS-Protection trio is
// always emitted using Echo's defaults — only the bigger-blast-radius
// fields (HSTS, CSP, Referrer-Policy) are surfaced here, since those
// either depend on the deployment topology or risk breaking downstream
// pages if the wrong value ships.
type SecurityConfig struct {
	// HSTSMaxAge is the `max-age` (seconds) emitted in the
	// Strict-Transport-Security header. The header is only sent on TLS
	// requests (or when the inbound `X-Forwarded-Proto` is `https`), so
	// it's safe to leave a non-zero default even on the plain HTTP
	// listener — the middleware suppresses HSTS unless the request is
	// secure end-to-end.
	//
	// Set to 0 to suppress HSTS entirely.
	HSTSMaxAge int `yaml:"hstsMaxAge" json:"hstsMaxAge"`

	// HSTSExcludeSubdomains drops the `includeSubDomains` directive when
	// true. Default false matches Mozilla's recommended posture.
	HSTSExcludeSubdomains bool `yaml:"hstsExcludeSubdomains" json:"hstsExcludeSubdomains"`

	// HSTSPreloadEnabled adds the `preload` directive. Default false —
	// preload submission is one-way and removal is slow, so consumers
	// must opt in deliberately.
	HSTSPreloadEnabled bool `yaml:"hstsPreloadEnabled" json:"hstsPreloadEnabled"`

	// ContentSecurityPolicy is sent verbatim in the Content-Security-Policy
	// header when non-empty. Default "" — a useful CSP is application-
	// specific and a wrong default (e.g. `default-src 'self'`) breaks
	// the bundled Swagger UI's inline scripts.
	ContentSecurityPolicy string `yaml:"contentSecurityPolicy" json:"contentSecurityPolicy"`

	// ReferrerPolicy is sent verbatim in the Referrer-Policy header when
	// non-empty. Common production values: `no-referrer`,
	// `strict-origin-when-cross-origin`. Default "".
	ReferrerPolicy string `yaml:"referrerPolicy" json:"referrerPolicy"`
}

// FromEnv hydrates the security-headers fields from environment variables
// under prefix.
func (c *SecurityConfig) FromEnv(prefix string) {
	env.SetIntValue(prefix, "hsts_max_age", &c.HSTSMaxAge)
	env.SetBoolValue(prefix, "hsts_exclude_subdomains", &c.HSTSExcludeSubdomains)
	env.SetBoolValue(prefix, "hsts_preload_enabled", &c.HSTSPreloadEnabled)
	env.SetStringValue(prefix, "content_security_policy", &c.ContentSecurityPolicy)
	env.SetStringValue(prefix, "referrer_policy", &c.ReferrerPolicy)
}

// MarshalZerologObject writes the security-headers configuration into e
// for the startup-log notifier.
func (c *SecurityConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("hstsMaxAge", c.HSTSMaxAge).
		Bool("hstsExcludeSubdomains", c.HSTSExcludeSubdomains).
		Bool("hstsPreloadEnabled", c.HSTSPreloadEnabled).
		Str("contentSecurityPolicy", c.ContentSecurityPolicy).
		Str("referrerPolicy", c.ReferrerPolicy)
}

// Validate rejects nonsensical security-header values. HSTS max-age cannot
// be negative; the Echo middleware would emit a malformed header otherwise.
func (c *SecurityConfig) Validate() error {
	if c == nil {
		return nil
	}
	if c.HSTSMaxAge < 0 {
		return fmt.Errorf("security: hstsMaxAge must be non-negative, got %d", c.HSTSMaxAge)
	}
	return nil
}

// DefaultSecurityConfig returns the starter's default header posture:
// HSTS enabled with a 1-year max-age (no preload, includes subdomains),
// no CSP, no Referrer-Policy. Matches Mozilla Observatory's "A" baseline
// without the irreversibility risk of preload.
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: false,
		HSTSPreloadEnabled:    false,
		ContentSecurityPolicy: "",
		ReferrerPolicy:        "",
	}
}
