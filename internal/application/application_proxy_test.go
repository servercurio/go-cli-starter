package application

import (
	"testing"
)

// Proxy-flag mutual exclusion now lives in ProxyConfig.Validate (see
// config_proxy_test.go for the table) and runs at Configure time, so the
// dedicated app-level wrappers were redundant. These tests are kept here
// historically; the live coverage is in TestProxyConfig_Validate.

func TestResolveProxyTrustOptions_MixedCIDR(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Proxy.TrustedIPRanges = []string{
		"10.0.0.0/8",     // valid
		"not-a-cidr",     // invalid — should be dropped, not crash
		"  ",             // empty after trim — should be skipped
		"192.168.0.0/16", // valid
	}

	app := &Application{config: cfg}
	opts := app.resolveProxyTrustOptions()

	// We can't easily inspect the TrustOption values, but we can assert the
	// total count: 2 valid CIDRs + 3 always-appended (private/loopback/link-local).
	if len(opts) != 5 {
		t.Fatalf("expected 5 trust options (2 CIDR + 3 builtin), got %d", len(opts))
	}
}

func TestResolveProxyTrustOptions_NilProxy(t *testing.T) {
	app := &Application{config: &Config{}}
	opts := app.resolveProxyTrustOptions()
	if len(opts) != 0 {
		t.Fatalf("expected 0 options when Proxy is nil, got %d", len(opts))
	}
}
