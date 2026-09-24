package docs

import _ "embed"

// OpenAPIYAML contains the raw OpenAPI 3.0 specification.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte

// SwaggerHTML contains the interactive Swagger UI page.
//
//go:embed swagger.html
var SwaggerHTML []byte
