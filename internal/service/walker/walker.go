package walker

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type Walker struct {
	spec *openapi3.T
}

// Visitor visits both endpoints and schema properties in the OpenAPI specification.
// All methods are optional - implement only the methods you need.
type Visitor interface {
	// Endpoint visiting methods
	VisitPath(path string, pathItem *openapi3.PathItem)
	VisitOperation(path string, method string, operation *openapi3.Operation)
	VisitParameter(path string, parameter *openapi3.Parameter)
	VisitRequestBody(path string, contentType string, requestBody *openapi3.RequestBody)
	VisitResponse(path string, statusCode string, contentType string, response *openapi3.Response)
	VisitComponentSchema(name string, schema *openapi3.SchemaRef)
	
	// Property visiting methods
	VisitProperty(path string, schema *openapi3.SchemaRef)
}

func NewWalker(spec *openapi3.T) *Walker {
	return &Walker{spec: spec}
}

// WalkSpec traverses all endpoints and properties in the OpenAPI specification
// and applies the Visitor to each element found.
// The visitor can be nil - only non-nil visitor methods will be called.
func (w *Walker) WalkSpec(visitor Visitor) {
	if w.spec == nil || visitor == nil {
		return
	}

	visitedSchemas := make(map[*openapi3.Schema]struct{})

	// Walk through all paths and operations
	for pathName, pathItem := range w.spec.Paths.Map() {
		if pathItem == nil {
			continue
		}

		visitor.VisitPath(pathName, pathItem)

		// Walk through path-level parameters (apply to all operations)
		for _, paramRef := range pathItem.Parameters {
			if paramRef != nil && paramRef.Value != nil {
				paramPath := fmt.Sprintf("%s/parameter/%s", pathName, paramRef.Value.Name)

				visitor.VisitParameter(paramPath, paramRef.Value)

				// Visit parameter schema properties if schema exists
				if paramRef.Value.Schema != nil {
					visitor.VisitProperty(paramPath, paramRef.Value.Schema)
					w.walkSchema(paramPath, paramRef.Value.Schema, visitor, visitedSchemas)
				}
			}
		}

		// Walk through operations (GET, POST, etc.)
		for operationName, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}

			operationPath := fmt.Sprintf("%s/%s", pathName, operationName)

			visitor.VisitOperation(operationPath, operationName, operation)

			// Walk through operation-level parameters
			for _, paramRef := range operation.Parameters {
				if paramRef != nil && paramRef.Value != nil {
					paramPath := fmt.Sprintf("%s/parameter/%s", operationPath, paramRef.Value.Name)

					visitor.VisitParameter(paramPath, paramRef.Value)

					// Visit parameter schema properties if schema exists
					if paramRef.Value.Schema != nil {
						visitor.VisitProperty(paramPath, paramRef.Value.Schema)
						w.walkSchema(paramPath, paramRef.Value.Schema, visitor, visitedSchemas)
					}
				}
			}

			// Walk through request body
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				for contentType, mediaType := range operation.RequestBody.Value.Content {
					bodyPath := fmt.Sprintf("%s/requestBody/%s", operationPath, contentType)

					visitor.VisitRequestBody(bodyPath, contentType, operation.RequestBody.Value)

					// Visit request body schema properties if schema exists
					if mediaType != nil && mediaType.Schema != nil {
						w.walkSchema(bodyPath, mediaType.Schema, visitor, visitedSchemas)
					}
				}
			}

			// Walk through responses
			for statusCode, responseRef := range operation.Responses.Map() {
				if responseRef != nil && responseRef.Value != nil {
					for contentType, mediaType := range responseRef.Value.Content {
						responsePath := fmt.Sprintf("%s/response/%s/%s", operationPath, statusCode, contentType)

						visitor.VisitResponse(responsePath, statusCode, contentType, responseRef.Value)

						// Visit response schema properties if schema exists
						if mediaType != nil && mediaType.Schema != nil {
							w.walkSchema(responsePath, mediaType.Schema, visitor, visitedSchemas)
						}
					}
				}
			}
		}
	}

	// Walk through components schemas
	if w.spec.Components != nil {
		for schemaName, schemaRef := range w.spec.Components.Schemas {
			if schemaRef != nil {
				visitor.VisitComponentSchema(schemaName, schemaRef)

				// Visit component schema properties
				schemaPath := fmt.Sprintf("#/components/schemas/%s", schemaName)
				w.walkSchema(schemaPath, schemaRef, visitor, visitedSchemas)
			}
		}
	}
}

// walkSchema recursively walks through a schema and visits properties
func (w *Walker) walkSchema(basePath string, schemaRef *openapi3.SchemaRef, visitor Visitor, visitedSchemas map[*openapi3.Schema]struct{}) {
	if schemaRef == nil {
		return
	}

	if schemaRef.Value == nil {
		return
	}

	schema := schemaRef.Value

	// Avoid infinite loops with circular references
	if _, seen := visitedSchemas[schema]; seen {
		return
	}
	visitedSchemas[schema] = struct{}{}
	defer delete(visitedSchemas, schema)

	// Visit properties of the current schema
	for propName, propRef := range schema.Properties {
		if propRef != nil {
			propPath := fmt.Sprintf("%s.%s", basePath, propName)
			visitor.VisitProperty(propPath, propRef)
			// Recursively walk into the property schema
			w.walkSchema(propPath, propRef, visitor, visitedSchemas)
		}
	}

	// Walk through array items
	if schema.Items != nil {
		itemPath := basePath + "[]"
		w.walkSchema(itemPath, schema.Items, visitor, visitedSchemas)
	}

	// Walk through allOf schemas
	for i, allOfRef := range schema.AllOf {
		if allOfRef != nil {
			allOfPath := fmt.Sprintf("%s/allOf[%d]", basePath, i)
			w.walkSchema(allOfPath, allOfRef, visitor, visitedSchemas)
		}
	}

	// Walk through anyOf schemas
	for i, anyOfRef := range schema.AnyOf {
		if anyOfRef != nil {
			anyOfPath := fmt.Sprintf("%s/anyOf[%d]", basePath, i)
			w.walkSchema(anyOfPath, anyOfRef, visitor, visitedSchemas)
		}
	}

	// Walk through oneOf schemas
	for i, oneOfRef := range schema.OneOf {
		if oneOfRef != nil {
			oneOfPath := fmt.Sprintf("%s/oneOf[%d]", basePath, i)
			w.walkSchema(oneOfPath, oneOfRef, visitor, visitedSchemas)
		}
	}

	// Walk through not schema
	if schema.Not != nil {
		notPath := basePath + "/not"
		w.walkSchema(notPath, schema.Not, visitor, visitedSchemas)
	}

	// Walk through additionalProperties (for map types)
	if schema.AdditionalProperties.Schema != nil {
		additionalPath := basePath + "/additionalProperties"
		// Visit the additionalProperties schema itself
		visitor.VisitProperty(additionalPath, schema.AdditionalProperties.Schema)
		w.walkSchema(additionalPath, schema.AdditionalProperties.Schema, visitor, visitedSchemas)
	}
}
