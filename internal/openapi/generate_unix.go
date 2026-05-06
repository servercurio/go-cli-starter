//go:build unix

package openapi

// The Server Curio brandmark lives at the repo root under docs/ so it can
// be referenced from human-facing docs (README, etc.) without pulling in a
// Go package. `go generate` copies it into this package directory at build
// time so the embed directive in embeds.go can pick it up; the copy is
// gitignored. The Windows companion is generate_windows.go.

//go:generate cp ../../docs/logo.svg logo.svg
