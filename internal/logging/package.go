// Package logging owns process-wide structured logging and HTTP access
// logging on top of zerolog. Two named loggers (Daemon and Access) are
// initialized once from a Config and then used package-wide; an Echo v5
// middleware emits one access-log event per request, and small helpers wire
// startup banners and standard-library log adapters.
package logging
