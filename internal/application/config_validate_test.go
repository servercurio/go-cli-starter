package application

import (
	"strings"
	"testing"
	"time"
)

// TestHttpConfig_Validate covers the eager checks that previously deferred
// to Initialize: zero port, negative timeouts, unparseable MaxBodySize.
// Each row mutates a copy of DefaultHttpConfig and asserts the joined
// error string mentions the offending field.
func TestHttpConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *HttpConfig)
		wantErr string
	}{
		{name: "default", mutate: func(c *HttpConfig) {}},
		{name: "zero port", mutate: func(c *HttpConfig) { c.Port = 0 }, wantErr: "port"},
		{
			name:    "negative read timeout",
			mutate:  func(c *HttpConfig) { c.ReadTimeout = -1 * time.Second },
			wantErr: "readTimeout",
		},
		{
			name:    "negative idle timeout",
			mutate:  func(c *HttpConfig) { c.IdleTimeout = -1 * time.Second },
			wantErr: "idleTimeout",
		},
		{
			name:    "bad MaxBodySize",
			mutate:  func(c *HttpConfig) { c.MaxBodySize = "not-a-size" },
			wantErr: "maxBodySize",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultHttpConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if !errMatches(t, err, tt.wantErr) {
				t.Fatalf("got %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// TestTlsConfig_Validate exercises the TLS-specific extras: ACME requires
// a hostname; static certs require both certificate and key (or neither);
// disabled TLS skips validation entirely.
func TestTlsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *TlsConfig)
		wantErr string
	}{
		{name: "disabled (default)", mutate: func(c *TlsConfig) {}},
		{
			name: "enabled without cert/key (auto-issuance)",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
			},
		},
		{
			name: "enabled with both cert+key",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
				c.Certificate = "/etc/tls/cert.pem"
				c.Key = "/etc/tls/key.pem"
			},
		},
		{
			name: "enabled with only cert",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
				c.Certificate = "/etc/tls/cert.pem"
			},
			wantErr: "certificate and key",
		},
		{
			name: "enabled with only key",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
				c.Key = "/etc/tls/key.pem"
			},
			wantErr: "certificate and key",
		},
		{
			name: "ACME without hostname",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
				c.UseAcmeIssuer = true
			},
			wantErr: "hostname is required",
		},
		{
			name: "ACME with hostname",
			mutate: func(c *TlsConfig) {
				c.Enabled = true
				c.UseAcmeIssuer = true
				c.Hostname = "api.example.com"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTlsConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if !errMatches(t, err, tt.wantErr) {
				t.Fatalf("got %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// TestCorsConfig_Validate pins the spec-illegal credentials-with-wildcard
// combination — browsers refuse it at request time, so we'd rather fail
// at boot.
func TestCorsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *CorsConfig)
		wantErr string
	}{
		{name: "default (cors disabled)", mutate: func(c *CorsConfig) {}},
		{
			name: "explicit origin + credentials",
			mutate: func(c *CorsConfig) {
				c.AllowOrigins = []string{"https://app.example.com"}
				c.AllowCredentials = true
			},
		},
		{
			name: "wildcard + credentials",
			mutate: func(c *CorsConfig) {
				c.AllowOrigins = []string{"*"}
				c.AllowCredentials = true
			},
			wantErr: "wildcard",
		},
		{
			name: "wildcard without credentials",
			mutate: func(c *CorsConfig) {
				c.AllowOrigins = []string{"*"}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultCorsConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if !errMatches(t, err, tt.wantErr) {
				t.Fatalf("got %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		if err := DefaultSecurityConfig().Validate(); err != nil {
			t.Fatalf("default config should validate: %v", err)
		}
	})
	t.Run("negative HSTS", func(t *testing.T) {
		c := DefaultSecurityConfig()
		c.HSTSMaxAge = -1
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "non-negative") {
			t.Fatalf("expected non-negative error, got %v", err)
		}
	})
}

// TestOpenAPIConfig_Validate ensures swagger UI without mount path or spec
// URL fails fast — leaving either empty would 404 every UI asset request.
func TestOpenAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *OpenAPIConfig)
		wantErr string
	}{
		{name: "default", mutate: func(c *OpenAPIConfig) {}},
		{
			name: "openapi disabled (no validation)",
			mutate: func(c *OpenAPIConfig) {
				c.Enabled = false
				c.Swagger.Enabled = true
				c.Swagger.Path = ""
			},
		},
		{
			name: "swagger enabled with empty path",
			mutate: func(c *OpenAPIConfig) {
				c.Swagger.Enabled = true
				c.Swagger.Path = ""
			},
			wantErr: "swagger.path",
		},
		{
			name: "swagger enabled with empty specUrl",
			mutate: func(c *OpenAPIConfig) {
				c.Swagger.Enabled = true
				c.Swagger.SpecURL = ""
			},
			wantErr: "swagger.specUrl",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultOpenAPIConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if !errMatches(t, err, tt.wantErr) {
				t.Fatalf("got %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// TestConfig_ValidateAggregatesErrors confirms the top-level Validate
// returns *all* failures via errors.Join — operators shouldn't have to
// fix-boot-fix-boot for each problem in turn. CSRF is included because
// CsrfConfig was added after the original aggregator and an early
// version of ServerConfig.Validate forgot to wire it in; this pins the
// regression.
func TestConfig_ValidateAggregatesErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Http.Port = 0                     // HttpConfig violation
	cfg.Server.Cors.AllowOrigins = []string{"*"} // CORS violation
	cfg.Server.Cors.AllowCredentials = true      // ↑
	cfg.Server.Csrf.Enabled = true               // CSRF violation
	cfg.Server.Csrf.CookieSameSite = "stricct"   // ↑
	cfg.Proxy.UseXFFHeader = true                // ProxyConfig violation (DirectIP also default true)
	cfg.OpenAPI.Swagger.Enabled = true           // OpenAPI violation
	cfg.OpenAPI.Swagger.Path = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"port", "wildcard", "stricct", "only one of", "swagger.path"} {
		if !strings.Contains(msg, want) {
			t.Errorf("joined error missing %q: %s", want, msg)
		}
	}
}

// errMatches encapsulates the "expected nil when wantErr empty, otherwise
// expected substring" pattern that every table test in this file uses.
func errMatches(t *testing.T, err error, wantErr string) bool {
	t.Helper()
	if wantErr == "" {
		return err == nil
	}
	return err != nil && strings.Contains(err.Error(), wantErr)
}
