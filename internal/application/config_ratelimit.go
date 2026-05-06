package application

import (
	"errors"
	"time"

	"github.com/rs/zerolog"

	"github.com/servercurio/go-cli-starter/internal/env"
)

// RateLimitConfig captures the request-rate and concurrent-connection
// limits applied to both HTTP and HTTPS servers. Mirrors the CSRF/CORS
// shape: disabled by default so the starter remains a thin scaffold;
// downstream consumers opt in.
//
// Two independent knobs ride together because they protect against the
// same threat (overwhelmed server) at different layers:
//
//   - Rate / Burst is a per-client request-rate limit at the middleware
//     layer keyed off c.RealIP(). Honours APP_TRUSTED_IP_RANGES so it
//     keys by the real client even behind a trusted proxy.
//   - MaxConnections caps total concurrent TCP connections at the
//     listener layer via netutil.LimitListener. Excess connections are
//     accepted only after one closes, providing a hard ceiling that
//     middleware can't reach (TLS handshake, slowloris, etc.).
type RateLimitConfig struct {
	// Enabled toggles the per-IP request rate limiter. When false the
	// middleware is omitted from the chain entirely.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Rate is the steady-state allowed requests per second per client IP.
	// Required when Enabled is true.
	Rate float64 `yaml:"rate" json:"rate"`

	// Burst is the bucket size — clients can make this many requests in
	// one shot before the rate kicks in. Defaults to ceil(Rate) when zero
	// at runtime; a negative value is rejected by Validate.
	Burst int `yaml:"burst" json:"burst"`

	// ExpiresIn is how long a per-client bucket sticks around with no
	// activity before the memory store evicts it. Default 3 minutes
	// matches Echo's `RateLimiterMemoryStoreConfig.ExpiresIn` default.
	ExpiresIn time.Duration `yaml:"expiresIn" json:"expiresIn"`

	// MaxConnections caps simultaneous TCP connections per listener. Zero
	// disables the netutil.LimitListener wrap (unlimited, the default).
	// This is independent of Enabled — operators can cap connections
	// without enabling per-IP rate limiting and vice versa.
	MaxConnections int `yaml:"maxConnections" json:"maxConnections"`
}

// Configured returns true when either knob is active. Used to decide
// whether to emit the configuration log line and to wire the
// LimitListener wrap independently of the request-rate middleware.
func (c *RateLimitConfig) Configured() bool {
	return c != nil && (c.Enabled || c.MaxConnections > 0)
}

// FromEnv hydrates the rate-limit fields from environment variables under
// prefix.
func (c *RateLimitConfig) FromEnv(prefix string) {
	env.SetBoolValue(prefix, "enabled", &c.Enabled)
	env.SetFloatValue(prefix, "rate", &c.Rate)
	env.SetIntValue(prefix, "burst", &c.Burst)
	env.SetDurationValue(prefix, "expires_in", &c.ExpiresIn)
	env.SetIntValue(prefix, "max_connections", &c.MaxConnections)
}

// MarshalZerologObject writes the rate-limit configuration into e for the
// startup-log notifier.
func (c *RateLimitConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Bool("enabled", c.Enabled).
		Float64("rate", c.Rate).
		Int("burst", c.Burst).
		Dur("expiresIn", c.ExpiresIn).
		Int("maxConnections", c.MaxConnections)
}

// Validate rejects misconfigurations that would otherwise produce
// confusing runtime behaviour: a zero rate with the limiter enabled
// would drop every request; a negative burst is meaningless.
func (c *RateLimitConfig) Validate() error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.Enabled && c.Rate <= 0 {
		errs = append(errs, errors.New("rate_limit: rate must be > 0 when enabled"))
	}
	if c.Burst < 0 {
		errs = append(errs, errors.New("rate_limit: burst must be >= 0"))
	}
	if c.ExpiresIn < 0 {
		errs = append(errs, errors.New("rate_limit: expires_in must be >= 0"))
	}
	if c.MaxConnections < 0 {
		errs = append(errs, errors.New("rate_limit: max_connections must be >= 0"))
	}
	return errors.Join(errs...)
}

// DefaultRateLimitConfig returns a fully-disabled policy. ExpiresIn is
// pre-populated so consumers who only flip Enabled and Rate inherit a
// sensible memory-store TTL without having to know it exists.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:        false,
		Rate:           0,
		Burst:          0,
		ExpiresIn:      3 * time.Minute,
		MaxConnections: 0,
	}
}
