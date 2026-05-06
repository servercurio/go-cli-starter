package health

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"gopkg.in/yaml.v3"
)

// Format identifies the response serialization picked by content negotiation.
type Format string

const (
	// FormatJSON serializes the health Report as JSON (the default).
	FormatJSON Format = "json"

	// FormatYAML serializes the health Report as YAML, selected when the
	// client's Accept header explicitly mentions a yaml media type.
	FormatYAML Format = "yaml"
)

// FormatFromAccept inspects an HTTP Accept header value and returns the
// preferred response format. YAML wins when the header explicitly mentions
// any yaml media type (`application/yaml`, `application/x-yaml`,
// `text/yaml`, or any `*+yaml` suffix); otherwise JSON is returned.
//
// Quality-value sorting (q=0.8) is intentionally not implemented — for a
// readiness/liveness endpoint the simple "did the client ask for yaml?"
// heuristic is unambiguous and avoids an RFC-7231 negotiation library.
func FormatFromAccept(accept string) Format {
	a := strings.ToLower(accept)
	if strings.Contains(a, "yaml") {
		return FormatYAML
	}
	return FormatJSON
}

// Render writes the report to the response using whichever format the
// client's Accept header asks for, sets the corresponding Content-Type, and
// uses the supplied HTTP status code. The status code is the caller's
// concern (200 for UP, 503 for DOWN) so this helper stays format-only.
func Render(c *echo.Context, statusCode int, report Report) error {
	format := FormatFromAccept(c.Request().Header.Get(echo.HeaderAccept))

	switch format {
	case FormatYAML:
		body, err := yaml.Marshal(report)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status": string(StatusDown),
				"error":  "failed to marshal yaml health report",
			})
		}
		return c.Blob(statusCode, "application/yaml; charset=utf-8", body)
	default:
		body, err := json.Marshal(report)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status": string(StatusDown),
				"error":  "failed to marshal json health report",
			})
		}
		return c.Blob(statusCode, echo.MIMEApplicationJSON, body)
	}
}
