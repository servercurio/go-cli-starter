// openapi-gen renders the project's OpenAPI spec to a file so CI can
// detect drift between the rendered spec and the version checked in at
// docs/openapi.yaml. The output is intentionally deterministic: the
// version field is pinned to a placeholder and servers list to canonical
// loopback URLs so commit-driven version metadata and per-environment
// config never appear in the rendered file.
package main
