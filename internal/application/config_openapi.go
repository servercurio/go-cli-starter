package application

import (
	"errors"
	"strings"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// OpenAPIConfig controls runtime exposure of the OpenAPI 3.0 spec generated
// from the registered routes, plus an optional embedded Swagger UI.
type OpenAPIConfig struct {
	// Enabled controls whether /openapi.yaml and /openapi.json are served.
	// Defaults to true; set false to suppress the spec endpoints (e.g. for
	// production deployments that don't want to advertise their API
	// surface).
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Title appears as the OpenAPI document `info.title`. Defaults to the
	// daemon name.
	Title string `yaml:"title" json:"title"`

	// Version appears as the OpenAPI document `info.version`. Defaults to
	// the build-injected SemVer (internal/version.Number()).
	Version string `yaml:"version" json:"version"`

	// Description optionally fills in `info.description`.
	Description string `yaml:"description" json:"description"`

	// Swagger nests the optional Swagger UI configuration.
	Swagger *SwaggerConfig `yaml:"swagger" json:"swagger"`
}

// SwaggerConfig configures the optional Swagger UI overlay.
type SwaggerConfig struct {
	// Enabled controls whether Swagger UI is mounted. Defaults to false
	// because the UI exposes a browseable API surface that production
	// deployments may not want.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Path is the URL prefix Swagger UI is mounted under. Defaults to
	// "/swagger". The UI loads at <Path>/index.html and serves its assets
	// from <Path>/* (the underlying handler is wildcard-rooted).
	Path string `yaml:"path" json:"path"`

	// SpecURL is the URL Swagger UI fetches the OpenAPI document from.
	// Defaults to "/openapi.yaml" — i.e. the file served by the spec
	// endpoint above. Override only if the spec is hosted externally or
	// you've changed the spec mount path.
	SpecURL string `yaml:"specUrl" json:"specUrl"`
}

// FromEnv hydrates the OpenAPI fields from environment variables under
// prefix and recurses into the Swagger sub-config.
func (c *OpenAPIConfig) FromEnv(prefix string) {
	env.SetBoolValue(prefix, "enabled", &c.Enabled)
	env.SetStringValue(prefix, "title", &c.Title)
	env.SetStringValue(prefix, "version", &c.Version)
	env.SetStringValue(prefix, "description", &c.Description)
	c.Swagger.FromEnv(env.AddPrefix(prefix, "swagger"))
}

// FromEnv hydrates the Swagger UI fields from environment variables under
// prefix.
func (c *SwaggerConfig) FromEnv(prefix string) {
	env.SetBoolValue(prefix, "enabled", &c.Enabled)
	env.SetStringValue(prefix, "path", &c.Path)
	env.SetStringValue(prefix, "spec_url", &c.SpecURL)
}

// MarshalZerologObject is the structured-log emitter used by the startup
// notifier. Mirrors the pattern other config types follow so the daemon
// startup log shows OpenAPI/Swagger state alongside server/proxy/database.
func (c *OpenAPIConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Bool("enabled", c.Enabled).
		Str("title", c.Title).
		Str("version", c.Version)
	if c.Swagger != nil {
		e.Bool("swaggerEnabled", c.Swagger.Enabled).
			Str("swaggerPath", c.Swagger.Path).
			Str("swaggerSpecUrl", c.Swagger.SpecURL)
	}
}

// Validate rejects an enabled Swagger UI without a mount path or spec URL —
// echo-swagger would 404 every request to the UI in that state. OpenAPI
// itself doesn't need title/version because Application backfills both
// from the daemon name and build-injected SemVer when they're empty.
func (c *OpenAPIConfig) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	var errs []error
	if c.Swagger != nil && c.Swagger.Enabled {
		if strings.TrimSpace(c.Swagger.Path) == "" {
			errs = append(errs, errors.New("openapi.swagger.path must be set when swagger UI is enabled"))
		}
		if strings.TrimSpace(c.Swagger.SpecURL) == "" {
			errs = append(errs, errors.New("openapi.swagger.specUrl must be set when swagger UI is enabled"))
		}
	}
	return errors.Join(errs...)
}

// DefaultOpenAPIConfig returns the conservative starter defaults: spec on,
// Swagger UI off. Title and Version are filled in by Application after
// Configure runs (so we don't import internal/version here and create a
// cycle with the Taskfile-time version generation).
func DefaultOpenAPIConfig() *OpenAPIConfig {
	return &OpenAPIConfig{
		Enabled: true,
		Title:   "",
		Version: "",
		Swagger: &SwaggerConfig{
			Enabled: false,
			Path:    "/swagger",
			SpecURL: "/openapi.yaml",
		},
	}
}
