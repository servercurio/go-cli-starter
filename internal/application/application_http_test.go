package application

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestConfigureHttpServer_BodyLimitRejects(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Http.MaxBodySize = "1KB"

	app := NewApplication(cfg)
	if err := app.configureHttpServer(); err != nil {
		t.Fatalf("configureHttpServer: %v", err)
	}

	app.httpServer.POST("/echo", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// 2 KB of body — should be rejected.
	body := bytes.Repeat([]byte("A"), 2048)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	app.httpServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestConfigureHttpServer_RejectsBadBodySize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Http.MaxBodySize = "not-a-size"

	app := NewApplication(cfg)
	if err := app.configureHttpServer(); err == nil {
		t.Fatalf("expected configureHttpServer to fail on invalid size")
	}
}
