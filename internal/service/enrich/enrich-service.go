package enricher

import (
	"fmt"
	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/d-iii-s/slsbench/internal/service/walker"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"strings"
)

type EnrichProcessor struct {
	selector model.PromptSelector
	spec     *openapi3.T
	walker   *walker.Walker
}

func NewEnrichProcessor(spec *openapi3.T, selector model.PromptSelector) *EnrichProcessor {
	return &EnrichProcessor{
		selector: selector,
		spec:     spec,
		walker:   walker.NewWalker(spec),
	}
}

type HintSetterVisitor struct {
	selector model.PromptSelector
}

func newHintSetterVisitor(selector model.PromptSelector) HintSetterVisitor {
	return HintSetterVisitor{selector: selector}
}

func (h HintSetterVisitor) VisitPath(path string, pathItem *openapi3.PathItem) {
}

func (h HintSetterVisitor) VisitOperation(path string, method string, operation *openapi3.Operation) {
}

func (h HintSetterVisitor) VisitParameter(path string, parameter *openapi3.Parameter) {
	setHints(h, parameter, path)
}

func (h HintSetterVisitor) VisitRequestBody(path string, contentType string, requestBody *openapi3.RequestBody) {
}

func (h HintSetterVisitor) VisitResponse(path string, statusCode string, contentType string, response *openapi3.Response) {
}

func (h HintSetterVisitor) VisitComponentSchema(name string, schema *openapi3.SchemaRef) {
	log.Printf("Visiting component schema: %s", name)
}

func (h HintSetterVisitor) VisitProperty(path string, schemaRef *openapi3.SchemaRef) {
	if schemaRef == nil || schemaRef.Value == nil {
		return
	}
	// Use the full walker path when asking the user to set hints for this field,
	// so they see complete context (path, operation, response, nested property, etc.).
	setHintsOnSchema(h, schemaRef.Value, path)
}

func (enrichProcessor *EnrichProcessor) SetHints() {
	visitor := newHintSetterVisitor(enrichProcessor.selector)
	enrichProcessor.walker.WalkSpec(visitor)
}

func (enrichProcessor *EnrichProcessor) GetSpec() *openapi3.T {
	return enrichProcessor.spec
}

// setHints sets hints on a parameter, using the full walker path for context.
func setHints(h HintSetterVisitor, parameter *openapi3.Parameter, path string) {
	if parameter == nil || parameter.Schema == nil || parameter.Schema.Value == nil {
		return
	}

	// Initialize Extensions if nil
	if parameter.Schema.Value.Extensions == nil {
		parameter.Schema.Value.Extensions = make(map[string]interface{})
	}

	// Use the full walker path for the parameter when prompting.
	setHintsOnSchema(h, parameter.Schema.Value, path)
}

// setHintsOnSchema sets x-user-hint on a schema
func setHintsOnSchema(h HintSetterVisitor, schema *openapi3.Schema, fieldName string) {
	if schema == nil {
		return
	}

	// If this is an array, apply hints to the items instead of the array container.
	if schema.Type != nil && len(*schema.Type) > 0 && (*schema.Type)[0] == "array" {
		if schema.Items != nil && schema.Items.Value != nil {
			setHintsOnSchema(h, schema.Items.Value, fieldName+"[]")
		}
		return
	}

	// If this is an object with properties, do not set hints on the container itself.
	// The walker will traverse and apply hints to individual properties instead.
	if schema.Type != nil && len(*schema.Type) > 0 && (*schema.Type)[0] == "object" && len(schema.Properties) > 0 {
		return
	}

	// Initialize Extensions if nil
	if schema.Extensions == nil {
		schema.Extensions = make(map[string]interface{})
	}

	// Set x-user-hint
	selector := h.selector
	if selector == nil {
		readablePath := formatPathForPrompt(fieldName)
		selector = model.NewPromptSelector(fmt.Sprintf("Select hint for %s", readablePath), utils.HintOptions)
	}
	idx, _, err := selector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}
	schema.Extensions["x-user-hint"] = utils.HintOptions[idx]
}

// formatPathForPrompt converts a technical path like "/wrk2-api/movie-info/write/POST/requestBody/application/json.avg_rating"
// into a more readable format like "avg_rating (POST /wrk2-api/movie-info/write - Request Body)"
func formatPathForPrompt(path string) string {
	// Extract field name from the end (after last dot)
	var fieldName string
	var basePath string
	if dotIdx := strings.LastIndex(path, "."); dotIdx >= 0 {
		fieldName = path[dotIdx+1:]
		basePath = path[:dotIdx]
	} else {
		// No dot, might be a parameter or top-level field
		basePath = path
		// Try to extract field name from last path segment
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// If last part looks like a field name (not a location type), use it
			if lastPart != "requestBody" && lastPart != "response" && lastPart != "parameter" {
				fieldName = lastPart
			}
		}
	}

	// Parse the base path to extract endpoint, method, and location
	parts := strings.Split(basePath, "/")
	// Remove empty first element if path starts with /
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		if fieldName != "" {
			return fieldName
		}
		return cleanPathForDisplay(path)
	}

	var endpointPath string
	var method string
	var location string

	// Find method (GET, POST, PUT, etc.) - it's usually in uppercase
	methods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "OPTIONS": true, "HEAD": true, "TRACE": true}

	methodIdx := -1
	for i, part := range parts {
		if methods[strings.ToUpper(part)] {
			method = strings.ToUpper(part)
			methodIdx = i
			// Everything before method is the endpoint path
			if i > 0 {
				endpointPath = "/" + strings.Join(parts[:i], "/")
			} else if i == 0 {
				// Method is first, no endpoint path
				endpointPath = ""
			}
			break
		}
	}

	// Extract location (requestBody, response, parameter) after method
	if methodIdx >= 0 && methodIdx+1 < len(parts) {
		locationPart := parts[methodIdx+1]
		switch locationPart {
		case "requestBody":
			location = "Request Body"
		case "response":
			// Next part is status code
			if methodIdx+2 < len(parts) {
				location = fmt.Sprintf("Response %s", parts[methodIdx+2])
			} else {
				location = "Response"
			}
		case "parameter":
			location = "Parameter"
			// Parameter name might be next, use it as field name if we don't have one
			if methodIdx+2 < len(parts) && fieldName == "" {
				fieldName = parts[methodIdx+2]
			}
		}
	}

	// Build readable string
	var result strings.Builder
	if fieldName != "" {
		result.WriteString(fieldName)
	}

	// Add context in parentheses
	contextParts := []string{}
	if method != "" {
		contextParts = append(contextParts, method)
	}
	if endpointPath != "" {
		contextParts = append(contextParts, endpointPath)
	}
	if location != "" {
		contextParts = append(contextParts, location)
	}

	if len(contextParts) > 0 {
		if fieldName != "" {
			result.WriteString(" (")
		}
		result.WriteString(strings.Join(contextParts, " - "))
		if fieldName != "" {
			result.WriteString(")")
		}
	} else if fieldName == "" {
		// No context and no field name, use cleaned path
		return cleanPathForDisplay(path)
	}

	if result.Len() == 0 {
		return cleanPathForDisplay(path)
	}
	return result.String()
}

// cleanPathForDisplay removes redundant parts and makes path more readable
func cleanPathForDisplay(path string) string {
	// Replace slashes with " → " for readability
	path = strings.ReplaceAll(path, "/", " → ")
	// Replace dots with " > " for nested properties
	path = strings.ReplaceAll(path, ".", " > ")
	return path
}
