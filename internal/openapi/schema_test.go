package openapi_test

import (
	"reflect"
	"testing"
	"time"

	asrt "github.com/stretchr/testify/assert"

	"github.com/servercurio/go-cli-starter/internal/openapi"
)

func TestSchemaRegistry_Primitives(t *testing.T) {
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	cases := []struct {
		val          any
		wantType     string
		wantFormat   string
		wantNullable bool
	}{
		{"x", "string", "", false},
		{true, "boolean", "", false},
		{int(1), "integer", "int32", false},
		{int64(1), "integer", "int64", false},
		{uint(1), "integer", "int32", false},
		{uint64(1), "integer", "int64", false},
		{float32(1), "number", "float", false},
		{float64(1), "number", "double", false},
	}

	for _, c := range cases {
		s := r.SchemaFor(reflect.TypeOf(c.val))
		assert.Equal(c.wantType, s.Type, "value=%v", c.val)
		assert.Equal(c.wantFormat, s.Format, "value=%v", c.val)
		assert.Empty(s.Ref)
	}

	assert.Empty(r.Schemas(), "primitives are inlined, never registered")
}

func TestSchemaRegistry_TimeBecomesStringDateTime(t *testing.T) {
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	s := r.SchemaFor(reflect.TypeOf(time.Time{}))

	assert.Equal("string", s.Type)
	assert.Equal("date-time", s.Format)
	assert.Empty(r.Schemas(), "time.Time is special-cased and not registered as a struct schema")
}

func TestSchemaRegistry_SliceMapByteSlice(t *testing.T) {
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	intSlice := r.SchemaFor(reflect.TypeOf([]int{}))
	assert.Equal("array", intSlice.Type)
	assert.Equal("integer", intSlice.Items.Type)

	bytes := r.SchemaFor(reflect.TypeOf([]byte{}))
	assert.Equal("string", bytes.Type, "[]byte → base64 string per OpenAPI convention")
	assert.Equal("byte", bytes.Format)

	stringMap := r.SchemaFor(reflect.TypeOf(map[string]int{}))
	assert.Equal("object", stringMap.Type)
	assert.Equal("integer", stringMap.AdditionalProperties.Type)
}

type fixtureUser struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Email     string       `json:"email,omitempty"`
	Tags      []string     `json:"tags"`
	CreatedAt time.Time    `json:"createdAt"`
	Friend    *fixtureUser `json:"friend,omitempty"`
	internal  string       //nolint:unused // unexported, must be skipped by the generator
	Skipped   string       `json:"-"`
}

func TestSchemaRegistry_StructHonorsJSONTags(t *testing.T) {
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	ref := r.SchemaFor(reflect.TypeOf(fixtureUser{}))

	// Top-level must be a $ref, with the actual schema registered.
	assert.NotEmpty(ref.Ref)
	assert.Equal("#/components/schemas/openapi_test.fixtureUser", ref.Ref)

	schemas := r.Schemas()
	body, ok := schemas["openapi_test.fixtureUser"]
	assert.True(ok)

	// Property names follow JSON tags, not Go field names.
	assert.Contains(body.Properties, "id")
	assert.Contains(body.Properties, "name")
	assert.Contains(body.Properties, "email")
	assert.Contains(body.Properties, "tags")
	assert.Contains(body.Properties, "createdAt")
	assert.Contains(body.Properties, "friend")
	assert.NotContains(body.Properties, "internal", "unexported fields must be skipped")
	assert.NotContains(body.Properties, "Skipped", "json:\"-\" fields must be skipped")

	// `omitempty` and pointer fields are not required.
	assert.Contains(body.Required, "id")
	assert.Contains(body.Required, "name")
	assert.Contains(body.Required, "tags")
	assert.Contains(body.Required, "createdAt")
	assert.NotContains(body.Required, "email", "omitempty → optional")
	assert.NotContains(body.Required, "friend", "pointer → optional")

	// time.Time stays inlined as string+date-time even inside a struct.
	createdAt := body.Properties["createdAt"]
	assert.Equal("string", createdAt.Type)
	assert.Equal("date-time", createdAt.Format)
}

func TestSchemaRegistry_RecursiveStructEmitsSelfRef(t *testing.T) {
	// fixtureUser.Friend is *fixtureUser — make sure we don't infinitely
	// recurse and instead emit a $ref back to the same schema.
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	r.SchemaFor(reflect.TypeOf(fixtureUser{}))

	body := r.Schemas()["openapi_test.fixtureUser"]
	friend := body.Properties["friend"]
	assert.Equal("#/components/schemas/openapi_test.fixtureUser", friend.Ref,
		"recursive struct field must $ref back to itself, not recurse forever")
}

func TestSchemaRegistry_DeduplicatesNamedTypes(t *testing.T) {
	// Calling SchemaFor twice for the same named type must not register
	// two copies — the second call should return the existing $ref.
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	r.SchemaFor(reflect.TypeOf(fixtureUser{}))
	r.SchemaFor(reflect.TypeOf(fixtureUser{}))

	assert.Len(r.Schemas(), 1, "named type must only be registered once")
}

type fixtureWithEmbed struct {
	fixtureUser
	Extra string `json:"extra"`
}

func TestSchemaRegistry_EmbeddedStructFieldsLifted(t *testing.T) {
	// encoding/json flattens fields from embedded structs into the parent
	// — the schema should mirror that behaviour so consumers see the same
	// shape they'd unmarshal from JSON.
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	r.SchemaFor(reflect.TypeOf(fixtureWithEmbed{}))

	body := r.Schemas()["openapi_test.fixtureWithEmbed"]
	assert.Contains(body.Properties, "id", "embedded field must be lifted")
	assert.Contains(body.Properties, "name", "embedded field must be lifted")
	assert.Contains(body.Properties, "extra", "outer-level field present")
}

func TestSchemaRegistry_AnyBecomesFreeFormObject(t *testing.T) {
	// `any` / interface{} fields must serialise as a free-form object —
	// no `type` constraint — so consumers can put anything there.
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	type holder struct {
		Payload any `json:"payload"`
	}

	r.SchemaFor(reflect.TypeOf(holder{}))
	body := r.Schemas()["openapi_test.holder"]
	payload := body.Properties["payload"]
	assert.Empty(payload.Type, "any → no type constraint")
	assert.Empty(payload.Ref)
}

func TestSchemaRegistry_NilTypeReturnsEmptySchema(t *testing.T) {
	// Defensive: callers shouldn't pass nil but if they do we'd rather
	// emit a placeholder than panic.
	assert := asrt.New(t)
	r := openapi.NewSchemaRegistry()

	s := r.SchemaFor(nil)
	assert.NotNil(s)
	assert.Empty(s.Type)
	assert.Empty(s.Ref)
}
