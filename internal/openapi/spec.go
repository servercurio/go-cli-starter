package openapi

import (
	"strconv"
	"strings"

	"github.com/servercurio/go-cli-starter/internal/router"
)

// Spec is the top-level OpenAPI 3.0.3 document.
type Spec struct {
	OpenAPI    string              `yaml:"openapi" json:"openapi"`
	Info       Info                `yaml:"info" json:"info"`
	Servers    []Server            `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths      map[string]PathItem `yaml:"paths" json:"paths"`
	Tags       []Tag               `yaml:"tags,omitempty" json:"tags,omitempty"`
	Components *Components         `yaml:"components,omitempty" json:"components,omitempty"`
}

// Info populates the OpenAPI `info` block.
type Info struct {
	Title       string `yaml:"title" json:"title"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Server is one entry in the OpenAPI `servers` array.
type Server struct {
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Tag groups operations in the spec; Swagger UI shows one section per tag.
type Tag struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PathItem holds the per-method operations for one URL template. Unset
// methods stay nil and are omitted from the marshaled output.
type PathItem struct {
	Get     *Operation `yaml:"get,omitempty" json:"get,omitempty"`
	Post    *Operation `yaml:"post,omitempty" json:"post,omitempty"`
	Put     *Operation `yaml:"put,omitempty" json:"put,omitempty"`
	Delete  *Operation `yaml:"delete,omitempty" json:"delete,omitempty"`
	Patch   *Operation `yaml:"patch,omitempty" json:"patch,omitempty"`
	Options *Operation `yaml:"options,omitempty" json:"options,omitempty"`
	Head    *Operation `yaml:"head,omitempty" json:"head,omitempty"`
}

// Operation is a single endpoint description: one method on one path.
type Operation struct {
	OperationID string              `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Summary     string              `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string            `yaml:"tags,omitempty" json:"tags,omitempty"`
	Parameters  []Parameter         `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody *RequestBody        `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses" json:"responses"`
}

// Parameter is currently used only for path parameters (in: "path"). Extend
// with `in: "query"` / `in: "header"` once the route abstraction surfaces
// query/header metadata.
type Parameter struct {
	Name     string  `yaml:"name" json:"name"`
	In       string  `yaml:"in" json:"in"`
	Required bool    `yaml:"required" json:"required"`
	Schema   *Schema `yaml:"schema" json:"schema"`
}

// RequestBody is the OpenAPI `requestBody` object. Required defaults to
// true when the endpoint declares a request type — there's no notion of an
// optional body in the current builder API; revisit if that becomes a need.
type RequestBody struct {
	Description string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool                    `yaml:"required,omitempty" json:"required,omitempty"`
	Content     map[string]MediaTypeObj `yaml:"content" json:"content"`
}

// Response is the per-status-code response descriptor.
type Response struct {
	Description string                  `yaml:"description" json:"description"`
	Content     map[string]MediaTypeObj `yaml:"content,omitempty" json:"content,omitempty"`
}

// MediaTypeObj is the OpenAPI Media Type Object — content under a specific
// media type within a request or response. Examples could go here too;
// keeping the surface lean for now.
type MediaTypeObj struct {
	Schema *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// Schema is an OpenAPI Schema Object. Either Ref is set (referencing a
// component) OR the inline fields are populated — never both. The
// reflection-based generator in schema.go always produces $ref for named
// struct types so each shape appears once under Components.Schemas.
type Schema struct {
	Ref string `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	Type   string `yaml:"type,omitempty" json:"type,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`

	// Composition / containers.
	Items                *Schema            `yaml:"items,omitempty" json:"items,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
	Required             []string           `yaml:"required,omitempty" json:"required,omitempty"`

	// Modifiers.
	Nullable    bool          `yaml:"nullable,omitempty" json:"nullable,omitempty"`
	Description string        `yaml:"description,omitempty" json:"description,omitempty"`
	Enum        []interface{} `yaml:"enum,omitempty" json:"enum,omitempty"`
}

// Components is the OpenAPI `components` block. Schemas is keyed by the
// component name (matching the trailing segment of $ref values).
type Components struct {
	Schemas map[string]*Schema `yaml:"schemas,omitempty" json:"schemas,omitempty"`
}

// Build walks every Module → Route → Endpoint reachable from the supplied
// modules and assembles a Spec. Endpoint metadata (Summary/Description/
// Request/Responses) is read via the router.Endpoint accessors and
// translated to OpenAPI Operation / RequestBody / Response objects, with
// schemas registered against a shared SchemaRegistry so each named type
// appears exactly once under Components.Schemas regardless of how many
// operations reference it.
//
// The result is deterministic for the same input: yaml.v3 and encoding/json
// both sort map keys at marshal time, sortTags below is a stable insertion
// sort, and SchemaRegistry preserves insertion order independent of map
// iteration since the schema map is itself sorted at marshal time.
func Build(info Info, servers []Server, modules []router.Module) *Spec {
	registry := NewSchemaRegistry()
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info:    info,
		Servers: servers,
		Paths:   map[string]PathItem{},
	}

	tagSet := map[string]struct{}{}
	walk(modules, "", spec.Paths, tagSet, registry)

	for name := range tagSet {
		spec.Tags = append(spec.Tags, Tag{Name: name})
	}
	sortTags(spec.Tags)

	if schemas := registry.Schemas(); len(schemas) > 0 {
		spec.Components = &Components{Schemas: schemas}
	}

	return spec
}

// walk is the recursive driver behind Build. It accumulates paths into the
// supplied map, tag names into the supplied set, and named schemas into the
// supplied registry; all are mutated in place.
func walk(mods []router.Module, parentPrefix string, paths map[string]PathItem, tags map[string]struct{}, reg *SchemaRegistry) {
	for _, m := range mods {
		modulePrefix := joinPath(parentPrefix, m.Prefix())
		moduleTag := m.Name()

		// Only mark a tag as present if the module actually exposes
		// reachable operations (directly or via a sub-module). For the
		// current usage every registered module has at least one route or
		// sub-module so the simpler approach is fine.
		tags[moduleTag] = struct{}{}

		for _, r := range m.Routes() {
			fullPath := joinPath(modulePrefix, r.Path())
			specPath, params := convertPath(fullPath)

			for _, ep := range r.Endpoints() {
				for _, method := range ep.Methods() {
					op := buildOperation(r.Name(), moduleTag, params, ep, reg)
					item := paths[specPath]
					setMethod(&item, method, op)
					paths[specPath] = item
				}
			}
		}

		walk(m.SubModules(), modulePrefix, paths, tags, reg)
	}
}

// buildOperation translates one (route, endpoint) pair into an OpenAPI
// Operation object. Pulls Summary / Description / RequestBody / Responses
// from the endpoint accessors; falls back to the route's Name as the
// operation summary when the endpoint doesn't supply one. If the endpoint
// declares no responses at all, a default 200 response is emitted so the
// spec is still valid (responses is a required field on Operation).
func buildOperation(routeName, tag string, params []Parameter, ep router.Endpoint, reg *SchemaRegistry) *Operation {
	summary := ep.Summary()
	if summary == "" {
		summary = routeName
	}

	op := &Operation{
		OperationID: ep.Id(),
		Summary:     summary,
		Description: ep.Description(),
		Tags:        []string{tag},
		Parameters:  params,
		Responses:   map[string]Response{},
	}

	if req := ep.Request(); req != nil && req.Type != nil {
		ct := req.ContentType
		if ct == "" {
			ct = "application/json"
		}
		op.RequestBody = &RequestBody{
			Description: req.Description,
			Required:    true,
			Content: map[string]MediaTypeObj{
				ct: {Schema: reg.SchemaFor(req.Type)},
			},
		}
	}

	for code, resp := range ep.Responses() {
		key := strconv.Itoa(code)
		out := Response{Description: resp.Description}
		if out.Description == "" {
			out.Description = "Response"
		}
		if resp.Type != nil {
			ct := resp.ContentType
			if ct == "" {
				ct = "application/json"
			}
			out.Content = map[string]MediaTypeObj{
				ct: {Schema: reg.SchemaFor(resp.Type)},
			}
		}
		op.Responses[key] = out
	}

	if len(op.Responses) == 0 {
		op.Responses["200"] = Response{Description: "Successful response"}
	}

	return op
}

// joinPath stitches a parent prefix to a child segment, ensuring exactly
// one leading slash and no doubled separators.
func joinPath(parent, child string) string {
	parent = strings.Trim(parent, "/")
	child = strings.Trim(child, "/")
	switch {
	case parent == "" && child == "":
		return "/"
	case parent == "":
		return "/" + child
	case child == "":
		return "/" + parent
	}
	return "/" + parent + "/" + child
}

// convertPath rewrites Echo's `:name` path-parameter syntax to OpenAPI's
// `{name}` template syntax and returns one Parameter per `:name` it finds.
// Wildcard segments (`*`) become `{wildcard}` parameters.
func convertPath(p string) (string, []Parameter) {
	parts := strings.Split(p, "/")
	var params []Parameter
	for i, part := range parts {
		switch {
		case strings.HasPrefix(part, ":"):
			name := strings.TrimPrefix(part, ":")
			parts[i] = "{" + name + "}"
			params = append(params, Parameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		case part == "*":
			parts[i] = "{wildcard}"
			params = append(params, Parameter{
				Name:     "wildcard",
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}
	}
	return strings.Join(parts, "/"), params
}

// setMethod assigns op to the right slot on item based on the HTTP method
// string. Unknown methods are silently dropped — extend the switch when you
// add support for a non-standard verb.
func setMethod(item *PathItem, method string, op *Operation) {
	switch strings.ToUpper(method) {
	case "GET":
		item.Get = op
	case "POST":
		item.Post = op
	case "PUT":
		item.Put = op
	case "DELETE":
		item.Delete = op
	case "PATCH":
		item.Patch = op
	case "OPTIONS":
		item.Options = op
	case "HEAD":
		item.Head = op
	}
}

// sortTags is a stable, allocation-free in-place sort. We keep it small and
// dependency-free rather than reaching for sort.Slice for a list that's
// almost always tiny (handful of modules).
func sortTags(tags []Tag) {
	for i := 1; i < len(tags); i++ {
		for j := i; j > 0 && tags[j-1].Name > tags[j].Name; j-- {
			tags[j-1], tags[j] = tags[j], tags[j-1]
		}
	}
}
