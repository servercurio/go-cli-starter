package application

import (
	"strings"
	"testing"
)

// TestProxyConfig_Validate replaces the deleted TestValidateProxyFlags_*
// pair. The mutual-exclusion check moved to ProxyConfig.Validate (called
// from Configure via Config.Validate); that's the canonical location.
func TestProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *ProxyConfig)
		wantErr string
	}{
		{name: "default (DirectIP only)", mutate: func(c *ProxyConfig) {}},
		{
			name:   "all flags off",
			mutate: func(c *ProxyConfig) { c.UseDirectIP = false },
		},
		{
			name:   "XFF only",
			mutate: func(c *ProxyConfig) { c.UseDirectIP = false; c.UseXFFHeader = true },
		},
		{
			name:    "DirectIP + XFF",
			mutate:  func(c *ProxyConfig) { c.UseXFFHeader = true },
			wantErr: "only one of",
		},
		{
			name:    "all three",
			mutate:  func(c *ProxyConfig) { c.UseXFFHeader = true; c.UseXRealIPHeader = true },
			wantErr: "only one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultProxyConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q did not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestProxyConfig_FromEnv_TrustedIPRanges pins the parsing contract for
// APP_PROXY_TRUSTED_IP_RANGES. The pre-fix implementation panicked on
// any non-empty value — it indexed the destination slice rather than
// the freshly-split source — and even when the index was lucky it
// never appended results. This table covers the matrix that broke
// previously:
//
//   - empty → no-op, slice stays empty
//   - single valid CIDR → singleton list
//   - multiple valid CIDRs (with surrounding whitespace, blank segment
//     from a trailing comma) → all retained, in source order
//   - mixed valid + invalid → only valid entries retained
//   - all invalid → empty list, no panic
//   - IPv6 → retained (net.ParseCIDR accepts both families)
func TestProxyConfig_FromEnv_TrustedIPRanges(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", value: "", want: []string{}},
		{name: "single valid", value: "10.0.0.0/8", want: []string{"10.0.0.0/8"}},
		{
			name:  "multiple valid with whitespace and trailing comma",
			value: " 10.0.0.0/8 , 192.168.0.0/16 , ",
			want:  []string{"10.0.0.0/8", "192.168.0.0/16"},
		},
		{
			name:  "mixed valid and invalid",
			value: "10.0.0.0/8,not-a-cidr,192.168.0.0/16",
			want:  []string{"10.0.0.0/8", "192.168.0.0/16"},
		},
		{name: "all invalid", value: "not-a-cidr, also-bogus", want: []string{}},
		{name: "ipv6", value: "2001:db8::/32", want: []string{"2001:db8::/32"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_PROXY_TRUSTED_IP_RANGES", tt.value)

			cfg := DefaultProxyConfig()
			cfg.FromEnv("APP_PROXY")

			if len(cfg.TrustedIPRanges) != len(tt.want) {
				t.Fatalf("TrustedIPRanges length: got %v (%d), want %v (%d)",
					cfg.TrustedIPRanges, len(cfg.TrustedIPRanges), tt.want, len(tt.want))
			}
			for i, got := range cfg.TrustedIPRanges {
				if got != tt.want[i] {
					t.Fatalf("TrustedIPRanges[%d]: got %q, want %q", i, got, tt.want[i])
				}
			}
		})
	}
}
