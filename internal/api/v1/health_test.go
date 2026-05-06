package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-echo-starter/internal/health"
	"github.com/servercurio/go-echo-starter/internal/router"
	asrt "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// serve attaches r to a fresh Echo instance, sends a GET against path with
// the supplied Accept header, and returns the recorder. Using the real
// route/endpoint constructors (rather than calling handler closures
// directly) exercises the routing path and option wiring, which is the bit
// most likely to regress when the std/* helpers change.
func serve(t *testing.T, r router.Route, path, accept string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	r.AttachGroup(e.Group(""))

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if accept != "" {
		req.Header.Set(echo.HeaderAccept, accept)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// allUpRegistry returns a Registry where every named component reports UP.
func allUpRegistry(names ...string) *health.Registry {
	reg := health.NewRegistry()
	for _, n := range names {
		reg.Register(n, func(_ context.Context) health.ComponentResult {
			return health.ComponentResult{Status: health.StatusUp}
		})
	}
	return reg
}

// ---------- /livez ----------

func TestLivenessRoute_AlwaysReturns200WithSelfComponent(t *testing.T) {
	assert := asrt.New(t)
	rec := serve(t, LivenessRoute(), "/livez", "")

	assert.Equal(http.StatusOK, rec.Code)

	var body health.Report
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(health.StatusUp, body.Status)
	assert.Contains(body.Components, "self")
	assert.Equal(health.StatusUp, body.Components["self"].Status)
}

func TestLivenessRoute_DoesNotConsultRegistry(t *testing.T) {
	// Even with a hypothetical DOWN registry attached, /livez is registry-
	// independent — kubelet should not restart a process just because a
	// dependency is degraded. This pins that contract.
	assert := asrt.New(t)
	rec := serve(t, LivenessRoute(), "/livez", "")

	assert.Equal(http.StatusOK, rec.Code)
}

// ---------- /readyz ----------

func TestReadinessRoute_AllUpReturns200(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle", "http", "database")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "")
	assert.Equal(http.StatusOK, rec.Code)

	var body health.Report
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(health.StatusUp, body.Status)
	assert.Len(body.Components, 3)
	assert.Equal(health.StatusUp, body.Components["lifecycle"].Status)
	assert.Equal(health.StatusUp, body.Components["http"].Status)
	assert.Equal(health.StatusUp, body.Components["database"].Status)
}

func TestReadinessRoute_AnyDownReturns503(t *testing.T) {
	assert := asrt.New(t)
	reg := health.NewRegistry()
	reg.Register("lifecycle", func(_ context.Context) health.ComponentResult {
		return health.ComponentResult{Status: health.StatusUp}
	})
	reg.Register("database", func(_ context.Context) health.ComponentResult {
		return health.ComponentResult{
			Status:  health.StatusDown,
			Details: map[string]any{"reason": "ping failed"},
		}
	})
	cfg := &router.Config{HealthRegistry: reg}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "")
	assert.Equal(http.StatusServiceUnavailable, rec.Code)

	var body health.Report
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(health.StatusDown, body.Status, "any DOWN component must flip the overall status")
	assert.Equal(health.StatusUp, body.Components["lifecycle"].Status)
	assert.Equal(health.StatusDown, body.Components["database"].Status)
	assert.Equal("ping failed", body.Components["database"].Details["reason"])
}

func TestReadinessRoute_NilRegistryFailsClosed(t *testing.T) {
	// A misconfigured router config (nil registry) must not panic and must
	// report DOWN — this codifies the fail-closed default in the handler so
	// a future refactor can't silently turn a missing registry into a 200.
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: nil}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "")
	assert.Equal(http.StatusServiceUnavailable, rec.Code)

	var body health.Report
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(health.StatusDown, body.Status)
}

func TestReadinessRoute_EmptyRegistryReturns200(t *testing.T) {
	// A server that has declared no dependencies is, by definition, ready.
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: health.NewRegistry()}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "")
	assert.Equal(http.StatusOK, rec.Code)
}

// ---------- /healthz (legacy alias for readiness) ----------

func TestHealthRoute_MatchesReadinessSemantics(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, HealthRoute(cfg), "/healthz", "")
	assert.Equal(http.StatusOK, rec.Code)

	var body health.Report
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(health.StatusUp, body.Status)
	assert.Contains(body.Components, "lifecycle")
}

// ---------- Content negotiation ----------

func TestReadyz_DefaultsToJSON(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "")
	assert.Contains(rec.Header().Get(echo.HeaderContentType), "application/json")

	var body map[string]any
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal("UP", body["status"])
}

func TestReadyz_AcceptApplicationJSONReturnsJSON(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "application/json")
	assert.Contains(rec.Header().Get(echo.HeaderContentType), "application/json")

	var body map[string]any
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal("UP", body["status"])
}

func TestReadyz_AcceptApplicationYAMLReturnsYAML(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "application/yaml")
	assert.Contains(rec.Header().Get(echo.HeaderContentType), "application/yaml")

	// Sanity-check it's valid YAML and not accidentally JSON.
	body := strings.TrimSpace(rec.Body.String())
	assert.True(strings.HasPrefix(body, "status:"), "expected yaml-style top-level key, got %q", body)

	var parsed map[string]any
	assert.NoError(yaml.Unmarshal([]byte(body), &parsed))
	assert.Equal("UP", parsed["status"])
}

func TestReadyz_AcceptTextYAMLReturnsYAML(t *testing.T) {
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "text/yaml")
	assert.Contains(rec.Header().Get(echo.HeaderContentType), "application/yaml")
}

func TestReadyz_AcceptStarStarReturnsJSON(t *testing.T) {
	// `Accept: */*` is the curl/wget default — must produce JSON, not yaml.
	assert := asrt.New(t)
	cfg := &router.Config{HealthRegistry: allUpRegistry("lifecycle")}

	rec := serve(t, ReadinessRoute(cfg), "/readyz", "*/*")
	assert.Contains(rec.Header().Get(echo.HeaderContentType), "application/json")
}
