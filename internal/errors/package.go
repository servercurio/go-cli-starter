// Package errors defines the joomcode/errorx namespaces and typed error
// values used across the project. Centralizing them here lets callers branch
// on category (filesystem, database, openapi, …) without string-matching on
// error messages, and gives downstream consumers a single import path for
// every well-known failure mode.
package errors
