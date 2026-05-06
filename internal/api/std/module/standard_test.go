package module_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	asrt "github.com/stretchr/testify/assert"

	"github.com/servercurio/go-cli-starter/internal/api/std/endpoint"
	"github.com/servercurio/go-cli-starter/internal/api/std/module"
	"github.com/servercurio/go-cli-starter/internal/api/std/route"
)

// TestStandard_PrefixNormalisation pins the contract Prefix() must satisfy.
// Empty / slash-only prefixes must collapse to "" so root-mounted modules
// can register routes at their literal path; otherwise router.initializeRouting
// calls server.Group("/", ...) and Echo concatenates the leading "/" with each
// route's "/openapi.yaml" into "//openapi.yaml" — every route 404s. This is a
// regression guard against the bug observed when registering the openapi /
// swagger modules with prefix="" before the fix.
func TestStandard_PrefixNormalisation(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},          // root-mount, empty
		{"/", ""},         // root-mount, slash-only
		{"//", ""},        // root-mount, doubled slash
		{"api", "/api"},   // bare segment
		{"/api", "/api"},  // already-slashed
		{"/api/", "/api"}, // trailing slash trimmed
		{"v1", "/v1"},     // sub-module shape
	}

	for _, c := range cases {
		m := module.New("id", "name", c.input)
		asrt.Equal(t, c.want, m.Prefix(), "prefix=%q", c.input)
	}
}

// TestStandard_RootMountedModuleServesAtLiteralPath proves end-to-end that a
// module constructed with prefix="" actually serves its routes at the path
// the route declares — not at "//path". This complements the unit-only
// Prefix() check above by covering the Echo group concatenation.
func TestStandard_RootMountedModuleServesAtLiteralPath(t *testing.T) {
	assert := asrt.New(t)

	rootMod := module.New("root", "root", "",
		module.WithRoutes(
			route.New("hello", "hello", "/hello",
				route.WithEndpoints(
					endpoint.New("hello-get", "hello-get",
						endpoint.WithGetMethod(),
						endpoint.WithHandler(func(c *echo.Context) error {
							return c.String(http.StatusOK, "hi")
						}),
					),
				),
			),
		),
	)

	e := echo.New()
	g := e.Group(rootMod.Prefix(), rootMod.Middleware()...)
	rootMod.AttachGroup(g)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	assert.Equal(http.StatusOK, rec.Code, "root-mounted route must answer at /hello, not //hello")
	assert.Equal("hi", rec.Body.String())
}

// TestStandard_PrefixedModuleStillNests validates that the fix didn't
// regress nested modules — the api → v1 path shape this starter actually
// uses must keep producing /api/v1/<route>.
func TestStandard_PrefixedModuleStillNests(t *testing.T) {
	assert := asrt.New(t)

	v1 := module.New("v1", "v1", "v1",
		module.WithRoutes(
			route.New("ping", "ping", "/ping",
				route.WithEndpoints(
					endpoint.New("ping-get", "ping-get",
						endpoint.WithGetMethod(),
						endpoint.WithHandler(func(c *echo.Context) error {
							return c.String(http.StatusOK, "pong")
						}),
					),
				),
			),
		),
	)
	api := module.New("api", "api", "api", module.WithSubModules(v1))

	e := echo.New()
	g := e.Group(api.Prefix(), api.Middleware()...)
	api.AttachGroup(g)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	assert.Equal(http.StatusOK, rec.Code)
	assert.Equal("pong", rec.Body.String())
}
