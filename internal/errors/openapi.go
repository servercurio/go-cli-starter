package errors

import e "github.com/joomcode/errorx"

var (
	// OpenAPIErrors is the errorx namespace for failures originating in the
	// OpenAPI subsystem (spec generation, marshaling, optional Swagger UI
	// mount).
	OpenAPIErrors = e.NewNamespace("openapi")

	// OpenAPIGenerationFailed marks failure to build, marshal, or register
	// the generated OpenAPI document or its associated route modules. Wrap
	// errors from yaml.Marshal / json.Marshal / RegisterModule with this
	// type so deploy automation can distinguish API-doc failures from
	// general initialization errors.
	OpenAPIGenerationFailed = OpenAPIErrors.NewType("generation_failed")
)
