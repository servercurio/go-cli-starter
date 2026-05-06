package env

import "strings"

// separator is the delimiter joining a prefix to a key in environment variable
// names (e.g. "APP" + "_" + "SERVER_PORT").
const separator = "_"

// AddPrefix joins prefix and key with the package separator, upper-casing
// both, trimming whitespace, and avoiding double-prefixing when key already
// starts with the prefix. An empty prefix returns key unchanged.
func AddPrefix(prefix, key string) string {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	key = strings.ToUpper(strings.TrimSpace(key))

	if prefix == "" {
		return key
	}

	if !strings.HasSuffix(prefix, separator) {
		prefix += separator
	}

	key = strings.TrimPrefix(key, separator)

	if strings.HasPrefix(key, prefix) {
		return key
	}

	return prefix + key
}

// RemovePrefix strips prefix from the start of key (after upper-casing and
// trimming both). If key does not start with prefix, key is returned
// unchanged. An empty prefix is a no-op.
func RemovePrefix(prefix, key string) string {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	key = strings.ToUpper(strings.TrimSpace(key))

	if prefix == "" {
		return key
	}

	if strings.HasPrefix(key, prefix) {
		return key[len(prefix):]
	}

	return key
}
