// Package api embeds the OpenAPI specification (the source of truth at
// api/openapi.yaml) into the binary so the documentation endpoints work
// identically in local dev and in the container, with no runtime file
// dependency.
package api

import _ "embed"

// OpenAPISpec is the raw bytes of the OpenAPI 3 specification.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
