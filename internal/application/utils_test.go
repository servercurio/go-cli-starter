package application

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestParseByteSize(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"0", 0, false},
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1kb", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"500MB", 500 * 1024 * 1024, false},
		{"2GB", 2 * 1024 * 1024 * 1024, false},
		{"1B", 1, false},
		{"  1MB  ", 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"-1", 0, true},
		{"1ZZ", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseByteSize(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseByteSize(%q): want error, got %d", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseByteSize(%q): unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("parseByteSize(%q): got %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestHTTPSRedirectUsesConfiguredHostname verifies the open-redirect mitigation:
// when a hostname is configured, the redirect target ignores the client-supplied
// Host header.
func TestHTTPSRedirectUsesConfiguredHostname(t *testing.T) {
	cfg := &TlsConfig{
		HttpConfig: &HttpConfig{
			Hostname: "api.example.com",
			Port:     8443,
		},
	}

	mw := HTTPSRedirectWithConfig(cfg)
	handler := mw(func(c *echo.Context) error {
		t.Fatalf("next handler should not run on plain HTTP")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://evil.attacker.tld/some/path", nil)
	req.Host = "evil.attacker.tld"
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected 308, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://api.example.com:8443/") {
		t.Fatalf("redirect target should use configured hostname, got %q", loc)
	}
	if strings.Contains(loc, "evil.attacker.tld") {
		t.Fatalf("redirect leaked attacker-supplied Host header: %q", loc)
	}
}

// TestHTTPSRedirectFallsBackToHostHeader verifies that when no hostname is
// configured, the middleware still works (using the request Host) — preserving
// the legacy behaviour for users who set up TLS but don't set Hostname.
func TestHTTPSRedirectFallsBackToHostHeader(t *testing.T) {
	cfg := &TlsConfig{
		HttpConfig: &HttpConfig{
			Hostname: "",
			Port:     443,
		},
	}

	mw := HTTPSRedirectWithConfig(cfg)
	handler := mw(func(c *echo.Context) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/x", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected 308, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://localhost/") {
		t.Fatalf("redirect should strip port and use Host: got %q", loc)
	}
}

// TestHTTPSRedirectSkipsInsecurePaths verifies health-check paths bypass the
// redirect so K8s liveness/readiness probes can still hit HTTP.
func TestHTTPSRedirectSkipsInsecurePaths(t *testing.T) {
	cfg := &TlsConfig{HttpConfig: &HttpConfig{Hostname: "api.example.com", Port: 443}}
	called := false
	mw := HTTPSRedirectWithConfig(cfg)
	handler := mw(func(c *echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/livez", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// /livez isn't in defaultInsecurePaths but /healthz and /readyz are. Verify
	// /readyz at least.
	called = false
	req2 := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/readyz", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	if err := handler(c2); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected /readyz to bypass HTTPS redirect")
	}
}

func TestCorsMiddlewareDisabledByDefault(t *testing.T) {
	cfg := DefaultCorsConfig()
	if got := CorsMiddleware(cfg); got != nil {
		t.Fatalf("expected CorsMiddleware to return nil for default config")
	}
}

func TestCorsMiddlewareEnabledWithOrigins(t *testing.T) {
	cfg := &CorsConfig{
		AllowOrigins: []string{"https://app.example.com"},
	}
	if got := CorsMiddleware(cfg); got == nil {
		t.Fatalf("expected CorsMiddleware to return a non-nil middleware when origins are set")
	}
}
