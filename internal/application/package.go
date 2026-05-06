// Package application owns the daemon lifecycle: loading and validating
// configuration, building the global middleware stack, configuring the HTTP
// and TLS servers (with optional ACME/auto-issued certs), wiring the
// database/ORM, OpenAPI/Swagger UI, health registry, and registered routing
// modules, and shutting everything down on signal. cmd/daemon constructs an
// Application via NewApplication, registers domain modules, then runs the
// Configure → Initialize → Start sequence.
package application
