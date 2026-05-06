package router

import "github.com/labstack/echo/v5"

// Route describes a single addressable URI under a Module: a path, the HTTP
// methods that respond to it (via its Endpoints), and any per-route
// middleware. Implementations are typically built with internal/api/std/route.
type Route interface {
	// Id returns the unique identifier of the route.
	Id() string

	// Name returns the user readable name of the route.
	Name() string

	// Path returns the path of the route.
	Path() string

	// Middleware returns the list of echo.MiddlewareFunc methods which should be registered for
	// this module.
	Middleware() []echo.MiddlewareFunc

	// Endpoints returns the list of endpoints associated with the route.
	Endpoints() []Endpoint

	// AttachGroup registers the echo.Group with the Module.
	AttachGroup(group *echo.Group)

	// Group returns the associated echo.Group instance.
	Group() *echo.Group
}
