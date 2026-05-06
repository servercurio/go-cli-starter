// Package openapi generates an OpenAPI 3.0.3 document directly from the
// router.Module / router.Route / router.Endpoint hierarchy this starter
// uses, and exposes the document at /openapi.yaml and /openapi.json. It can
// optionally serve a Swagger UI on top of that spec via Swaggo's
// echo-swagger v2 (see swagger.go).
//
// Request and response schemas are derived from the Go types declared on
// each endpoint via endpoint.WithRequest / endpoint.WithResponse — see
// schema.go for the reflection-based type-to-Schema conversion. Named
// struct types are emitted under Components.Schemas and referenced via
// $ref so each type appears once regardless of how many operations use it.
package openapi
