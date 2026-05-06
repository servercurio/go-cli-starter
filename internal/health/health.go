package health

import (
	"context"
	"sync"
	"time"
)

// defaultCheckTimeout is the per-check budget enforced inside Snapshot.
// Each registered CheckFunc receives a child context with this deadline;
// a check that doesn't honour the deadline will still complete in its own
// time, but a check that observes the context (e.g. forwards it to a
// PingContext call) will be cancelled and Snapshot can move on. 500ms
// leaves another 500ms under kubelet's default 1s probe timeout for
// marshalling and writing the response body.
const defaultCheckTimeout = 500 * time.Millisecond

// Status is the overall or component-level health state.
type Status string

const (
	// StatusUp means the component (or aggregated report) is healthy.
	StatusUp Status = "UP"

	// StatusDown means the component (or aggregated report) is unhealthy and
	// callers should treat it as not-ready.
	StatusDown Status = "DOWN"
)

// ComponentResult is the per-component status returned by a CheckFunc.
// Details is optional — leave it nil when there's nothing useful to surface.
type ComponentResult struct {
	Status  Status         `json:"status" yaml:"status"`
	Details map[string]any `json:"details,omitempty" yaml:"details,omitempty"`
}

// Report is the aggregated response body returned by /readyz and /healthz.
// Status is the conjunction of every component's status: UP iff all UP.
type Report struct {
	Status     Status                     `json:"status" yaml:"status"`
	Components map[string]ComponentResult `json:"components" yaml:"components"`
}

// CheckFunc is the contract a component implements to participate in health
// reports. Implementations should be cheap (the function may be called on
// every readyz request) and must honour the supplied context — Snapshot
// derives a per-check deadline from it (defaultCheckTimeout) so a hung
// dependency can't stall the readiness probe past kubelet's threshold.
// Checks that do no I/O may safely ignore the context.
type CheckFunc func(ctx context.Context) ComponentResult

// Registry is a thread-safe collection of named CheckFuncs.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

// NewRegistry returns an empty Registry. Lifecycle code should construct one
// per Application instance and pass it to consumers (router.Config, the
// v1 handlers, etc.).
func NewRegistry() *Registry {
	return &Registry{checks: map[string]CheckFunc{}}
}

// Register associates a check with a component name. A subsequent Register
// call with the same name replaces the previous check.
func (r *Registry) Register(name string, check CheckFunc) {
	if r == nil || check == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = check
}

// Unregister removes a previously-registered check. No-op if absent.
func (r *Registry) Unregister(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checks, name)
}

// Snapshot runs every registered check and aggregates the results into a
// Report. Overall Status is UP iff every component reports UP. An empty
// registry returns Status=UP with an empty components map (a server with no
// declared dependencies is, by definition, ready).
//
// Each check is invoked with a child of ctx that carries a deadline of
// defaultCheckTimeout. Checks that honour the context (e.g. by forwarding
// it to PingContext or http calls) get hard-cancelled when the budget
// expires; checks that ignore it still run synchronously to completion,
// so this is cooperative rather than preemptive. The four shipped checks
// are all either trivial struct reads or already context-aware.
func (r *Registry) Snapshot(ctx context.Context) Report {
	if r == nil {
		return Report{Status: StatusUp, Components: map[string]ComponentResult{}}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make(map[string]ComponentResult, len(r.checks))
	overall := StatusUp
	for name, check := range r.checks {
		checkCtx, cancel := context.WithTimeout(ctx, defaultCheckTimeout)
		result := check(checkCtx)
		cancel()
		components[name] = result
		if result.Status != StatusUp {
			overall = StatusDown
		}
	}
	return Report{Status: overall, Components: components}
}
