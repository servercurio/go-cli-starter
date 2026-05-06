package logging

import (
	"github.com/labstack/echo/v5"
	mw "github.com/labstack/echo/v5/middleware"
)

// EchoMiddleware returns an Echo v5 middleware that emits one structured
// access-log event per request to the package-level Access logger.
// Successful responses log at info; responses carrying an error log at
// error with the underlying error attached.
func EchoMiddleware() echo.MiddlewareFunc {
	return mw.RequestLoggerWithConfig(mw.RequestLoggerConfig{
		LogMethod:        true,
		LogURI:           true,
		LogRemoteIP:      true,
		LogProtocol:      true,
		LogStatus:        true,
		LogLatency:       true,
		LogHost:          true,
		LogRequestID:     true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogResponseSize:  true,
		LogContentLength: true,
		LogRoutePath:     true,
		LogURIPath:       true,
		LogValuesFunc: func(c *echo.Context, v mw.RequestLoggerValues) error {
			l := Access
			if v.Error != nil {
				l.Error().
					Err(v.Error).
					Str("method", v.Method).
					Str("uri", v.URI).
					Str("remoteIp", v.RemoteIP).
					Str("protocol", v.Protocol).
					Int("status", v.Status).
					Str("latency", v.Latency.String()).
					Dur("latencyMillis", v.Latency).
					Str("host", v.Host).
					Str("requestId", v.RequestID).
					Str("referer", v.Referer).
					Str("userAgent", v.UserAgent).
					Int64("responseSize", v.ResponseSize).
					Str("contentLength", v.ContentLength).
					Str("routePath", v.RoutePath).
					Str("uriPath", v.URIPath).
					Bool("tls", c.IsTLS()).
					Msgf("%s - %d Error", v.Protocol, v.Status)
			} else {
				l.Info().
					Str("method", v.Method).
					Str("uri", v.URI).
					Str("remoteIp", v.RemoteIP).
					Str("protocol", v.Protocol).
					Int("status", v.Status).
					Str("latency", v.Latency.String()).
					Dur("latencyMillis", v.Latency).
					Str("host", v.Host).
					Str("requestId", v.RequestID).
					Str("referer", v.Referer).
					Str("userAgent", v.UserAgent).
					Int64("responseSize", v.ResponseSize).
					Str("contentLength", v.ContentLength).
					Str("routePath", v.RoutePath).
					Str("uriPath", v.URIPath).
					Bool("tls", c.IsTLS()).
					Msgf("%s - %d OK", v.Protocol, v.Status)
			}

			return nil
		},
	})
}
