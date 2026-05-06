package application

import (
	"context"
	"strings"

	"github.com/servercurio/go-cli-starter/internal/health"
)

// HealthRegistry returns the per-Application health registry. Wire this into
// router.Config so the v1 health/liveness/readiness handlers can snapshot
// component status on every request.
func (app *Application) HealthRegistry() *health.Registry {
	return app.healthRegistry
}

// registerHealthChecks populates the application's health registry with one
// entry per active subsystem. Called once during Initialize, after every
// subsystem has had a chance to read its configuration. The closures
// captured here are invoked on every /readyz and /healthz request, so they
// must stay cheap — long checks belong behind a cached background probe.
func (app *Application) registerHealthChecks() {
	// Lifecycle: reflects the atomic.Bool flipped by Start/shutdown.
	app.healthRegistry.Register("lifecycle", func(_ context.Context) health.ComponentResult {
		if app.IsReady() {
			return health.ComponentResult{Status: health.StatusUp}
		}
		return health.ComponentResult{
			Status:  health.StatusDown,
			Details: map[string]any{"reason": "application has not yet entered ready state, or shutdown has begun"},
		}
	})

	// HTTP: if the request reaches this handler the listener is up by
	// definition. Surface the bind details for diagnostics.
	httpCfg := app.config.Server.Http
	app.healthRegistry.Register("http", func(_ context.Context) health.ComponentResult {
		return health.ComponentResult{
			Status: health.StatusUp,
			Details: map[string]any{
				"port":        httpCfg.Port,
				"bindAddress": httpCfg.BindAddress,
				"hostname":    httpCfg.Hostname,
			},
		}
	})

	// HTTPS: only registered when enabled — disabled subsystems shouldn't
	// appear in the report at all. Three certificate-source flags surface
	// which path the daemon is taking at runtime, mirroring the truth
	// table in TlsConfig.MarshalZerologObject:
	//
	//   autoCertIssuance       — true when neither cert nor key file is
	//                             configured; the daemon issues its own
	//                             certificate (either ACME or ephemeral).
	//   useAcmeIssuer          — config flag; true means use ACME
	//                             (Let's Encrypt), false means use an
	//                             ephemeral self-signed cert.
	//   ephemeralCertIssuance  — derived: autoCertIssuance && !useAcmeIssuer.
	//                             Operators glance at this to confirm
	//                             they're not accidentally serving a
	//                             self-signed cert in production.
	tlsCfg := app.config.Server.Https
	if tlsCfg != nil && tlsCfg.Enabled {
		app.healthRegistry.Register("https", func(_ context.Context) health.ComponentResult {
			autoCert := strings.TrimSpace(tlsCfg.Certificate) == "" || strings.TrimSpace(tlsCfg.Key) == ""
			ephemeral := autoCert && !tlsCfg.UseAcmeIssuer

			return health.ComponentResult{
				Status: health.StatusUp,
				Details: map[string]any{
					"port":                  tlsCfg.Port,
					"bindAddress":           tlsCfg.BindAddress,
					"hostname":              tlsCfg.Hostname,
					"autoCertIssuance":      autoCert,
					"useAcmeIssuer":         tlsCfg.UseAcmeIssuer,
					"ephemeralCertIssuance": ephemeral,
				},
			}
		})
	}

	// Database: only registered when configured. Status reflects a live
	// PingContext on the pool (see app.IsDatabaseHealthy).
	dbCfg := app.config.Database
	if dbCfg.Enabled() {
		app.healthRegistry.Register("database", func(ctx context.Context) health.ComponentResult {
			details := map[string]any{
				"driver": dbCfg.Driver,
			}
			if app.IsDatabaseHealthy(ctx) {
				return health.ComponentResult{Status: health.StatusUp, Details: details}
			}
			if ctx.Err() == context.DeadlineExceeded {
				details["reason"] = "ping exceeded readiness probe budget"
			} else {
				details["reason"] = "ping failed"
			}
			return health.ComponentResult{Status: health.StatusDown, Details: details}
		})
	}
}
