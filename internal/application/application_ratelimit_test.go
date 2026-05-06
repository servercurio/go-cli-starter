package application

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

// TestRateLimit_DisabledByDefault pins the starter's default: a high-rate
// caller is never throttled when RateLimit.Enabled stays false. Wiring a
// rate limit on by default would break every API consumer that doesn't
// know to send an X-Forwarded-For header (or runs unkeyed traffic from a
// shared NAT).
func TestRateLimit_DisabledByDefault(t *testing.T) {
	app := newAppForRateLimit(t, DefaultConfig())
	app.httpServer.GET("/probe", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	for i := 0; i < 25; i++ {
		rec := httptest.NewRecorder()
		app.httpServer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200 with rate-limit disabled, got %d", i, rec.Code)
		}
	}
}

// TestRateLimit_EnforcesRateAndBurst pins the wired-through behaviour:
// the first Burst calls succeed, then subsequent calls within the same
// second are rejected with 429. Echo keys by RealIP() — every test
// request comes from the same in-memory address so they share a bucket.
func TestRateLimit_EnforcesRateAndBurst(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.RateLimit.Enabled = true
	cfg.Server.RateLimit.Rate = 1
	cfg.Server.RateLimit.Burst = 2
	cfg.Server.RateLimit.ExpiresIn = time.Minute
	app := newAppForRateLimit(t, cfg)
	app.httpServer.GET("/probe", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		app.httpServer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("burst[%d]: expected 200, got %d", i, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("post-burst call: expected 429, got %d", rec.Code)
	}
}

// TestRateLimitConfig_ValidateRejectsBadCombos pins the set of
// startup-time refusals the audit calls for: enabled-with-zero-rate
// would silently drop every request; negative knobs are nonsense. These
// are config errors, not runtime errors, so the daemon refuses to boot.
func TestRateLimitConfig_ValidateRejectsBadCombos(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*RateLimitConfig)
		wantErr bool
	}{
		{"defaults pass", func(c *RateLimitConfig) {}, false},
		{"enabled with zero rate fails", func(c *RateLimitConfig) {
			c.Enabled = true
			c.Rate = 0
		}, true},
		{"enabled with positive rate ok", func(c *RateLimitConfig) {
			c.Enabled = true
			c.Rate = 5
		}, false},
		{"negative burst fails", func(c *RateLimitConfig) { c.Burst = -1 }, true},
		{"negative expiresIn fails", func(c *RateLimitConfig) { c.ExpiresIn = -1 }, true},
		{"negative maxConnections fails", func(c *RateLimitConfig) { c.MaxConnections = -1 }, true},
		{"only maxConnections is fine", func(c *RateLimitConfig) { c.MaxConnections = 100 }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultRateLimitConfig()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}

func newAppForRateLimit(t *testing.T, cfg *Config) *Application {
	t.Helper()
	app := NewApplication(cfg)
	app.buildMiddleware()
	if err := app.configureHttpServer(); err != nil {
		t.Fatalf("configureHttpServer: %v", err)
	}
	return app
}
