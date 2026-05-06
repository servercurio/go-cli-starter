package v1

import (
	"github.com/servercurio/go-echo-starter/internal/api/std/module"
	"github.com/servercurio/go-echo-starter/internal/router"
)

// Module identity and URL prefix for the v1 API submodule. Constants live at
// the package level so multiple registration sites stay in agreement.
const (
	moduleIdentifier = "v1"
	moduleName       = "api/v1"
	modulePrefix     = "v1"
)

// Module returns the v1 router.Module: a "v1" prefix plus the three health
// routes (/livez, /readyz, /healthz). cfg is plumbed through so /readyz and
// /healthz can read the shared HealthRegistry.
func Module(cfg *router.Config) router.Module {
	return module.New(
		moduleIdentifier,
		moduleName,
		modulePrefix,
		module.WithRoutes(
			LivenessRoute(),
			ReadinessRoute(cfg),
			HealthRoute(cfg),
		),
	)
}
