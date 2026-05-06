package health

import (
	"context"
	"testing"
	"time"

	asrt "github.com/stretchr/testify/assert"
)

// upCheck and downCheck are tiny helpers for the registry-shape tests.
// They ignore the context — the timeout-budget enforcement is exercised
// separately in TestRegistry_SnapshotEnforcesPerCheckBudget.
func upCheck(_ context.Context) ComponentResult   { return ComponentResult{Status: StatusUp} }
func downCheck(_ context.Context) ComponentResult { return ComponentResult{Status: StatusDown} }

func TestRegistry_EmptyRegistryIsUp(t *testing.T) {
	// "No declared dependencies" is, by definition, healthy.
	assert := asrt.New(t)
	r := NewRegistry()

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusUp, rep.Status)
	assert.Empty(rep.Components)
}

func TestRegistry_AllUpAggregatesUp(t *testing.T) {
	assert := asrt.New(t)
	r := NewRegistry()
	r.Register("a", upCheck)
	r.Register("b", upCheck)

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusUp, rep.Status)
	assert.Len(rep.Components, 2)
}

func TestRegistry_AnyDownAggregatesDown(t *testing.T) {
	// Pin the conjunction semantics: a single DOWN component flips the
	// overall report. Mirrors how Spring Boot Actuator and Quarkus health
	// aggregate component status.
	assert := asrt.New(t)
	r := NewRegistry()
	r.Register("a", upCheck)
	r.Register("b", downCheck)

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusDown, rep.Status)
	assert.Equal(StatusUp, rep.Components["a"].Status)
	assert.Equal(StatusDown, rep.Components["b"].Status)
}

func TestRegistry_RegisterReplacesExisting(t *testing.T) {
	assert := asrt.New(t)
	r := NewRegistry()
	r.Register("x", upCheck)
	r.Register("x", downCheck)

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusDown, rep.Status)
}

func TestRegistry_UnregisterRemovesCheck(t *testing.T) {
	assert := asrt.New(t)
	r := NewRegistry()
	r.Register("flapper", downCheck)
	r.Unregister("flapper")

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusUp, rep.Status, "removing the only DOWN component should restore overall UP")
	assert.NotContains(rep.Components, "flapper")
}

func TestRegistry_NilReceiverIsSafe(t *testing.T) {
	// Defensive: the v1 handler holds the registry by pointer; if a future
	// caller hands us a nil pointer we'd rather report "ready, no
	// components" than panic. Pin that here.
	assert := asrt.New(t)
	var r *Registry

	rep := r.Snapshot(context.Background())
	assert.Equal(StatusUp, rep.Status)
	assert.Empty(rep.Components)

	// Register/Unregister on nil receiver are no-ops, not panics.
	assert.NotPanics(func() {
		r.Register("never", downCheck)
		r.Unregister("never")
	})
}

// TestRegistry_SnapshotEnforcesPerCheckBudget pins the latency-budget
// contract: each registered check receives a ctx with a deadline of
// defaultCheckTimeout. A check that observes the ctx and surfaces a
// timeout-aware result lets Snapshot return a DOWN report instead of
// stalling /readyz past kubelet's probe deadline.
//
// We use a check that blocks on <-ctx.Done() so the test is fast and
// deterministic — no real sleep — and confirms two things: (1) Snapshot
// returns *before* the upstream context's deadline expires, and (2) the
// component result reflects the timeout.
func TestRegistry_SnapshotEnforcesPerCheckBudget(t *testing.T) {
	assert := asrt.New(t)
	r := NewRegistry()
	r.Register("hung", func(ctx context.Context) ComponentResult {
		<-ctx.Done()
		return ComponentResult{
			Status:  StatusDown,
			Details: map[string]any{"reason": ctx.Err().Error()},
		}
	})
	r.Register("fast", upCheck)

	// Parent ctx has a much longer deadline than the per-check budget so
	// the test is sensitive to the per-check enforcement, not the parent.
	parent, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	rep := r.Snapshot(parent)
	elapsed := time.Since(start)

	assert.Equal(StatusDown, rep.Status, "any DOWN component flips overall")
	assert.Equal(StatusDown, rep.Components["hung"].Status)
	assert.Equal(StatusUp, rep.Components["fast"].Status)
	assert.Contains(rep.Components["hung"].Details["reason"], "deadline exceeded",
		"hung check should surface the ctx deadline error")
	assert.Less(elapsed, 2*defaultCheckTimeout,
		"Snapshot should return within ~one per-check budget, not stall on the hung check")
}

func TestFormatFromAccept(t *testing.T) {
	assert := asrt.New(t)

	cases := []struct {
		accept string
		want   Format
	}{
		{"", FormatJSON},
		{"*/*", FormatJSON},
		{"application/json", FormatJSON},
		{"application/json, text/plain", FormatJSON},
		{"application/yaml", FormatYAML},
		{"application/x-yaml", FormatYAML},
		{"text/yaml", FormatYAML},
		{"application/health+yaml", FormatYAML},
		{"APPLICATION/YAML", FormatYAML}, // case-insensitive
	}

	for _, c := range cases {
		assert.Equal(c.want, FormatFromAccept(c.accept), "Accept=%q", c.accept)
	}
}
