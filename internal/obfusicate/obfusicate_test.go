package obfusicate

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestConcealUriCredential_HidesPasswordWhenPresent(t *testing.T) {
	assert := asrt.New(t)

	got := ConcealUriCredential("postgres://user:hunter2@db.example.com:5432/app?sslmode=disable")

	assert.NotContains(got, "hunter2", "raw password must not appear in the masked URI")
	assert.Contains(got, "user", "username should be preserved for diagnostics")
	assert.Contains(got, "db.example.com", "host should be preserved for diagnostics")
	assert.Contains(got, "5432", "port should be preserved for diagnostics")
}

func TestConcealUriCredential_LeavesPasswordlessUriUnchanged(t *testing.T) {
	assert := asrt.New(t)

	in := "postgres://user@db.example.com:5432/app?sslmode=disable"

	assert.Equal(in, ConcealUriCredential(in))
}

func TestConcealUriCredential_PassesThroughEmpty(t *testing.T) {
	assert := asrt.New(t)
	assert.Equal("", ConcealUriCredential(""))
	assert.Equal("   ", ConcealUriCredential("   "))
}

func TestConcealUriCredential_PassesThroughUnparseable(t *testing.T) {
	assert := asrt.New(t)
	in := "not a uri at all"
	assert.Equal(in, ConcealUriCredential(in))
}
