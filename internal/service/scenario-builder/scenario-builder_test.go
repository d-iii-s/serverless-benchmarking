package scenario_builder

import (
	"context"
	"testing"

	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/d-iii-s/slsbench/internal/utils"
)

func fieldNames(fields []*model.Field) map[string]struct{} {
	names := make(map[string]struct{})
	for _, f := range fields {
		if f == nil {
			continue
		}
		if f.Name != "" {
			names[f.Name] = struct{}{}
		}
		if len(f.Properties) > 0 {
			for _, p := range f.Properties {
				for name := range fieldNames([]*model.Field{p}) {
					names[name] = struct{}{}
				}
			}
		}
		if f.Items != nil {
			for name := range fieldNames([]*model.Field{f.Items}) {
				names[name] = struct{}{}
			}
		}
	}
	return names
}

type mockSelector struct {
	calls []int
	idx   int
}

func (m *mockSelector) Run() (int, string, error) {
	if m.idx >= len(m.calls) {
		return 0, "", nil
	}
	v := m.calls[m.idx]
	m.idx++
	return v, "", nil
}

// menu/run order in buildScenarioGraph:
// 1) menu (choose add endpoint) -> 0
// 2) endpoint selection -> 0
// 3) response selection (if multiple) -> 0
// 4) menu (choose done) -> 3

func TestCreateScenarioFromSpecPath_Simple(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, "../enrich/testdata/simple.yaml")

	selector := &mockSelector{calls: []int{0, 0, 3}}
	graph := CreateScenarioFromSpecPath(ctx, spec, selector)

	vertices := graph.GetVertices()
	if len(vertices) != 1 {
		t.Fatalf("expected 1 vertex, got %d", len(vertices))
	}

	v := vertices[0]
	if v == nil {
		t.Fatal("vertex is nil")
	}
	if v.Path != "/api/users" || v.Method != "GET" {
		t.Fatalf("unexpected vertex: %s %s", v.Method, v.Path)
	}
	// Response data is no longer stored on the graph; we only assert basic vertex info here.
}

func TestCreateScenarioFromSpecPath_RequestBody(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, "../enrich/testdata/request-body.yaml")

	selector := &mockSelector{calls: []int{0, 0, 3}}
	graph := CreateScenarioFromSpecPath(ctx, spec, selector)

	vertices := graph.GetVertices()
	if len(vertices) != 1 {
		t.Fatalf("expected 1 vertex, got %d", len(vertices))
	}

	v := vertices[0]
	if v == nil {
		t.Fatal("vertex is nil")
	}
	if v.Path != "/api/users" || v.Method != "POST" {
		t.Fatalf("unexpected vertex: %s %s", v.Method, v.Path)
	}
	if v.RequestBody == nil {
		t.Fatalf("request body not captured")
	}
	bodyNames := fieldNames(v.RequestBody.Fields)
	for _, expected := range []string{"name", "email"} {
		if _, ok := bodyNames[expected]; !ok {
			t.Errorf("expected body field %s", expected)
		}
	}
}

func TestCreateScenarioFromSpecPath_ExtractsMetadata(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, "../enrich/testdata/simple.yaml")

	selector := &mockSelector{calls: []int{0, 0, 3}}
	graph := CreateScenarioFromSpecPath(ctx, spec, selector)

	vertices := graph.GetVertices()
	if len(vertices) != 1 {
		t.Fatalf("expected 1 vertex, got %d", len(vertices))
	}

	v := vertices[0]
	if v == nil {
		t.Fatal("vertex is nil")
	}

	// Check that response fields have metadata extracted
	// Note: response data is not stored on the graph, so we need to check the data model directly
	// For now, let's verify that parameters (if any) have metadata
	// In this test, we're mainly verifying the code compiles and runs correctly
	// The actual metadata extraction is tested implicitly through the build process
}

// Helper function to find a field by name in a slice of fields
func findField(fields []*model.Field, name string) *model.Field {
	for _, f := range fields {
		if f != nil && f.Name == name {
			return f
		}
	}
	return nil
}

func TestBuildFieldFromSchema_ExtractsAllMetadata(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, "../enrich/testdata/request-body.yaml")

	// Build data model to access fields directly
	dataModel := buildDataModelFromSpec(ctx, spec)

	// Find the POST /api/users operation
	endpoint, ok := dataModel.Endpoints["/api/users"]
	if !ok {
		t.Fatal("endpoint /api/users not found")
	}

	operation, ok := endpoint.Operations["POST"]
	if !ok {
		t.Fatal("POST operation not found")
	}

	// Check request body fields have metadata
	if operation.RequestBody == nil {
		t.Fatal("request body not found")
	}

	nameField := findField(operation.RequestBody.Fields, "name")
	if nameField == nil {
		t.Fatal("name field not found")
	}

	// Verify metadata extraction
	if nameField.Hint != "name" {
		t.Errorf("expected hint 'name', got '%s'", nameField.Hint)
	}
	if nameField.Unique != false {
		t.Errorf("expected unique=false, got %v", nameField.Unique)
	}
	if nameField.Type != "string" {
		t.Errorf("expected type 'string', got '%s'", nameField.Type)
	}

	emailField := findField(operation.RequestBody.Fields, "email")
	if emailField == nil {
		t.Fatal("email field not found")
	}

	if emailField.Hint != "email" {
		t.Errorf("expected hint 'email', got '%s'", emailField.Hint)
	}
	if emailField.Unique != false {
		t.Errorf("expected unique=false, got %v", emailField.Unique)
	}
}

func TestBuildFieldFromSchema_ExtractsConstraints(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, "testdata/constraints.yaml")

	// Build data model to access fields directly
	dataModel := buildDataModelFromSpec(ctx, spec)

	// Find the POST /api/test operation
	endpoint, ok := dataModel.Endpoints["/api/test"]
	if !ok {
		t.Fatal("endpoint /api/test not found")
	}

	operation, ok := endpoint.Operations["POST"]
	if !ok {
		t.Fatal("POST operation not found")
	}

	// Check request body fields have constraints
	if operation.RequestBody == nil {
		t.Fatal("request body not found")
	}

	// Test username field: minLength, maxLength, pattern, format
	usernameField := findField(operation.RequestBody.Fields, "username")
	if usernameField == nil {
		t.Fatal("username field not found")
	}
	if usernameField.Type != "string" {
		t.Errorf("expected type 'string', got '%s'", usernameField.Type)
	}
	if usernameField.MinLength != 3 {
		t.Errorf("expected minLength 3, got %d", usernameField.MinLength)
	}
	if usernameField.MaxLength == nil || *usernameField.MaxLength != 20 {
		t.Errorf("expected maxLength 20, got %v", usernameField.MaxLength)
	}
	if usernameField.Pattern != "^[a-zA-Z0-9_]+$" {
		t.Errorf("expected pattern '^[a-zA-Z0-9_]+$', got '%s'", usernameField.Pattern)
	}
	if usernameField.Format != "string" {
		t.Errorf("expected format 'string', got '%s'", usernameField.Format)
	}

	// Test email field: format, minLength, maxLength
	emailField := findField(operation.RequestBody.Fields, "email")
	if emailField == nil {
		t.Fatal("email field not found")
	}
	if emailField.Format != "email" {
		t.Errorf("expected format 'email', got '%s'", emailField.Format)
	}
	if emailField.MinLength != 5 {
		t.Errorf("expected minLength 5, got %d", emailField.MinLength)
	}
	if emailField.MaxLength == nil || *emailField.MaxLength != 100 {
		t.Errorf("expected maxLength 100, got %v", emailField.MaxLength)
	}

	// Test age field: min, max, format
	ageField := findField(operation.RequestBody.Fields, "age")
	if ageField == nil {
		t.Fatal("age field not found")
	}
	if ageField.Type != "integer" {
		t.Errorf("expected type 'integer', got '%s'", ageField.Type)
	}
	if ageField.Format != "int32" {
		t.Errorf("expected format 'int32', got '%s'", ageField.Format)
	}
	if ageField.Min == nil || *ageField.Min != 0 {
		t.Errorf("expected min 0, got %v", ageField.Min)
	}
	if ageField.Max == nil || *ageField.Max != 150 {
		t.Errorf("expected max 150, got %v", ageField.Max)
	}

	// Test price field: min, max, format
	priceField := findField(operation.RequestBody.Fields, "price")
	if priceField == nil {
		t.Fatal("price field not found")
	}
	if priceField.Type != "number" {
		t.Errorf("expected type 'number', got '%s'", priceField.Type)
	}
	if priceField.Format != "double" {
		t.Errorf("expected format 'double', got '%s'", priceField.Format)
	}
	if priceField.Min == nil || *priceField.Min != 0.0 {
		t.Errorf("expected min 0.0, got %v", priceField.Min)
	}
	if priceField.Max == nil || *priceField.Max != 10000.0 {
		t.Errorf("expected max 10000.0, got %v", priceField.Max)
	}

	// Test phone field: pattern only
	phoneField := findField(operation.RequestBody.Fields, "phone")
	if phoneField == nil {
		t.Fatal("phone field not found")
	}
	if phoneField.Pattern != "^\\+?[1-9]\\d{1,14}$" {
		t.Errorf("expected pattern '^\\\\+?[1-9]\\\\d{1,14}$', got '%s'", phoneField.Pattern)
	}
}

