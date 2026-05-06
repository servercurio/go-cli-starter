package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-echo-starter/internal/env"
)

// HttpConfig captures the HTTP listener parameters: bind address, port,
// timeouts, and a max body size. Used as the standalone HTTP server
// config and embedded as the base of TlsConfig so the TLS listener
// inherits the same timeout knobs.
type HttpConfig struct {
	Hostname          string        `yaml:"hostname" json:"hostname"`
	BindAddress       string        `yaml:"bindAddress" json:"bindAddress"`
	Port              uint16        `yaml:"port" json:"port"`
	ShutdownTimeout   time.Duration `yaml:"shutdownTimeout" json:"shutdownTimeout"`
	ReadTimeout       time.Duration `yaml:"readTimeout" json:"readTimeout"`
	ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout" json:"readHeaderTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout" json:"writeTimeout"`
	IdleTimeout       time.Duration `yaml:"idleTimeout" json:"idleTimeout"`
	MaxBodySize       string        `yaml:"maxBodySize" json:"maxBodySize"`
}

// MarshalZerologObject writes the HTTP listener configuration into e for
// the structured-startup-log notifier. BindAddress is normalised to
// "0.0.0.0" when blank so the log line shows what the kernel will actually
// bind to.
func (h *HttpConfig) MarshalZerologObject(e *zerolog.Event) {
	ba := h.BindAddress

	if strings.TrimSpace(ba) == "" {
		ba = "0.0.0.0"
	}

	e.Str("hostname", h.Hostname)
	e.Str("bindAddress", ba)
	e.Uint16("port", h.Port)
	e.Str("shutdownTimeout", h.ShutdownTimeout.String())
	e.Str("readTimeout", h.ReadTimeout.String())
	e.Str("readHeaderTimeout", h.ReadHeaderTimeout.String())
	e.Str("writeTimeout", h.WriteTimeout.String())
	e.Str("idleTimeout", h.IdleTimeout.String())
	e.Str("maxBodySize", h.MaxBodySize)
}

// FromEnv hydrates the HTTP listener fields from environment variables
// under prefix (e.g. <prefix>_PORT, <prefix>_READ_TIMEOUT).
func (h *HttpConfig) FromEnv(prefix string) {
	env.SetStringValue(prefix, "hostname", &h.Hostname)
	env.SetStringValue(prefix, "bind_address", &h.BindAddress)
	env.SetUint16Value(prefix, "port", &h.Port)
	env.SetDurationValue(prefix, "shutdown_timeout", &h.ShutdownTimeout)
	env.SetDurationValue(prefix, "read_timeout", &h.ReadTimeout)
	env.SetDurationValue(prefix, "read_header_timeout", &h.ReadHeaderTimeout)
	env.SetDurationValue(prefix, "write_timeout", &h.WriteTimeout)
	env.SetDurationValue(prefix, "idle_timeout", &h.IdleTimeout)
	env.SetStringValue(prefix, "max_body_size", &h.MaxBodySize)
}

// TlsConfig captures the TLS listener configuration. Embeds HttpConfig so
// the same timeout/body-size knobs apply, and adds cert-source fields
// (static cert/key file paths, ACME cache, ACME-vs-ephemeral toggle).
type TlsConfig struct {
	*HttpConfig

	// Enabled indicates whether TLS is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Certificate is the path to the TLS certificate file. If omitted or empty, then Let's Encrypt
	// based AutoTLS certificate issuance will be attempted.
	Certificate string `yaml:"certificate" json:"certificate"`
	// Key is the path to the TLS key file. If omitted or empty, then Let's Encrypt based AutoTLS
	// certificate issuance will be attempted.
	Key string `yaml:"key" json:"key"`
	// CertificateCache is the path to the directory where AutoTLS will store its certificates.
	CertificateCache string `yaml:"certificateCache" json:"certificateCache"`
	// UseAcmeIssuer indicates whether to use the ACME issuer for certificate issuance or generate an
	// ephemeral certificate.
	UseAcmeIssuer bool `yaml:"useAcmeIssuer" json:"useAcmeIssuer"`
}

// MarshalZerologObject writes the TLS listener configuration into e for
// the startup-log notifier. The certificate and key file paths are only
// emitted when statically configured; the autocert cache and ACME toggle
// only when auto-issuance applies.
func (t *TlsConfig) MarshalZerologObject(e *zerolog.Event) {
	autoCertIssuance := strings.TrimSpace(t.Certificate) == "" || strings.TrimSpace(t.Key) == ""

	e.EmbedObject(t.HttpConfig).
		Bool("enabled", t.Enabled).
		Bool("autoCertIssuance", autoCertIssuance)

	if !autoCertIssuance {
		e.Str("certificate", t.Certificate).
			Str("key", t.Key)
	} else {
		e.Str("certificateCache", t.CertificateCache).
			Bool("useAcmeIssuer", t.UseAcmeIssuer)
	}
}

// FromEnv hydrates the TLS listener fields from environment variables
// under prefix, plus the embedded HttpConfig fields under the same prefix.
func (t *TlsConfig) FromEnv(prefix string) {
	t.HttpConfig.FromEnv(prefix)
	env.SetBoolValue(prefix, "enabled", &t.Enabled)
	env.SetStringValue(prefix, "certificate", &t.Certificate)
	env.SetStringValue(prefix, "key", &t.Key)
	env.SetStringValue(prefix, "certificate_cache", &t.CertificateCache)
	env.SetBoolValue(prefix, "use_acme_issuer", &t.UseAcmeIssuer)
}

// Validate verifies the HTTP listener config will produce a working server.
// Port=0 is rejected because Echo will fail at listen time anyway and we'd
// rather refuse early; negative timeouts trip Go's http.Server hard; bad
// MaxBodySize would crash configureHttpServer at Initialize. Catching all
// of these at Configure time means cmd/daemon/main.go exits non-zero before
// any subsystem warms up.
func (h *HttpConfig) Validate() error {
	if h == nil {
		return nil
	}
	var errs []error
	if h.Port == 0 {
		errs = append(errs, errors.New("port must be 1-65535"))
	}
	if h.ShutdownTimeout < 0 {
		errs = append(errs, fmt.Errorf("shutdownTimeout must be non-negative, got %s", h.ShutdownTimeout))
	}
	if h.ReadTimeout < 0 {
		errs = append(errs, fmt.Errorf("readTimeout must be non-negative, got %s", h.ReadTimeout))
	}
	if h.ReadHeaderTimeout < 0 {
		errs = append(errs, fmt.Errorf("readHeaderTimeout must be non-negative, got %s", h.ReadHeaderTimeout))
	}
	if h.WriteTimeout < 0 {
		errs = append(errs, fmt.Errorf("writeTimeout must be non-negative, got %s", h.WriteTimeout))
	}
	if h.IdleTimeout < 0 {
		errs = append(errs, fmt.Errorf("idleTimeout must be non-negative, got %s", h.IdleTimeout))
	}
	if strings.TrimSpace(h.MaxBodySize) != "" {
		if _, err := parseByteSize(h.MaxBodySize); err != nil {
			errs = append(errs, fmt.Errorf("maxBodySize: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Validate extends HttpConfig.Validate with TLS-specific checks. ACME without
// a hostname can't issue a cert; partial cert/key config is ambiguous (either
// both must be set for static certs, or both empty for auto-issuance).
func (t *TlsConfig) Validate() error {
	if t == nil || !t.Enabled {
		return nil
	}
	var errs []error
	if err := t.HttpConfig.Validate(); err != nil {
		errs = append(errs, err)
	}

	certSet := strings.TrimSpace(t.Certificate) != ""
	keySet := strings.TrimSpace(t.Key) != ""
	if certSet != keySet {
		errs = append(errs, errors.New("certificate and key must both be set for static TLS, or both empty for auto-issuance"))
	}
	if t.UseAcmeIssuer && strings.TrimSpace(t.Hostname) == "" {
		errs = append(errs, errors.New("hostname is required when useAcmeIssuer=true (ACME challenge needs a routable name)"))
	}
	return errors.Join(errs...)
}

// DefaultHttpConfig returns the starter's HTTP listener defaults: bind to
// every interface, port 8080, conservative read/write/idle timeouts, 1MB
// max body.
func DefaultHttpConfig() *HttpConfig {
	return &HttpConfig{
		BindAddress:       "",
		Port:              8080,
		ShutdownTimeout:   10 * time.Second,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxBodySize:       "1MB",
	}
}

// DefaultTlsConfig returns the starter's TLS listener defaults: disabled
// by default, port 8443 when enabled, no certificates configured (so the
// daemon falls into the auto-issuance path).
func DefaultTlsConfig() *TlsConfig {
	return &TlsConfig{
		HttpConfig: &HttpConfig{
			BindAddress:       "",
			Port:              8443,
			ShutdownTimeout:   10 * time.Second,
			ReadTimeout:       30 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxBodySize:       "1MB",
		},
		Enabled:          false,
		Certificate:      "",
		Key:              "",
		CertificateCache: "",
		UseAcmeIssuer:    false,
	}
}
