package walker

import (
	"context"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestVisitor is a test implementation of Visitor that collects visited endpoints and properties
type TestVisitor struct {
	// Endpoint tracking
	paths            []string
	operations       []string
	parameters       []string
	requestBodies    []string
	responses        []string
	componentSchemas []string
	
	// Property tracking
	visitedProperties []VisitedProperty
}

type VisitedProperty struct {
	Path   string
	Schema *openapi3.SchemaRef
}

// Endpoint visitor methods
func (v *TestVisitor) VisitPath(path string, pathItem *openapi3.PathItem) {
	v.paths = append(v.paths, path)
}

func (v *TestVisitor) VisitOperation(path string, method string, operation *openapi3.Operation) {
	v.operations = append(v.operations, path)
}

func (v *TestVisitor) VisitParameter(path string, parameter *openapi3.Parameter) {
	v.parameters = append(v.parameters, path)
}

func (v *TestVisitor) VisitRequestBody(path string, contentType string, requestBody *openapi3.RequestBody) {
	v.requestBodies = append(v.requestBodies, path)
}

func (v *TestVisitor) VisitResponse(path string, statusCode string, contentType string, response *openapi3.Response) {
	v.responses = append(v.responses, path)
}

func (v *TestVisitor) VisitComponentSchema(name string, schema *openapi3.SchemaRef) {
	v.componentSchemas = append(v.componentSchemas, name)
}

// Property visitor methods
func (v *TestVisitor) VisitProperty(path string, schema *openapi3.SchemaRef) {
	v.visitedProperties = append(v.visitedProperties, VisitedProperty{
		Path:   path,
		Schema: schema,
	})
}

// Getter methods for endpoints
func (v *TestVisitor) GetPaths() []string {
	sort.Strings(v.paths)
	return v.paths
}

func (v *TestVisitor) GetOperations() []string {
	sort.Strings(v.operations)
	return v.operations
}

func (v *TestVisitor) GetParameters() []string {
	sort.Strings(v.parameters)
	return v.parameters
}

func (v *TestVisitor) GetRequestBodies() []string {
	sort.Strings(v.requestBodies)
	return v.requestBodies
}

func (v *TestVisitor) GetResponses() []string {
	sort.Strings(v.responses)
	return v.responses
}

func (v *TestVisitor) GetComponentSchemas() []string {
	sort.Strings(v.componentSchemas)
	return v.componentSchemas
}

// Getter methods for properties
func (v *TestVisitor) GetVisitedPaths() []string {
	paths := make([]string, len(v.visitedProperties))
	for i, vp := range v.visitedProperties {
		paths[i] = vp.Path
	}
	sort.Strings(paths)
	return paths
}

func (v *TestVisitor) Reset() {
	v.paths = nil
	v.operations = nil
	v.parameters = nil
	v.requestBodies = nil
	v.responses = nil
	v.componentSchemas = nil
	v.visitedProperties = nil
}

func TestWalkAllProperties_SimpleSchema(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json.email",
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_NestedObjects(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "nested.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json.address",
		"/api/users/GET/response/200/application/json.address.city",
		"/api/users/GET/response/200/application/json.address.country",
		"/api/users/GET/response/200/application/json.address.street",
		"/api/users/GET/response/200/application/json.address.zipCode",
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_Arrays(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "arrays.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json[].id",
		"/api/users/GET/response/200/application/json[].name",
		"/api/users/GET/response/200/application/json[].tags",
		"/api/users/GET/response/200/application/json[].tags[].name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_AllOf(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "allof.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json/allOf[0].id",
		"/api/users/GET/response/200/application/json/allOf[0].name",
		"/api/users/GET/response/200/application/json/allOf[1].email",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_AnyOf(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "anyof.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json/anyOf[0].id",
		"/api/users/GET/response/200/application/json/anyOf[0].name",
		"/api/users/GET/response/200/application/json/anyOf[1].email",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_OneOf(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "oneof.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json/oneOf[0].id",
		"/api/users/GET/response/200/application/json/oneOf[0].name",
		"/api/users/GET/response/200/application/json/oneOf[1].email",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_References(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "references.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"#/components/schemas/Address.city",
		"#/components/schemas/Address.country",
		"#/components/schemas/Address.street",
		"#/components/schemas/Address.zipCode",
		"#/components/schemas/User.address",
		"#/components/schemas/User.address.city",
		"#/components/schemas/User.address.country",
		"#/components/schemas/User.address.street",
		"#/components/schemas/User.address.zipCode",
		"#/components/schemas/User.id",
		"#/components/schemas/User.name",
		"/api/users/GET/response/200/application/json.address",
		"/api/users/GET/response/200/application/json.address.city",
		"/api/users/GET/response/200/application/json.address.country",
		"/api/users/GET/response/200/application/json.address.street",
		"/api/users/GET/response/200/application/json.address.zipCode",
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_AdditionalProperties(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "additional-properties.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json/additionalProperties",
		"/api/users/GET/response/200/application/json/additionalProperties.key",
		"/api/users/GET/response/200/application/json/additionalProperties.value",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_RequestParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/{id}/GET/parameter/id",
		"/api/users/{id}/GET/response/200/application/json.id",
		"/api/users/{id}/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_PathLevelParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "path-level-parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/vets/{vetId}/GET/response/200/application/json.id",
		"/api/vets/{vetId}/GET/response/200/application/json.name",
		"/api/vets/{vetId}/parameter/vetId",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_AllParameterTypes(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "all-parameter-types.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/{userId}/GET/parameter/X-Request-ID",
		"/api/users/{userId}/GET/parameter/limit",
		"/api/users/{userId}/GET/parameter/page",
		"/api/users/{userId}/GET/parameter/sessionId",
		"/api/users/{userId}/GET/response/200/application/json.id",
		"/api/users/{userId}/parameter/userId",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_PathAndOperationParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "path-and-operation-parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/vets/{vetId}/pets/{petId}/GET/parameter/includeHistory",
		"/api/vets/{vetId}/pets/{petId}/GET/response/200/application/json.id",
		"/api/vets/{vetId}/pets/{petId}/parameter/petId",
		"/api/vets/{vetId}/pets/{petId}/parameter/vetId",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_RequestBody(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "request-body.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/POST/requestBody/application/json.email",
		"/api/users/POST/requestBody/application/json.name",
		"/api/users/POST/response/201/application/json.id",
		"/api/users/POST/response/201/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkAllProperties_ComplexNested(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "complex-nested.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	// Verify we get all nested properties
	if len(paths) == 0 {
		t.Error("Expected to find properties in complex nested schema")
	}

	// Check for deeply nested properties
	hasDeepNesting := false
	for _, path := range paths {
		if len(path) > 50 { // Deep nesting will have long paths
			hasDeepNesting = true
			break
		}
	}
	if !hasDeepNesting {
		t.Error("Expected to find deeply nested properties")
	}
}

func TestWalkAllProperties_EmptySpec(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "empty.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	if len(paths) != 0 {
		t.Errorf("Expected no paths for empty spec, got %v", paths)
	}
}

func TestWalkAllProperties_MinimalSpec(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "minimal.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	// Minimal spec should have at least some paths
	if len(paths) == 0 {
		t.Error("Expected to find at least one property in minimal spec")
	}
}

func TestWalkAllProperties_CircularReference(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "circular.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	// This should not cause infinite loop
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	// Should handle circular references gracefully
	if len(paths) == 0 {
		t.Error("Expected to find properties even with circular references")
	}
}

func TestWalkAllProperties_NilVisitor(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	// Should not panic with nil visitor
	walker.WalkSpec(nil)
}

// Tests for WalkSpec

func TestWalkEndpoints_SimpleSchema(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetPaths()
	expectedPaths := []string{"/api/users"}
	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}

	operations := visitor.GetOperations()
	expectedOperations := []string{"/api/users/GET"}
	if !reflect.DeepEqual(operations, expectedOperations) {
		t.Errorf("Expected operations %v, got %v", expectedOperations, operations)
	}

	responses := visitor.GetResponses()
	expectedResponses := []string{"/api/users/GET/response/200/application/json"}
	if !reflect.DeepEqual(responses, expectedResponses) {
		t.Errorf("Expected responses %v, got %v", expectedResponses, responses)
	}
}

func TestWalkEndpoints_PathLevelParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "path-level-parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	parameters := visitor.GetParameters()
	expectedParameters := []string{"/api/vets/{vetId}/parameter/vetId"}
	if !reflect.DeepEqual(parameters, expectedParameters) {
		t.Errorf("Expected parameters %v, got %v", expectedParameters, parameters)
	}
}

func TestWalkEndpoints_AllParameterTypes(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "all-parameter-types.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	parameters := visitor.GetParameters()
	expectedParameters := []string{
		"/api/users/{userId}/GET/parameter/X-Request-ID",
		"/api/users/{userId}/GET/parameter/limit",
		"/api/users/{userId}/GET/parameter/page",
		"/api/users/{userId}/GET/parameter/sessionId",
		"/api/users/{userId}/parameter/userId",
	}
	if !reflect.DeepEqual(parameters, expectedParameters) {
		t.Errorf("Expected parameters %v, got %v", expectedParameters, parameters)
	}
}

func TestWalkEndpoints_RequestBody(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "request-body.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	requestBodies := visitor.GetRequestBodies()
	expectedRequestBodies := []string{"/api/users/POST/requestBody/application/json"}
	if !reflect.DeepEqual(requestBodies, expectedRequestBodies) {
		t.Errorf("Expected request bodies %v, got %v", expectedRequestBodies, requestBodies)
	}
}

func TestWalkEndpoints_MultipleOperations(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "multiple-operations.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	operations := visitor.GetOperations()
	expectedOperations := []string{"/api/users/GET", "/api/users/POST"}
	if !reflect.DeepEqual(operations, expectedOperations) {
		t.Errorf("Expected operations %v, got %v", expectedOperations, operations)
	}
}

func TestWalkEndpoints_References(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "references.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	componentSchemas := visitor.GetComponentSchemas()
	expectedSchemas := []string{"Address", "User"}
	if !reflect.DeepEqual(componentSchemas, expectedSchemas) {
		t.Errorf("Expected component schemas %v, got %v", expectedSchemas, componentSchemas)
	}
}

func TestWalkEndpoints_EmptySpec(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "empty.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	if len(visitor.GetPaths()) != 0 {
		t.Error("Expected no paths for empty spec")
	}
}

func TestWalkEndpoints_NilVisitor(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	// Should not panic with nil visitor
	walker.WalkSpec(nil)
}

func TestWalkEndpoints_NilSpec(t *testing.T) {
	walker := NewWalker(nil)
	visitor := &TestVisitor{}

	// Should not panic with nil spec
	walker.WalkSpec(visitor)

	if len(visitor.GetPaths()) != 0 {
		t.Error("Expected no paths with nil spec")
	}
}

func TestWalkEndpoints_PathAndOperationParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "path-and-operation-parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	parameters := visitor.GetParameters()
	expectedParameters := []string{
		"/api/vets/{vetId}/pets/{petId}/GET/parameter/includeHistory",
		"/api/vets/{vetId}/pets/{petId}/parameter/petId",
		"/api/vets/{vetId}/pets/{petId}/parameter/vetId",
	}
	if !reflect.DeepEqual(parameters, expectedParameters) {
		t.Errorf("Expected parameters %v, got %v", expectedParameters, parameters)
	}
}

func TestWalkEndpoints_PropertyVisitorOnly(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/GET/response/200/application/json.email",
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkEndpoints_BothVisitors(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "simple.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	// Verify endpoint visitor was called
	paths := visitor.GetPaths()
	if len(paths) == 0 {
		t.Error("Expected endpoint visitor to be called")
	}

	operations := visitor.GetOperations()
	if len(operations) == 0 {
		t.Error("Expected endpoint visitor to visit operations")
	}

	// Verify property visitor was called
	propertyPaths := visitor.GetVisitedPaths()
	if len(propertyPaths) == 0 {
		t.Error("Expected property visitor to be called")
	}

	// Verify we got both endpoint and property visits
	expectedPropertyPaths := []string{
		"/api/users/GET/response/200/application/json.email",
		"/api/users/GET/response/200/application/json.id",
		"/api/users/GET/response/200/application/json.name",
	}
	if !reflect.DeepEqual(propertyPaths, expectedPropertyPaths) {
		t.Errorf("Expected property paths %v, got %v", expectedPropertyPaths, propertyPaths)
	}
}

func TestWalkEndpoints_PropertyVisitorWithParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	expectedPaths := []string{
		"/api/users/{id}/GET/parameter/id",
		"/api/users/{id}/GET/response/200/application/json.id",
		"/api/users/{id}/GET/response/200/application/json.name",
	}

	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("Expected paths %v, got %v", expectedPaths, paths)
	}
}

func TestWalkEndpoints_BothVisitorsWithParameters(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "path-level-parameters.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	// Verify endpoint visitor got parameters
	endpointParams := visitor.GetParameters()
	expectedEndpointParams := []string{"/api/vets/{vetId}/parameter/vetId"}
	if !reflect.DeepEqual(endpointParams, expectedEndpointParams) {
		t.Errorf("Expected endpoint parameters %v, got %v", expectedEndpointParams, endpointParams)
	}

	// Verify property visitor got parameter schema
	propertyPaths := visitor.GetVisitedPaths()
	hasParameterProperty := false
	for _, path := range propertyPaths {
		if path == "/api/vets/{vetId}/parameter/vetId" {
			hasParameterProperty = true
			break
		}
	}
	if !hasParameterProperty {
		t.Error("Expected property visitor to visit parameter schema")
	}
}

func TestWalkAllProperties_NilSpec(t *testing.T) {
	walker := NewWalker(nil)
	visitor := &TestVisitor{}

	// Should not panic with nil spec
	walker.WalkSpec(visitor)

	if len(visitor.visitedProperties) != 0 {
		t.Error("Expected no visits with nil spec")
	}
}

func TestWalkAllProperties_MultipleOperations(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "multiple-operations.yaml")
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	walker := NewWalker(spec)

	visitor := &TestVisitor{}
	walker.WalkSpec(visitor)

	paths := visitor.GetVisitedPaths()
	// Should visit properties from multiple operations
	hasGet := false
	hasPost := false
	for _, path := range paths {
		if contains(path, "/GET/") {
			hasGet = true
		}
		if contains(path, "/POST/") {
			hasPost = true
		}
	}

	if !hasGet {
		t.Error("Expected to find properties from GET operation")
	}
	if !hasPost {
		t.Error("Expected to find properties from POST operation")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
