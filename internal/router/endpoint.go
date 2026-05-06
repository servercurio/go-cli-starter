package router

import (
	"reflect"

	"github.com/labstack/echo/v5"
)

// RequestSpec describes the body an endpoint expects. ContentType defaults
// to application/json when populated via the std/endpoint builder options;
// Type is the Go type the body deserialises into (used by the openapi
// schema generator to derive a JSON Schema).
type RequestSpec struct {
	Type        reflect.Type
	ContentType string
	Description string
}

// ResponseSpec describes one possible response from an endpoint, keyed by
// HTTP status code in Endpoint.Responses(). Type may be nil for
// no-body responses (e.g. 204 No Content). ContentType defaults to
// application/json when populated via the builder options.
type ResponseSpec struct {
	Type        reflect.Type
	ContentType string
	Description string
}

// Endpoint is a single HTTP-method handler attached to a Route. Endpoints
// carry their own OpenAPI metadata (summary, description, request/response
// shapes) so the spec generator at internal/openapi can derive schemas
// without a separate annotation grammar.
type Endpoint interface {
	// Id returns the unique identifier of the endpoint.
	Id() string

	// Name returns the user readable name of the endpoint.
	Name() string

	// Methods returns the list of HTTP methods supported by the endpoint.
	Methods() []string

	// Middleware returns the list of echo.MiddlewareFunc methods which should be registered for
	// this module.
	Middleware() []echo.MiddlewareFunc

	// HandleRequest processes the incoming request and produces a response.
	HandleRequest(c *echo.Context) error

	// AttachGroup registers the echo.Group with the Module.
	AttachGroup(group *echo.Group)

	// Group returns the associated echo.Group instance.
	Group() *echo.Group

	// Summary returns the OpenAPI operation summary, or "" when not set.
	Summary() string

	// Description returns the OpenAPI operation description, or "" when not set.
	Description() string

	// Request returns the OpenAPI request body spec, or nil when the
	// endpoint does not consume a request body.
	Request() *RequestSpec

	// Responses returns OpenAPI response specs keyed by HTTP status code.
	// Returns an empty map (never nil) when no responses are declared, so
	// callers don't need to nil-guard during iteration.
	Responses() map[int]ResponseSpec
}
