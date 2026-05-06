package application

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

// TestCsrfMiddleware_DisabledByDefault verifies the starter ships CSRF
// off — a POST without a token must reach the handler. CSRF only makes
// sense once the application uses session cookies; defaulting it on
// would break every API client that doesn't speak the cookie/header
// double-submit dance.
func TestCsrfMiddleware_DisabledByDefault(t *testing.T) {
	app := newAppForCsrf(t, DefaultConfig())

	app.httpServer.POST("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/probe", nil)
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with CSRF disabled, got %d body=%q", rec.Code, rec.Body.String())
	}
}

// TestCsrfMiddleware_EnabledRejectsPostWithoutToken pins the canonical
// CSRF defence: a state-changing request that doesn't carry a token is
// refused before the handler runs. Echo's middleware returns 400 when
// the configured token extractor (header lookup) finds nothing — and
// 403 when the token is present but doesn't match the cookie. Either
// flavour is a rejection; we assert non-2xx and that the handler body
// did not run.
func TestCsrfMiddleware_EnabledRejectsPostWithoutToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Csrf.Enabled = true
	app := newAppForCsrf(t, cfg)

	app.httpServer.POST("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/probe", nil)
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("expected 4xx rejection with CSRF enabled and no token, got %d body=%q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("handler must not run when CSRF rejects the request, got body=%q", rec.Body.String())
	}
}

// TestCsrfMiddleware_GetIssuesTokenAndAcceptsSubsequentPost walks the
// happy path: a GET seeds the cookie with a freshly-minted token; the
// client echoes that token in the X-CSRF-Token header on the next POST,
// which is then accepted. This is the contract every CSRF-aware client
// implements.
func TestCsrfMiddleware_GetIssuesTokenAndAcceptsSubsequentPost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Csrf.Enabled = true
	app := newAppForCsrf(t, cfg)

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	app.httpServer.POST("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	getRec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/probe", nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("seed GET should be 200, got %d", getRec.Code)
	}

	var csrfCookie *http.Cookie
	for _, c := range getRec.Result().Cookies() {
		if c.Name == cfg.Server.Csrf.CookieName {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil {
		t.Fatalf("seed GET should set %q cookie, headers were %v", cfg.Server.Csrf.CookieName, getRec.Header())
	}

	postReq := httptest.NewRequest(http.MethodPost, "/probe", nil)
	postReq.AddCookie(csrfCookie)
	postReq.Header.Set(echo.HeaderXCSRFToken, csrfCookie.Value)
	postRec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST with cookie+header should be 200, got %d body=%q", postRec.Code, postRec.Body.String())
	}
}

// TestCsrfConfig_ValidateRejectsUnknownSameSite pins the SameSite
// validator behind Config.Validate. Echo would silently treat an unknown
// SameSite value as the default mode; refusing at startup means
// "CookieSameSite=stricct" surfaces at boot rather than as a debugging
// mystery weeks later.
func TestCsrfConfig_ValidateRejectsUnknownSameSite(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty stays valid", "", false},
		{"default", "default", false},
		{"lax", "Lax", false},
		{"strict", "STRICT", false},
		{"none", "none", false},
		{"typo rejected", "stricct", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultCsrfConfig()
			cfg.Enabled = true
			cfg.CookieSameSite = tt.value
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil for %q, got %v", tt.value, err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.value) {
				t.Fatalf("error %q should mention %q", err.Error(), tt.value)
			}
		})
	}
}

// newAppForCsrf builds an Application with the supplied config, runs the
// same buildMiddleware → configureHttpServer sequence as production, and
// returns the result. Mirrors the helper in application_security_test.go;
// each test file has its own copy so the test suites stay independently
// readable.
func newAppForCsrf(t *testing.T, cfg *Config) *Application {
	t.Helper()
	app := NewApplication(cfg)
	app.buildMiddleware()
	if err := app.configureHttpServer(); err != nil {
		t.Fatalf("configureHttpServer: %v", err)
	}
	return app
}
