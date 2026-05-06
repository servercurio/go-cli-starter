// Package router defines the framework-agnostic abstractions used to assemble
// HTTP routes, endpoints, and modules. Concrete implementations live under
// internal/api/std/...; consumers wire them through router.Module and
// router.Route so the rest of the codebase never imports Echo types directly.
package router
