package application

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	apperrors "github.com/servercurio/go-cli-starter/internal/errors"
	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/openapi"
	"github.com/servercurio/go-cli-starter/internal/router"
	"github.com/servercurio/go-cli-starter/internal/version"
)

// initializeOpenAPI builds the OpenAPI spec from every currently registered
// module, marshals it to YAML and JSON once, and registers the openapi
// module (and optionally the Swagger UI module) with the application. Spec
// generation runs only when OpenAPI.Enabled is true; the spec endpoints
// themselves are also gated on Enabled, so disabling the subsystem leaves
// no openapi-related routes attached.
//
// The spec is built before initializeRouting attaches modules to Echo, so
// the new routes ride along with everything else on first boot. The spec
// itself does NOT include the openapi or swagger meta-paths — those are
// implementation detail and conventionally left out of API documentation.
func (app *Application) initializeOpenAPI() error {
	cfg := app.config.OpenAPI
	if cfg == nil || !cfg.Enabled {
		logging.Daemon.Info().Msg("openapi subsystem disabled")
		return nil
	}

	info := openapi.Info{
		Title:       firstNonEmpty(cfg.Title, app.Name),
		Version:     firstNonEmpty(cfg.Version, version.Number()),
		Description: cfg.Description,
	}
	servers := app.openapiServers()

	spec := openapi.Build(info, servers, modulesAsSlice(app.modules))

	yamlBytes, err := yaml.Marshal(spec)
	if err != nil {
		return apperrors.OpenAPIGenerationFailed.Wrap(err, "failed to marshal openapi yaml")
	}
	jsonBytes, err := json.Marshal(spec)
	if err != nil {
		return apperrors.OpenAPIGenerationFailed.Wrap(err, "failed to marshal openapi json")
	}

	if err := app.RegisterModule(openapi.Module(yamlBytes, jsonBytes)); err != nil {
		return apperrors.OpenAPIGenerationFailed.Wrap(err, "failed to register openapi module")
	}

	if cfg.Swagger != nil && cfg.Swagger.Enabled {
		swaggerOpts := openapi.SwaggerOptions{
			Path:    cfg.Swagger.Path,
			SpecURL: cfg.Swagger.SpecURL,
		}
		if err := app.RegisterModule(openapi.SwaggerModule(swaggerOpts)); err != nil {
			return apperrors.OpenAPIGenerationFailed.Wrap(err, "failed to register swagger ui module")
		}
		logging.Daemon.Info().
			Str("path", swaggerOpts.Path).
			Str("specUrl", swaggerOpts.SpecURL).
			Msg("swagger ui mounted")
	}

	logging.Daemon.Info().
		Int("paths", len(spec.Paths)).
		Msg("openapi spec generated and mounted at /openapi.yaml and /openapi.json")
	return nil
}

// openapiServers returns the OpenAPI `servers` array derived from current
// HTTP/HTTPS configuration. We list the HTTPS server first when enabled so
// Swagger UI defaults to the secure URL.
func (app *Application) openapiServers() []openapi.Server {
	var servers []openapi.Server

	if app.config.Server.Https != nil && app.config.Server.Https.Enabled {
		host := app.config.Server.Https.Hostname
		if host == "" {
			host = "localhost"
		}
		servers = append(servers, openapi.Server{
			URL:         fmt.Sprintf("https://%s:%d", host, app.config.Server.Https.Port),
			Description: "TLS endpoint",
		})
	}

	httpHost := app.config.Server.Http.Hostname
	if httpHost == "" {
		httpHost = "localhost"
	}
	servers = append(servers, openapi.Server{
		URL:         fmt.Sprintf("http://%s:%d", httpHost, app.config.Server.Http.Port),
		Description: "HTTP endpoint",
	})

	return servers
}

// modulesAsSlice flattens app.modules to a stable slice. Iteration order on
// the underlying map isn't guaranteed, but the spec output is deterministic
// because Build sorts tags and yaml.v3/json sort path keys at marshal time.
func modulesAsSlice(m map[string]router.Module) []router.Module {
	out := make([]router.Module, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// firstNonEmpty returns the first argument that isn't the empty string.
// Used for config fields that should fall back to a derived default when
// the user hasn't supplied an override.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
