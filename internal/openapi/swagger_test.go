package openapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	asrt "github.com/stretchr/testify/assert"

	"github.com/servercurio/go-echo-starter/internal/openapi"
	"github.com/servercurio/go-echo-starter/internal/version"
)

// attachSwagger wires SwaggerModule into a fresh Echo so each test case can
// hit it directly. SwaggerModule is registered at the root group so its
// declared routes (`/swagger` and `/swagger/*`) land at their literal
// paths, mirroring how Application.initializeRouting attaches it in
// production.
func attachSwagger(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	mod := openapi.SwaggerModule(openapi.SwaggerOptions{
		Path:    "/swagger",
		SpecURL: "/openapi.yaml",
	})
	g := e.Group(mod.Prefix(), mod.Middleware()...)
	mod.AttachGroup(g)
	return e
}

// TestSwaggerModule_BarePrefixRedirectsToIndex pins the fix for the
// "GET /swagger → 404" bug. Echo's wildcard `/swagger/*` only matches paths
// starting with `/swagger/` — not the bare prefix users actually type into
// a browser — so the module registers a sibling redirect at `/swagger`. We
// jump straight to /swagger/index.html rather than /swagger/ to skip the
// intermediate redirect that echo-swagger would otherwise emit, saving
// one network round-trip in the common case.
func TestSwaggerModule_BarePrefixRedirectsToIndex(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger", nil))

	assert.Equal(http.StatusMovedPermanently, rec.Code, "bare /swagger must redirect, not 404")
	assert.Equal("/swagger/index.html", rec.Header().Get(echo.HeaderLocation),
		"redirect should target index.html directly to skip echo-swagger's own /swagger/ → /index.html hop")
}

// TestSwaggerModule_WildcardHandlerStillMounted is the inverse safety net:
// the redirect must not have replaced the actual UI handler. Hitting the
// trailing-slash form should reach echo-swagger's handler — which itself
// emits a 301 to /index.html. Either response code (the 301 from
// echo-swagger or a 200 if its behaviour changes) confirms the wildcard
// route is still wired up; what we're guarding against is the wildcard
// not being registered at all (404).
func TestSwaggerModule_WildcardHandlerStillMounted(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/", nil))

	assert.NotEqual(http.StatusNotFound, rec.Code, "trailing-slash form must hit a handler, not 404")
}

// TestSwaggerModule_AssetPathsServed checks that the wildcard captures the
// supporting assets too — without it the UI would render but be unable to
// fetch its own JS/CSS.
func TestSwaggerModule_AssetPathsServed(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(http.StatusOK, rec.Code, "explicit /swagger/index.html should also be served")
}

// TestSwaggerModule_RenderedIndexUsesConfiguredSpecURLOnly is a regression
// guard against the "Fetch error response status is 500 doc.json" bug.
//
// echoSwagger.URL(spec) APPENDS to the default URLs list (`["doc.json",
// "doc.yaml"]`) rather than replacing it, so the rendered index.html ends
// up offering doc.json first, which has no corresponding spec on disk and
// 500s when Swagger UI tries to load it. SwaggerModule works around that
// by setting Config.URLs directly. This test pins that behaviour by
// scraping the rendered HTML and asserting:
//
//   - the configured spec URL appears (UI will load it), and
//   - the doc.json / doc.yaml defaults do not (no fetch error banner).
func TestSwaggerModule_RenderedIndexUsesConfiguredSpecURLOnly(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(http.StatusOK, rec.Code)

	body := rec.Body.String()
	// "/openapi.yaml" appears with a backslash-escape inside the rendered
	// JS literal (\/openapi.yaml). Match the unescaped form by stripping
	// backslashes before comparison.
	stripped := strings.ReplaceAll(body, `\/`, `/`)

	assert.Contains(stripped, `"/openapi.yaml"`,
		"rendered swagger UI must list the configured spec URL")
	assert.NotContains(stripped, `"doc.json"`,
		"echo-swagger's default doc.json must be cleared, otherwise the UI 500s on first load")
	assert.NotContains(stripped, `"doc.yaml"`,
		"echo-swagger's default doc.yaml must also be cleared")
}

// TestSwaggerModule_IndexHidesPickerKeepsBrandedTopbar pins the topbar
// styling contract. We keep StandaloneLayout (so the topbar renders), but
// hide the spec URL/picker controls (`.download-url-wrapper`) via CSS and
// swap the default Swagger logo for the Server Curio brandmark served at
// the sibling /logo.svg route. Three things must hold:
//
//   - the topbar is still rendered (StandaloneLayout in the bootstrap),
//   - the picker block is suppressed via CSS, and
//   - the brandmark is referenced.
func TestSwaggerModule_IndexHidesPickerKeepsBrandedTopbar(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(body, `layout: "StandaloneLayout"`,
		"index.html must render with StandaloneLayout so the branded topbar stays visible")
	assert.Contains(body, ".download-url-wrapper { display: none",
		"the spec URL/picker block must be hidden via CSS")
	assert.Contains(body, `url("./logo.svg")`,
		"the topbar must reference the Server Curio brandmark")
	assert.Contains(body, `url: "/openapi.yaml"`,
		"index.html must still pin the spec URL")
}

// TestSwaggerModule_LogoServed verifies the brandmark referenced by the
// rendered topbar CSS is actually reachable at the sibling route. A 404
// here would leave the topbar with a blank gap instead of the logo.
func TestSwaggerModule_LogoServed(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/logo.svg", nil))

	assert.Equal(http.StatusOK, rec.Code)
	assert.Equal("image/svg+xml", rec.Header().Get(echo.HeaderContentType))
	assert.Contains(rec.Body.String(), "<svg",
		"response must be an SVG document, not the echo-swagger asset handler's fallback")
}

// TestSwaggerModule_IndexRendersBuildAndLicenseFooter pins the footer
// contract: the rendered index must surface the build's version, the
// short commit prefix (first 7 chars of the embedded commit hash), and
// the project copyright/license attribution. These ride along with the
// rendered HTML so consumers reading the spec UI can see exactly which
// build of the daemon is serving them.
func TestSwaggerModule_IndexRendersBuildAndLicenseFooter(t *testing.T) {
	assert := asrt.New(t)

	e := attachSwagger(t)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(http.StatusOK, rec.Code)

	body := rec.Body.String()

	assert.Contains(body, `class="swagger-footer"`,
		"rendered index must include the footer container")
	assert.Contains(body, version.Number(),
		"footer must surface the embedded version number")

	commit := version.Commit()
	if len(commit) >= 7 {
		assert.Contains(body, commit[:7],
			"footer must surface the 7-char commit prefix")
		assert.NotContains(body, commit,
			"footer must not leak the full commit hash — only the short prefix")
	}

	assert.Contains(body, "Server Curio",
		"footer must carry the project copyright attribution")
	assert.Contains(body, "Apache License, Version 2.0",
		"footer must declare the license")
	assert.Contains(body, "https://www.apache.org/licenses/LICENSE-2.0",
		"footer must link to the license text")
}

// TestSwaggerModule_RespectsCustomPath validates that callers can mount
// Swagger UI under any prefix, not just /swagger — and that the redirect
// target tracks the custom prefix.
func TestSwaggerModule_RespectsCustomPath(t *testing.T) {
	assert := asrt.New(t)

	e := echo.New()
	mod := openapi.SwaggerModule(openapi.SwaggerOptions{
		Path:    "/api-docs",
		SpecURL: "/openapi.yaml",
	})
	g := e.Group(mod.Prefix(), mod.Middleware()...)
	mod.AttachGroup(g)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api-docs", nil))

	assert.Equal(http.StatusMovedPermanently, rec.Code)
	assert.Equal("/api-docs/index.html", rec.Header().Get(echo.HeaderLocation))
}
