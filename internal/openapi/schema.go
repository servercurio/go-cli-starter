package openapi

import (
	"reflect"
	"strings"
	"time"
)

// SchemaRegistry converts Go types to OpenAPI Schema objects, deduplicating
// named struct types behind $ref so each shape appears once in the
// generated spec regardless of how many operations reference it.
//
// Concurrency: not safe for parallel use. The Build call is single-threaded
// today; if that ever changes, wrap the schemas map in a sync.Mutex.
type SchemaRegistry struct {
	// schemas is the by-name set returned via Schemas() and embedded in
	// Components.Schemas. Keyed by the leading segment of the $ref —
	// typically the bare Go type name (e.g. "Report") with collisions
	// disambiguated by package qualifier ("health.Report") if needed.
	schemas map[string]*Schema

	// inProgress holds the names of types we're currently building schemas
	// for, so recursive types (struct A → []A) emit a $ref to themselves
	// instead of recursing forever.
	inProgress map[reflect.Type]string

	// names maps each reflect.Type we've named to the schema-registry name
	// we assigned it. Used both to issue stable $refs across calls and to
	// detect when a previously-seen type comes around again.
	names map[reflect.Type]string
}

// NewSchemaRegistry returns a fresh, empty registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas:    map[string]*Schema{},
		inProgress: map[reflect.Type]string{},
		names:      map[reflect.Type]string{},
	}
}

// Schemas returns the by-name map embedded under Components.Schemas. The
// caller should treat the result as read-only.
func (r *SchemaRegistry) Schemas() map[string]*Schema {
	return r.schemas
}

// SchemaFor returns the OpenAPI Schema for the given Go type. Named struct
// types are registered under Components.Schemas and a $ref is returned;
// primitives, slices, maps, and unnamed struct types are inlined.
func (r *SchemaRegistry) SchemaFor(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{}
	}
	return r.schemaFor(t)
}

// schemaFor is the recursive worker behind SchemaFor. Handles pointer,
// time.Time, primitive, container, and struct cases; structs delegate to
// structSchema for the named-vs-anonymous split.
func (r *SchemaRegistry) schemaFor(t reflect.Type) *Schema {
	// Pointer wrapper: emit the pointee's schema with nullable=true.
	// Repeated wraps collapse — reflect handles that for us.
	if t.Kind() == reflect.Pointer {
		s := r.schemaFor(t.Elem())
		// $ref schemas can't carry siblings in OpenAPI 3.0, so don't
		// stamp nullable on a ref. The pointer-ness is documented at
		// the field-level via the parent struct's `required` list
		// instead (omitted required ⇒ optional/nullable).
		if s.Ref == "" {
			s.Nullable = true
		}
		return s
	}

	// time.Time is a named struct but its useful schema is a string with
	// the date-time format, not the {wall, ext, loc} field set. Special
	// case it before the named-struct path.
	if t == reflect.TypeOf(time.Time{}) {
		return &Schema{Type: "string", Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		// []byte is conventionally rendered as a base64-encoded string in
		// OpenAPI; matches how encoding/json marshals byte slices.
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: r.schemaFor(t.Elem())}
	case reflect.Map:
		// Only map[string]T has a sensible OpenAPI representation.
		// Non-string keys collapse to a generic object.
		if t.Key().Kind() != reflect.String {
			return &Schema{Type: "object"}
		}
		return &Schema{
			Type:                 "object",
			AdditionalProperties: r.schemaFor(t.Elem()),
		}
	case reflect.Interface:
		// `any` / `interface{}` becomes a free-form object — no `type`
		// constraint, so callers can put anything there.
		return &Schema{}
	case reflect.Struct:
		return r.structSchema(t)
	default:
		// chan, func, complex, unsafe.Pointer — not representable in
		// JSON schemas. Emit a placeholder so the spec stays valid.
		return &Schema{Type: "object"}
	}
}

// structSchema either returns a $ref to an already-registered named
// struct, or registers a new one and returns the $ref. Anonymous structs
// are inlined since they have no name to register under.
func (r *SchemaRegistry) structSchema(t reflect.Type) *Schema {
	name := schemaName(t)
	if name == "" {
		// Anonymous struct — inline it.
		return r.inlineStructSchema(t)
	}

	// Already fully built? Return the $ref directly.
	if existing, ok := r.names[t]; ok {
		return &Schema{Ref: refPath(existing)}
	}

	// Currently being built (recursive cycle)? Return the $ref to the
	// in-progress name; the caller's schema will be filled in once we
	// unwind.
	if existing, ok := r.inProgress[t]; ok {
		return &Schema{Ref: refPath(existing)}
	}

	r.inProgress[t] = name
	body := r.inlineStructSchema(t)
	delete(r.inProgress, t)

	r.schemas[name] = body
	r.names[t] = name

	return &Schema{Ref: refPath(name)}
}

// inlineStructSchema walks a struct's fields and produces an `object` schema
// inline. Honours JSON struct tags (rename, omit, omitempty) the way
// encoding/json does.
func (r *SchemaRegistry) inlineStructSchema(t reflect.Type) *Schema {
	props := map[string]*Schema{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Embedded struct fields flatten regardless of whether the
		// field name itself is exported — encoding/json lifts the
		// embedded type's own (exported) fields into the parent. Check
		// this BEFORE the IsExported gate so an embedded struct with a
		// lowercase reflect.StructField.Name still gets walked.
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			embedded := r.inlineStructSchema(f.Type)
			for k, v := range embedded.Properties {
				props[k] = v
			}
			required = append(required, embedded.Required...)
			continue
		}

		if !f.IsExported() {
			continue
		}

		name, optional, skip := jsonFieldName(f)
		if skip {
			continue
		}

		props[name] = r.schemaFor(f.Type)

		// A field is required iff its JSON tag does not say omitempty
		// AND the underlying type is not a pointer (pointers are
		// nullable so we treat them as optional). This is a pragmatic
		// approximation of "what the consumer must always send".
		if !optional && f.Type.Kind() != reflect.Pointer {
			required = append(required, name)
		}
	}

	return &Schema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

// jsonFieldName extracts the JSON serialization name for a struct field
// from its `json:"…"` tag, applying encoding/json's rules:
//
//   - tag of "-" → field is skipped.
//   - empty tag (or no tag) → use the field name.
//   - "name,omitempty" → use "name" and mark optional.
//   - ",omitempty" → use the field name and mark optional.
func jsonFieldName(f reflect.StructField) (name string, optional bool, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}

	if tag == "" {
		return f.Name, false, false
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = f.Name
	}
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			optional = true
		}
	}
	return name, optional, false
}

// schemaName returns a stable, registry-unique name for a named struct
// type. Anonymous structs return "" so callers know to inline them.
//
// The name format is `<PackageName><TypeName>` when the type is named, e.g.
// `health.Report` becomes `healthReport`. Bare type name without any
// package qualifier risks collisions across packages but reads cleanly in
// $ref output; the package-prefixed form prevents that. Tweak here if a
// future schema collision shows up.
func schemaName(t reflect.Type) string {
	if t.Name() == "" {
		return ""
	}

	pkgPath := t.PkgPath()
	if pkgPath == "" {
		// Built-in like `error` — treat as anonymous.
		return ""
	}

	// Use just the trailing path segment as the package name.
	pkg := pkgPath
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
		pkg = pkg[idx+1:]
	}

	return pkg + "." + t.Name()
}

// refPath returns the JSON Pointer used in $ref values.
func refPath(name string) string {
	return "#/components/schemas/" + name
}
