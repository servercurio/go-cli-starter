package route

import (
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-cli-starter/internal/logging"
	"github.com/servercurio/go-cli-starter/internal/router"
)

// Standard is the canonical router.Route implementation: a path, a set of
// endpoints keyed by id (so duplicate WithEndpoints calls are idempotent),
// and an optional per-route middleware chain. Built via New + Option helpers.
type Standard struct {
	id         string
	name       string
	path       string
	endpoints  map[string]router.Endpoint
	middleware []echo.MiddlewareFunc
	group      *echo.Group
}

// Id returns the route's stable identifier.
func (r *Standard) Id() string {
	return r.id
}

// Name returns the route's human-readable name.
func (r *Standard) Name() string {
	return r.name
}

// Path returns the URI path the route is mounted at, relative to its
// containing module's prefix.
func (r *Standard) Path() string {
	return r.path
}

// Middleware returns the per-route middleware chain in registration order.
func (r *Standard) Middleware() []echo.MiddlewareFunc {
	return r.middleware
}

// Endpoints returns the route's endpoints in unspecified order. The slice is
// freshly allocated; the caller may mutate it without affecting the route.
func (r *Standard) Endpoints() []router.Endpoint {
	var ret []router.Endpoint
	for _, v := range r.endpoints {
		ret = append(ret, v)
	}
	return ret
}

// AttachGroup binds the route to an *echo.Group, wiring its middleware and
// each endpoint's HTTP-method handlers. No-op when group is nil or the route
// has already been attached, so re-registration is safe.
func (r *Standard) AttachGroup(group *echo.Group) {
	if group == nil || r.group != nil {
		return
	}

	r.group = group
	group.Use(r.middleware...)

	for _, ep := range r.endpoints {
		ep.AttachGroup(group)

		path := r.path
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		for _, method := range ep.Methods() {
			group.Add(method, path, ep.HandleRequest, ep.Middleware()...)
		}

		logging.Daemon.
			Info().
			Str("name", ep.Name()).
			Strs("methods", ep.Methods()).
			Msg("http router - registering route endpoint")
	}
}

// Group returns the *echo.Group the route is attached to, or nil before
// AttachGroup has been called.
func (r *Standard) Group() *echo.Group {
	return r.group
}

// New returns a Standard route with the supplied id, name, and path,
// configured by the provided Options.
func New(id, name, path string, options ...Option) *Standard {
	std := &Standard{
		id:         id,
		name:       name,
		path:       path,
		endpoints:  make(map[string]router.Endpoint),
		middleware: make([]echo.MiddlewareFunc, 0),
	}

	for _, opt := range options {
		opt(std)
	}

	return std
}

// Option configures a Standard during New. Used as a functional-options
// builder so route construction stays composable.
type Option func(std *Standard)

// WithMiddleware appends one or more middleware functions to the route's
// middleware chain. Multiple calls accumulate in registration order.
func WithMiddleware(middleware ...echo.MiddlewareFunc) Option {
	return func(r *Standard) {
		r.middleware = append(r.middleware, middleware...)
	}
}

// WithEndpoints registers one or more endpoints on the route, keyed by their
// id. Repeated registrations of the same id are ignored, so callers can
// merge default + override sets safely.
func WithEndpoints(endpoints ...router.Endpoint) Option {
	return func(r *Standard) {
		if r.endpoints == nil {
			r.endpoints = make(map[string]router.Endpoint)
		}

		for _, ep := range endpoints {
			if _, ok := r.endpoints[ep.Id()]; !ok {
				r.endpoints[ep.Id()] = ep
			}
		}
	}
}
