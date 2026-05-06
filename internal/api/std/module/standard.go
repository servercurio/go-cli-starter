package module

import (
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/router"
)

// Standard is the canonical router.Module implementation. Routes and
// sub-modules are keyed by id so duplicate registrations are idempotent;
// AttachGroup wires everything onto a supplied *echo.Group exactly once.
type Standard struct {
	id         string
	name       string
	prefix     string
	routes     map[string]router.Route
	modules    map[string]router.Module
	middleware []echo.MiddlewareFunc
	group      *echo.Group
}

// Id returns the module's stable identifier.
func (m *Standard) Id() string {
	return m.id
}

// Name returns the module's human-readable name (used as the OpenAPI tag).
func (m *Standard) Name() string {
	return m.name
}

// Prefix returns the URL prefix this module mounts under, normalised to
// either:
//
//   - the empty string for root-mounted modules (prefix was "" or "/"), so
//     the caller's `server.Group(prefix, ...)` invocation doesn't introduce
//     a stray leading "/" that Echo would naively concatenate with route
//     paths into "//route" and 404 every request, or
//
//   - a leading-slash form ("/api", "/v1") for nested modules.
//
// Without this normalisation, a module constructed with prefix=""
// effectively can't expose top-level routes — a real footgun, since "no
// prefix" is exactly what the openapi / swagger modules want.
func (m *Standard) Prefix() string {
	trimmed := strings.Trim(m.prefix, router.PathSeparator)
	if trimmed == "" {
		return ""
	}
	return router.PathSeparator + trimmed
}

// Routes returns the module's directly-registered routes in unspecified
// order. The slice is freshly allocated; the caller may mutate it without
// affecting the module.
func (m *Standard) Routes() []router.Route {
	var ret []router.Route
	for _, v := range m.routes {
		ret = append(ret, v)
	}
	return ret
}

// SubModules returns the module's nested sub-modules in unspecified order.
// The slice is freshly allocated; the caller may mutate it without affecting
// the module.
func (m *Standard) SubModules() []router.Module {
	var ret []router.Module
	for _, v := range m.modules {
		ret = append(ret, v)
	}
	return ret
}

// HasRoutes reports whether the module has any directly-registered routes.
func (m *Standard) HasRoutes() bool {
	return len(m.routes) > 0
}

// HasSubModules reports whether the module has any registered sub-modules.
func (m *Standard) HasSubModules() bool {
	return len(m.modules) > 0
}

// Middleware returns the module-level middleware chain in registration
// order.
func (m *Standard) Middleware() []echo.MiddlewareFunc {
	return m.middleware
}

// AttachGroup binds the module to an *echo.Group, recursively attaching
// sub-modules and routes. No-op when group is nil or the module has already
// been attached, so re-registration is safe.
func (m *Standard) AttachGroup(group *echo.Group) {
	if group == nil || m.group != nil {
		return
	}

	logging.Daemon.Info().
		Str("name", m.Name()).
		Str("prefix", m.Prefix()).
		Msg("http router - registering module prefix")

	m.group = group
	m.group.Use(m.middleware...)

	if m.HasSubModules() {
		for _, module := range m.modules {
			g := m.group.Group(module.Prefix(), module.Middleware()...)
			module.AttachGroup(g)
		}
	}

	if m.HasRoutes() {
		for _, route := range m.routes {
			route.AttachGroup(m.group)
		}
	}
}

// Group returns the *echo.Group the module is attached to, or nil before
// AttachGroup has been called.
func (m *Standard) Group() *echo.Group {
	return m.group
}

// Option configures a Standard during New. Used as a functional-options
// builder so module construction stays composable.
type Option func(m *Standard)

// New returns a router.Module backed by Standard with the given id, name,
// and URL prefix, configured by the supplied Options.
func New(id, name, prefix string, opts ...Option) router.Module {
	m := &Standard{
		id:         id,
		name:       name,
		prefix:     prefix,
		routes:     make(map[string]router.Route),
		modules:    make(map[string]router.Module),
		middleware: make([]echo.MiddlewareFunc, 0),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithMiddleware appends one or more middleware functions to the module's
// middleware chain. Multiple calls accumulate in registration order.
func WithMiddleware(mw ...echo.MiddlewareFunc) Option {
	return func(m *Standard) {
		if m.middleware == nil {
			m.middleware = make([]echo.MiddlewareFunc, 0)
		}

		m.middleware = append(m.middleware, mw...)
	}
}

// WithRoutes registers one or more routes on the module, keyed by id.
// Repeated registrations of the same id are ignored, so callers can merge
// default + override sets safely.
func WithRoutes(routes ...router.Route) Option {
	return func(m *Standard) {
		if m.routes == nil {
			m.routes = make(map[string]router.Route)
		}

		for _, route := range routes {
			if _, ok := m.routes[route.Id()]; !ok {
				m.routes[route.Id()] = route
			}
		}
	}
}

// WithSubModules registers one or more sub-modules on the module, keyed by
// id. Repeated registrations of the same id are ignored.
func WithSubModules(modules ...router.Module) Option {
	return func(m *Standard) {
		if m.modules == nil {
			m.modules = make(map[string]router.Module)
		}

		for _, mod := range modules {
			if _, ok := m.modules[mod.Id()]; !ok {
				m.modules[mod.Id()] = mod
			}
		}
	}
}
