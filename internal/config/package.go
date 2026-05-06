// Package config locates and parses YAML/JSON configuration files into Go
// structs. It is layered under the application: callers (e.g. cmd/daemon)
// supply a list of search paths and a base name, and FromPaths walks them
// looking for matching files. The package handles file discovery, format
// detection by extension, and the unmarshal step; it does not know about
// any specific config schema — that's the caller's responsibility.
package config
