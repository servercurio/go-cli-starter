package openapi

import _ "embed"

// logoSVG is the Server Curio "Project Templates" brandmark, copied here
// from docs/logo.svg by `go generate` (see generate_unix.go) and embedded
// so the Swagger UI topbar can serve it without depending on a file on
// disk at runtime.
//
//go:embed logo.svg
var logoSVG []byte
