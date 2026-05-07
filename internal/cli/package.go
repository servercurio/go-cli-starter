// Package cli owns the Cobra command tree and the shared
// PersistentPreRunE that loads configuration before any subcommand runs.
//
// Cobra owns argv parsing, flag handling, help text, and shell-completion
// generation. Configuration loading remains the responsibility of
// internal/application (which in turn delegates YAML/JSON parsing to
// internal/config and env-var hydration to internal/env), so the layering
// is: defaults → file → env → flags. Each subcommand's PreRunE applies the
// final flag overlay before invoking the body.
//
// To add a new subcommand: write a file under this package returning a
// *cobra.Command, then add it to the slice in NewRootCommand.
package cli
