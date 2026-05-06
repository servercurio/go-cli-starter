package endpoint

import (
	"net/http"
	"reflect"
	"slices"

	"github.com/labstack/echo/v5"

	"github.com/servercurio/go-cli-starter/internal/router"
)

// defaultContentType is what builder options assume when a caller doesn't
// specify a content type explicitly. JSON is the dominant case for
// modern APIs and matches what the v1 health endpoints emit; richer
// negotiation belongs in a future With…ContentType-style overload.
const defaultContentType = "application/json"

// Standard is the canonical router.Endpoint implementation: an HTTP handler
// plus the OpenAPI metadata (summary, description, request/response shapes)
// the spec generator reads at build time. Construct via New + Option helpers.
type Standard struct {
	id          string
	name        string
	summary     string
	description string
	methods     []string
	middleware  []echo.MiddlewareFunc
	group       *echo.Group
	handler     echo.HandlerFunc
	request     *router.RequestSpec
	responses   map[int]router.ResponseSpec
}

// Id returns the endpoint's stable identifier (used as the OpenAPI
// operationId).
func (e *Standard) Id() string {
	return e.id
}

// Name returns the endpoint's human-readable name.
func (e *Standard) Name() string {
	return e.name
}

// Methods returns the HTTP methods the endpoint responds to, deduplicated
// and sorted (the order is the slice order returned by New).
func (e *Standard) Methods() []string {
	return e.methods
}

// Middleware returns the per-endpoint middleware chain in registration
// order.
func (e *Standard) Middleware() []echo.MiddlewareFunc {
	return e.middleware
}

// Summary returns the OpenAPI operation summary set via WithSummary, or "".
func (e *Standard) Summary() string {
	return e.summary
}

// Description returns the OpenAPI operation description set via
// WithDescription, or "".
func (e *Standard) Description() string {
	return e.description
}

// Request returns the OpenAPI request body spec set via WithRequest, or nil
// when the endpoint declares no body.
func (e *Standard) Request() *router.RequestSpec {
	return e.request
}

// Responses returns the OpenAPI response specs keyed by HTTP status code.
// Returns a freshly-allocated empty map (never nil) when no responses are
// declared, so callers don't need to nil-guard.
func (e *Standard) Responses() map[int]router.ResponseSpec {
	if e.responses == nil {
		return map[int]router.ResponseSpec{}
	}
	return e.responses
}

// AttachGroup binds the endpoint to an *echo.Group. No-op when group is nil
// or the endpoint has already been attached.
func (e *Standard) AttachGroup(group *echo.Group) {
	if group == nil || e.group != nil {
		return
	}

	e.group = group
}

// HandleRequest invokes the registered handler. Returns 501 Not Implemented
// when no handler has been attached, so an Endpoint declared for OpenAPI
// purposes alone fails predictably rather than panicking.
func (e *Standard) HandleRequest(c *echo.Context) error {
	if e.handler == nil {
		return c.NoContent(http.StatusNotImplemented)
	}

	return e.handler(c)
}

// Group returns the *echo.Group the endpoint is attached to, or nil before
// AttachGroup has been called.
func (e *Standard) Group() *echo.Group {
	return e.group
}

// New returns a Standard endpoint with the given id and name, configured by
// the supplied Options. Method registrations from With*Method calls are
// deduplicated before the constructor returns.
func New(id, name string, options ...Option) *Standard {
	std := &Standard{
		id:         id,
		name:       name,
		methods:    make([]string, 0),
		middleware: make([]echo.MiddlewareFunc, 0),
		responses:  make(map[int]router.ResponseSpec),
	}

	for _, opt := range options {
		opt(std)
	}

	// Remove duplicates from methods
	slices.Sort(std.methods)
	std.methods = slices.Compact(std.methods)

	return std
}

// Option is a function that configures the Standard endpoint.
type Option func(*Standard)

// WithHandler sets the handler for the Standard endpoint.
func WithHandler(handler echo.HandlerFunc) Option {
	return func(e *Standard) {
		e.handler = handler
	}
}

// WithMiddleware adds middleware to the Standard endpoint.
func WithMiddleware(middleware ...echo.MiddlewareFunc) Option {
	return func(e *Standard) {
		if e.middleware == nil {
			e.middleware = make([]echo.MiddlewareFunc, 0)
		}
		e.middleware = append(e.middleware, middleware...)
	}
}

// WithMethods adds HTTP methods to the Standard endpoint.
func WithMethods(methods ...string) Option {
	return func(e *Standard) {
		if e.methods == nil {
			e.methods = make([]string, 0)
		}
		e.methods = append(e.methods, methods...)
	}
}

// WithGetMethod registers the endpoint to respond to HTTP GET.
func WithGetMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodGet)
	}
}

// WithPostMethod registers the endpoint to respond to HTTP POST.
func WithPostMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodPost)
	}
}

// WithPutMethod registers the endpoint to respond to HTTP PUT.
func WithPutMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodPut)
	}
}

// WithDeleteMethod registers the endpoint to respond to HTTP DELETE.
func WithDeleteMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodDelete)
	}
}

// WithPatchMethod registers the endpoint to respond to HTTP PATCH.
func WithPatchMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodPatch)
	}
}

// WithOptionsMethod registers the endpoint to respond to HTTP OPTIONS.
func WithOptionsMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodOptions)
	}
}

// WithHeadMethod registers the endpoint to respond to HTTP HEAD.
func WithHeadMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodHead)
	}
}

// WithTraceMethod registers the endpoint to respond to HTTP TRACE.
func WithTraceMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodTrace)
	}
}

// WithConnectMethod registers the endpoint to respond to HTTP CONNECT.
func WithConnectMethod() Option {
	return func(e *Standard) {
		e.methods = append(e.methods, http.MethodConnect)
	}
}

// WithSummary sets the OpenAPI operation summary — a short, human-readable
// title that Swagger UI surfaces in the operation header.
func WithSummary(s string) Option {
	return func(e *Standard) {
		e.summary = s
	}
}

// WithDescription sets the OpenAPI operation description — a longer free-form
// explanation rendered below the summary in Swagger UI.
func WithDescription(s string) Option {
	return func(e *Standard) {
		e.description = s
	}
}

// WithRequest declares that the endpoint accepts a request body of the given
// Go type. Pass a zero value of the body type — e.g. WithRequest(CreateUserRequest{})
// — and the openapi schema generator reflects on it to produce the JSON
// Schema. The request is documented as application/json.
//
// Pass nil to clear a previously-declared request body.
func WithRequest(value any) Option {
	return func(e *Standard) {
		if value == nil {
			e.request = nil
			return
		}
		e.request = &router.RequestSpec{
			Type:        reflect.TypeOf(value),
			ContentType: defaultContentType,
		}
	}
}

// WithResponse declares one possible response for the endpoint, keyed by the
// HTTP status code. Pass a zero value of the response body type — e.g.
// WithResponse(200, health.Report{}, "OK") — and the openapi schema generator
// reflects on it to produce the JSON Schema. Pass nil for no-body responses
// (e.g. 204 No Content). The response is documented as application/json.
//
// Repeated calls with the same status replace the previous declaration so a
// route constructor can override an inherited default.
func WithResponse(code int, value any, description string) Option {
	return func(e *Standard) {
		if e.responses == nil {
			e.responses = make(map[int]router.ResponseSpec)
		}
		spec := router.ResponseSpec{
			ContentType: defaultContentType,
			Description: description,
		}
		if value != nil {
			spec.Type = reflect.TypeOf(value)
		}
		e.responses[code] = spec
	}
}
