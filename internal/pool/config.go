package pool

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/servercurio/go-cli-starter/internal/env"
)

// Config holds the goroutine-pool sizing and lifecycle parameters that the
// Application passes to New at Initialize time. Fields mirror the upstream
// ants.Options shape but stay pointer-free so they round-trip cleanly
// through YAML and zerolog.
type Config struct {
	// Size is the maximum number of concurrently-running goroutines. A
	// value <= 0 means unlimited (ants treats this as math.MaxInt32).
	// Defaults to runtime.NumCPU() * 2.
	Size int `yaml:"size" json:"size"`

	// NonBlocking selects rejection-on-full behaviour. When true, Submit
	// returns PoolExhausted instead of blocking the caller waiting for a
	// worker to free up. Defaults to false (block) so default callers
	// match the natural "wait my turn" intuition; set true for
	// latency-sensitive paths that prefer to shed load.
	NonBlocking bool `yaml:"nonBlocking" json:"nonBlocking"`

	// ExpiryDuration is how long an idle worker may sit in the pool
	// before being reaped. Defaults to 1 minute.
	ExpiryDuration time.Duration `yaml:"expiryDuration" json:"expiryDuration"`

	// PreAlloc allocates the worker queue at construction time instead of
	// growing it on demand. Useful when Size is large and known up front.
	// Defaults to false.
	PreAlloc bool `yaml:"preAlloc" json:"preAlloc"`

	// MaxBlockingTasks bounds the number of callers that may block in
	// Submit when the pool is full and NonBlocking is false. A value <= 0
	// means unbounded blocking. Defaults to 0.
	MaxBlockingTasks int `yaml:"maxBlockingTasks" json:"maxBlockingTasks"`
}

// FromEnv overlays Config fields with values from environment variables
// under the given prefix (e.g. APP_POOL_SIZE, APP_POOL_NON_BLOCKING).
func (c *Config) FromEnv(prefix string) {
	env.SetIntValue(prefix, "size", &c.Size)
	env.SetBoolValue(prefix, "non_blocking", &c.NonBlocking)
	env.SetDurationValue(prefix, "expiry_duration", &c.ExpiryDuration)
	env.SetBoolValue(prefix, "pre_alloc", &c.PreAlloc)
	env.SetIntValue(prefix, "max_blocking_tasks", &c.MaxBlockingTasks)
}

// MarshalZerologObject writes the pool configuration to a zerolog event so
// it can be embedded in startup notifications.
func (c *Config) MarshalZerologObject(e *zerolog.Event) {
	e.Int("size", c.Size).
		Bool("nonBlocking", c.NonBlocking).
		Str("expiryDuration", c.ExpiryDuration.String()).
		Bool("preAlloc", c.PreAlloc).
		Int("maxBlockingTasks", c.MaxBlockingTasks)
}

// Validate rejects obviously-invalid sizing. A zero or negative Size is
// permitted (it means "unlimited" to ants), so only ExpiryDuration and
// MaxBlockingTasks have hard bounds.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.ExpiryDuration < 0 {
		errs = append(errs, fmt.Errorf("pool: expiryDuration must be non-negative, got %s", c.ExpiryDuration))
	}
	if c.MaxBlockingTasks < 0 {
		errs = append(errs, fmt.Errorf("pool: maxBlockingTasks must be non-negative, got %d", c.MaxBlockingTasks))
	}
	return errors.Join(errs...)
}

// DefaultConfig returns a Config sized to runtime.NumCPU() * 2 with a
// one-minute idle-worker reap. Callers can mutate before passing to New.
func DefaultConfig() *Config {
	return &Config{
		Size:             runtime.NumCPU() * 2,
		NonBlocking:      false,
		ExpiryDuration:   time.Minute,
		PreAlloc:         false,
		MaxBlockingTasks: 0,
	}
}
