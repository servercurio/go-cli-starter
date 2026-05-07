// Package health implements an in-process health-check registry and the
// report model snapshotted by callers (e.g. a "health" CLI subcommand or a
// daemon's readiness loop).
//
// The model intentionally mirrors what Spring Boot Actuator, Quarkus
// SmallRye Health, and Micronaut return: an overall status plus a map of
// per-component statuses with optional details. Components register
// themselves with a Registry; consumers ask the registry for a Snapshot.
//
// The Registry is an explicit dependency (exposed by Application via
// app.HealthRegistry()) rather than a package-level singleton, in keeping
// with the no-global-state convention documented in CLAUDE.md.
package health
