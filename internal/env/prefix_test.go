package env

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestAddPrefix(t *testing.T) {
	assert := asrt.New(t)

	assert.Equal("APP_DAEMON_LOG_LEVEL", AddPrefix("APP", "daemon_log_level"))
	assert.Equal("APP_DAEMON_LOG_LEVEL", AddPrefix("app", "daemon_log_level"),
		"prefix and key should be upper-cased")
	assert.Equal("APP_DAEMON_LOG_LEVEL", AddPrefix("  APP  ", "  daemon_log_level  "),
		"whitespace should be trimmed")
	assert.Equal("DAEMON_LOG_LEVEL", AddPrefix("", "daemon_log_level"),
		"empty prefix returns key unchanged (after upper-casing)")
	assert.Equal("APP_FOO", AddPrefix("APP_", "foo"),
		"prefix with trailing separator is not double-joined")
	assert.Equal("APP_DAEMON_LOG_LEVEL", AddPrefix("APP", "_daemon_log_level"),
		"leading separator on key should be trimmed before joining")
	assert.Equal("APP_DAEMON_LOG_LEVEL", AddPrefix("APP", "APP_DAEMON_LOG_LEVEL"),
		"key already prefixed should not be double-prefixed")
}

func TestRemovePrefix(t *testing.T) {
	assert := asrt.New(t)

	// RemovePrefix strips exactly the prefix string; the leading separator
	// remains because it isn't part of the prefix as supplied.
	assert.Equal("_DAEMON_LOG_LEVEL", RemovePrefix("APP", "APP_DAEMON_LOG_LEVEL"))
	assert.Equal("_DAEMON_LOG_LEVEL", RemovePrefix("app", "app_daemon_log_level"),
		"both args upper-cased before comparing")
	assert.Equal("APP_DAEMON_LOG_LEVEL", RemovePrefix("", "APP_DAEMON_LOG_LEVEL"),
		"empty prefix returns key unchanged")
	assert.Equal("OTHER_KEY", RemovePrefix("APP", "OTHER_KEY"),
		"key without prefix returned unchanged")
}
