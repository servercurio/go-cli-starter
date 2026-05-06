package api

import (
	"github.com/servercurio/go-echo-starter/internal/api/std/module"
	v1 "github.com/servercurio/go-echo-starter/internal/api/v1"
	"github.com/servercurio/go-echo-starter/internal/router"
)

// Module identity and URL prefix for the umbrella api module. Kept as
// constants so the registration sites and any future test fixtures stay in
// agreement.
const (
	moduleIdentifier = "api"
	moduleName       = "api"
	modulePrefix     = "api"
)

// Module returns the umbrella router.Module that aggregates every API
// version. cfg is forwarded down to each version so handlers share the same
// HealthRegistry and any future cross-cutting wiring.
func Module(cfg *router.Config) router.Module {
	return module.New(
		moduleIdentifier,
		moduleName,
		modulePrefix,
		module.WithSubModules(v1.Module(cfg)),
	)
}
