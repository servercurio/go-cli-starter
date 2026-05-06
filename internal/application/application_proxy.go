package application

import (
	"net"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/servercurio/go-echo-starter/internal/logging"
)

// configureProxySupport wires the IP-extractor for both servers based on
// the proxy config. Mutually-exclusive-flag validation is enforced earlier
// at Configure time via ProxyConfig.Validate, so by the time we reach here
// at most one of the three flags is true.
func (app *Application) configureProxySupport() error {
	pc := app.config.Proxy
	if pc == nil {
		return nil
	}

	var extractor echo.IPExtractor
	if pc.UseXRealIPHeader {
		extractor = echo.ExtractIPFromRealIPHeader(app.resolveProxyTrustOptions()...)
	}

	if pc.UseXFFHeader {
		extractor = echo.ExtractIPFromXFFHeader(app.resolveProxyTrustOptions()...)
	}

	if extractor == nil {
		extractor = echo.ExtractIPDirect()
	}

	app.httpServer.IPExtractor = extractor
	app.tlsServer.IPExtractor = extractor
	return nil
}

// resolveProxyTrustOptions builds the slice of echo.TrustOption values
// passed to ExtractIPFromRealIPHeader / ExtractIPFromXFFHeader. Configured
// CIDRs are validated; invalid entries are dropped with a warn-level log
// rather than refused outright (we already validate at Configure time, so
// anything bad here is a runtime regression worth surfacing). Loopback,
// private, and link-local nets are always trusted.
func (app *Application) resolveProxyTrustOptions() []echo.TrustOption {
	pc := app.config.Proxy
	if pc == nil {
		return []echo.TrustOption{}
	}

	var invalidRanges = make([]string, 0)
	var opts = make([]echo.TrustOption, 0)
	for _, trustedRange := range pc.TrustedIPRanges {
		trustedRange = strings.TrimSpace(trustedRange)
		if trustedRange == "" {
			continue
		}

		if _, network, err := net.ParseCIDR(trustedRange); err == nil && network != nil {
			opts = append(opts, echo.TrustIPRange(network))
		} else {
			invalidRanges = append(invalidRanges, trustedRange)
		}
	}

	if len(invalidRanges) > 0 {
		logging.Daemon.
			Warn().
			Strs("cidrAddresses", invalidRanges).
			Msg("proxy config contains invalid CIDR addresses")
	}

	opts = append(opts,
		echo.TrustPrivateNet(true),
		echo.TrustLoopback(true),
		echo.TrustLinkLocal(true))

	return opts
}
