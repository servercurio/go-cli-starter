package obfusicate

import (
	"net/url"
	"strings"
)

// obMask is the single-character glyph repeated to fill the masked portion
// of a value.
const obMask = "*"

// ConcealPrefix replaces all but the last revealChars characters of a string.
func ConcealPrefix(s string, revealChars int) string {
	if len(s) <= revealChars {
		return repeat(obMask, len(s))
	}

	return repeat(obMask, len(s)-revealChars) + s[len(s)-revealChars:]
}

// ConcealUriCredential parses s as a URI and, if it contains a userinfo
// password component, replaces the password with a fixed mask. Returns the
// input unchanged if it isn't a parseable URI or doesn't contain a password.
//
// Intended for safe logging of connection strings (database DSNs, etc.) where
// the URL host/path is operationally useful but the password must not appear.
func ConcealUriCredential(s string) string {
	if len(s) == 0 || strings.TrimSpace(s) == "" {
		return s
	}

	uri, err := url.ParseRequestURI(s)
	if err != nil {
		return s
	}

	if _, ok := uri.User.Password(); ok {
		username := uri.User.Username()
		concealedPass := repeat("+", 4)
		uri.User = url.UserPassword(username, concealedPass)
		return uri.String()
	}

	return uri.String()
}

// repeat returns s concatenated count times, or the empty string when count
// is zero or negative.
func repeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	if count == 1 {
		return s
	}

	var r string
	for i := 0; i < count; i++ {
		r += s
	}

	return r
}
