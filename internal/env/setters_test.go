package env

import (
	"testing"
	"time"

	asrt "github.com/stretchr/testify/assert"
)

func TestSetStringValue(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_FOO", "hello")
	v := "default"
	assert.True(SetStringValue("APP", "foo", &v))
	assert.Equal("hello", v)

	v2 := "kept"
	assert.False(SetStringValue("APP", "missing_key", &v2))
	assert.Equal("kept", v2, "unset env var must leave value untouched")
}

func TestSetBoolValue(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_FLAG", "true")
	b := false
	assert.True(SetBoolValue("APP", "flag", &b))
	assert.True(b)

	t.Setenv("APP_BADFLAG", "notabool")
	b2 := true
	assert.False(SetBoolValue("APP", "badflag", &b2),
		"unparseable value must return false and leave receiver alone")
	assert.True(b2)
}

func TestSetIntValue(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_N", "42")
	n := 0
	assert.True(SetIntValue("APP", "n", &n))
	assert.Equal(42, n)

	t.Setenv("APP_BADN", "x")
	n2 := 7
	assert.False(SetIntValue("APP", "badn", &n2))
	assert.Equal(7, n2)
}

func TestSetUint16Value(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_PORT", "8080")
	var p uint16
	assert.True(SetUint16Value("APP", "port", &p))
	assert.Equal(uint16(8080), p)

	t.Setenv("APP_BIGPORT", "70000")
	var p2 uint16 = 1
	assert.False(SetUint16Value("APP", "bigport", &p2),
		"value out of uint16 range must be rejected")
	assert.Equal(uint16(1), p2)
}

func TestSetFloatValue(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_RATE", "3.14")
	var f float64
	assert.True(SetFloatValue("APP", "rate", &f))
	assert.InDelta(3.14, f, 0.0001)

	t.Setenv("APP_BADRATE", "x")
	f2 := 1.0
	assert.False(SetFloatValue("APP", "badrate", &f2))
	assert.Equal(1.0, f2)
}

func TestSetDurationValue(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_TIMEOUT", "30s")
	var d time.Duration
	assert.True(SetDurationValue("APP", "timeout", &d))
	assert.Equal(30*time.Second, d)

	t.Setenv("APP_BADTIMEOUT", "x")
	d2 := time.Minute
	assert.False(SetDurationValue("APP", "badtimeout", &d2))
	assert.Equal(time.Minute, d2)
}
