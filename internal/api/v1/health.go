package v1

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-cli-starter/internal/api/std/endpoint"
	"github.com/servercurio/go-cli-starter/internal/api/std/route"
	"github.com/servercurio/go-cli-starter/internal/health"
	"github.com/servercurio/go-cli-starter/internal/router"
)

// LivenessRoute mounts the Kubernetes-style liveness probe at /livez.
//
// Liveness reports whether the process can serve HTTP at all. By design it
// does NOT consult the registry: liveness must not depend on application
// readiness, downstream services, or shutdown state. A liveness failure
// tells the kubelet to restart the pod, so we always emit StatusUp here —
// if the listener is down, the request never reaches this handler.
//
// The body shape mirrors /readyz so consumers can treat them uniformly.
func LivenessRoute() router.Route {
	return route.New("liveness", "liveness", "/livez",
		route.WithEndpoints(
			endpoint.New("liveness-get", "liveness-get",
				endpoint.WithGetMethod(),
				endpoint.WithSummary("Liveness probe"),
				endpoint.WithDescription("Returns 200 whenever the process is able to respond to HTTP. Used by Kubernetes' liveness probe to decide whether to restart the pod."),
				endpoint.WithResponse(http.StatusOK, health.Report{}, "Process is alive"),
				endpoint.WithHandler(func(c *echo.Context) error {
					report := health.Report{
						Status: health.StatusUp,
						Components: map[string]health.ComponentResult{
							"self": {Status: health.StatusUp},
						},
					}
					return health.Render(c, http.StatusOK, report)
				}),
			),
		),
	)
}

// ReadinessRoute mounts the Kubernetes-style readiness probe at /readyz.
//
// Readiness aggregates every component registered on cfg.HealthRegistry.
// HTTP status follows the aggregate: 200 when all UP, 503 when any
// component is DOWN. Body format is JSON by default, YAML when the Accept
// header asks for any yaml media type.
func ReadinessRoute(cfg *router.Config) router.Route {
	return route.New("readiness", "readiness", "/readyz",
		route.WithEndpoints(snapshotEndpoint("readiness-get", "readiness-get", cfg)),
	)
}

// HealthRoute mounts the legacy /healthz path. Kept as an alias for /readyz
// so existing consumers that default to /healthz (older uptime checks,
// default cloud LB health-check paths) keep working without configuration
// changes. Behaviour and body shape are identical to /readyz.
func HealthRoute(cfg *router.Config) router.Route {
	return route.New("health", "health", "/healthz",
		route.WithEndpoints(snapshotEndpoint("health-get", "health-get", cfg)),
	)
}

// snapshotEndpoint is the shared handler used by /readyz and /healthz so
// the readiness contract has exactly one definition. A nil registry is
// treated as not-ready (fail closed) — the test in health_test.go pins
// this behaviour so a future refactor can't silently turn a misconfiguration
// into a misleading 200 OK.
func snapshotEndpoint(id, name string, cfg *router.Config) router.Endpoint {
	return endpoint.New(id, name,
		endpoint.WithGetMethod(),
		endpoint.WithSummary("Readiness probe"),
		endpoint.WithDescription("Aggregates the status of every registered component. Returns 200 when every component reports UP; 503 when any reports DOWN. Used by load balancers / Kubernetes readiness probes to drain traffic from unhealthy instances."),
		endpoint.WithResponse(http.StatusOK, health.Report{}, "All components are UP"),
		endpoint.WithResponse(http.StatusServiceUnavailable, health.Report{}, "At least one component is DOWN"),
		endpoint.WithHandler(func(c *echo.Context) error {
			if cfg == nil || cfg.HealthRegistry == nil {
				return health.Render(c, http.StatusServiceUnavailable, health.Report{
					Status:     health.StatusDown,
					Components: map[string]health.ComponentResult{},
				})
			}

			report := cfg.HealthRegistry.Snapshot(c.Request().Context())
			statusCode := http.StatusOK
			if report.Status != health.StatusUp {
				statusCode = http.StatusServiceUnavailable
			}
			return health.Render(c, statusCode, report)
		}),
	)
}
