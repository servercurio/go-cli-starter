package openapi

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-cli-starter/internal/api/std/endpoint"
	"github.com/servercurio/go-cli-starter/internal/api/std/module"
	"github.com/servercurio/go-cli-starter/internal/api/std/route"
	"github.com/servercurio/go-cli-starter/internal/router"
)

// Content types and URL paths for the served OpenAPI spec. Extracted so the
// route registrations and handlers stay in lock-step.
const (
	yamlContentType = "application/yaml; charset=utf-8"
	jsonContentType = "application/json; charset=utf-8"

	specYAMLPath = "/openapi.yaml"
	specJSONPath = "/openapi.json"
)

// Module returns a router.Module that serves a precomputed OpenAPI spec at
// /openapi.yaml and /openapi.json. The bytes are captured by the handler
// closures and reused on every request — there's no per-request marshaling
// or module re-walk, since the spec is immutable once Application.Initialize
// has finished registering modules.
//
// The module is registered at the root of the URL tree (prefix ""), so the
// final paths are exactly /openapi.yaml and /openapi.json regardless of any
// versioned API prefix that other modules use.
func Module(yamlBytes, jsonBytes []byte) router.Module {
	return module.New("openapi", "openapi", "",
		module.WithRoutes(
			route.New("openapi-yaml", "openapi-yaml", specYAMLPath,
				route.WithEndpoints(
					endpoint.New("openapi-yaml-get", "openapi-yaml-get",
						endpoint.WithGetMethod(),
						endpoint.WithHandler(func(c *echo.Context) error {
							return c.Blob(http.StatusOK, yamlContentType, yamlBytes)
						}),
					),
				),
			),
			route.New("openapi-json", "openapi-json", specJSONPath,
				route.WithEndpoints(
					endpoint.New("openapi-json-get", "openapi-json-get",
						endpoint.WithGetMethod(),
						endpoint.WithHandler(func(c *echo.Context) error {
							return c.Blob(http.StatusOK, jsonContentType, jsonBytes)
						}),
					),
				),
			),
		),
	)
}
