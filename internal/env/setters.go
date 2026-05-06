package env

import (
	"os"
	"strconv"
	"time"
)

// SetStringValue assigns the value of the prefix+key environment variable to
// *value when set. Returns true on assignment, false when the variable is
// unset; *value is left untouched in the latter case.
func SetStringValue(prefix, key string, value *string) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		*value = v
		return true
	}
	return false
}

// SetBoolValue parses the prefix+key environment variable as a bool (using
// strconv.ParseBool semantics) and assigns it to *value. Returns false when
// the variable is unset or fails to parse; *value is left untouched in those
// cases.
func SetBoolValue(prefix, key string, value *bool) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			*value = b
			return true
		}
	}
	return false
}

// SetIntValue parses the prefix+key environment variable as a base-10 int and
// assigns it to *value. Returns false when unset or unparseable; *value is
// left untouched in those cases.
func SetIntValue(prefix, key string, value *int) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		if i, err := strconv.Atoi(v); err == nil {
			*value = i
			return true
		}
	}
	return false
}

// SetUint16Value parses the prefix+key environment variable as an unsigned
// 16-bit integer (typical for TCP/UDP ports) and assigns it to *value.
// Returns false when unset or unparseable; *value is left untouched in those
// cases.
func SetUint16Value(prefix, key string, value *uint16) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		if i, err := strconv.ParseUint(v, 10, 16); err == nil {
			*value = uint16(i)
			return true
		}
	}
	return false
}

// SetFloatValue parses the prefix+key environment variable as a 64-bit float
// and assigns it to *value. Returns false when unset or unparseable; *value
// is left untouched in those cases.
func SetFloatValue(prefix, key string, value *float64) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			*value = f
			return true
		}
	}
	return false
}

// SetDurationValue parses the prefix+key environment variable using
// time.ParseDuration (e.g. "30s", "5m", "1h") and assigns it to *value.
// Returns false when unset or unparseable; *value is left untouched in those
// cases.
func SetDurationValue(prefix, key string, value *time.Duration) bool {
	if v, ok := os.LookupEnv(AddPrefix(prefix, key)); ok {
		if d, err := time.ParseDuration(v); err == nil {
			*value = d
			return true
		}
	}
	return false
}
