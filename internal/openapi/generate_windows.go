//go:build windows

package openapi

// Windows companion to generate_unix.go. `copy` is a cmd builtin (not a
// standalone executable), so it has to be invoked through `cmd /c`.
// Backslash path separators avoid surprises with older `copy` versions
// that don't normalise forward slashes.

//go:generate cmd /c copy /Y ..\..\docs\logo.svg logo.svg
