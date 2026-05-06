// Package health implements an in-process health-check registry and the
// response model returned by the /api/v1/livez, /readyz, and /healthz
// endpoints.
//
// The model intentionally mirrors what Spring Boot Actuator, Quarkus
// SmallRye Health, and Micronaut return: an overall status plus a map of
// per-component statuses with optional details. Components register
// themselves with a Registry; the v1 handlers ask the registry for a
// Snapshot per request.
//
// The Registry is an explicit dependency (passed via router.Config) rather
// than a package-level singleton, in keeping with the no-global-state
// convention documented in CLAUDE.md.
package health
