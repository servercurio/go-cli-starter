// Package logging owns process-wide structured logging on top of zerolog.
// A single named logger (Daemon) is initialized once from a Config and then
// used package-wide; small helpers wire startup banners and a
// standard-library log adapter.
package logging
