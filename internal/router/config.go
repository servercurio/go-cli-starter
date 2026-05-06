package router

import "github.com/servercurio/go-echo-starter/internal/health"

// Config carries cross-cutting dependencies that route/module constructors
// need access to without coupling them to a concrete server implementation.
type Config struct {
	// HealthRegistry is the source of truth for the rich health-check
	// response shape returned by /api/v1/livez, /readyz, and /healthz.
	// Lifecycle owners (e.g. the application daemon) populate it with
	// per-component checks during initialization; the v1 health handlers
	// snapshot it on every request.
	//
	// NewConfig returns a non-nil registry so handlers don't need to
	// nil-guard.
	HealthRegistry *health.Registry
}

// NewConfig returns a router Config with a fresh, empty health.Registry so
// callers can use the value as-is without nil-guarding HealthRegistry.
func NewConfig() *Config {
	return &Config{
		HealthRegistry: health.NewRegistry(),
	}
}
