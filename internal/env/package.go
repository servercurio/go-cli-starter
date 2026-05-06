// Package env provides small helpers for reading environment variables into
// typed config fields. The conventions are: prefixes are upper-cased and
// joined to keys with an underscore (see AddPrefix), and SetXxxValue helpers
// only mutate their target when the variable is set AND parses cleanly —
// callers can chain them and rely on existing field defaults remaining intact
// for unset or malformed values.
package env
