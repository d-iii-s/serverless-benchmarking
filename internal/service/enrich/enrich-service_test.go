package enricher

import (
	"context"
	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"path/filepath"
	"testing"
)

type mockPromptSelector struct{}

func (m *mockPromptSelector) Run() (int, string, error) {
	return 0, "string", nil
}

// loadTestSpec is a helper function to load a test OpenAPI spec
func loadTestSpec(t *testing.T, filename string) *openapi3.T {
	t.Helper()
	ctx := context.Background()
	specPath := filepath.Join("testdata", filename)
	spec := utils.OpenOpenApiSpecFile(ctx, specPath)
	return spec
}

func TestNewEnrichProcessor(t *testing.T) {
	tests := []struct {
		name     string
		spec     *openapi3.T
		selector model.PromptSelector
		wantNil  bool
	}{
		{
			name:     "valid spec and selector",
			spec:     loadTestSpec(t, "simple.yaml"),
			selector: &mockPromptSelector{},
			wantNil:  false,
		},
		{
			name:     "nil spec",
			spec:     nil,
			selector: &mockPromptSelector{},
			wantNil:  false,
		},
		{
			name:     "nil selector",
			spec:     loadTestSpec(t, "simple.yaml"),
			selector: nil,
			wantNil:  false,
		},
		{
			name:     "both nil",
			spec:     nil,
			selector: nil,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewEnrichProcessor(tt.spec, tt.selector)

			if tt.wantNil && processor != nil {
				t.Error("NewEnrichProcessor expected nil, got non-nil")
			}
			if !tt.wantNil && processor == nil {
				t.Fatal("NewEnrichProcessor returned nil")
			}

			if processor != nil {
				if processor.spec != tt.spec {
					t.Errorf("processor.spec = %v, want %v", processor.spec, tt.spec)
				}
				if processor.selector != tt.selector {
					t.Errorf("processor.selector = %v, want %v", processor.selector, tt.selector)
				}
				if processor.walker == nil {
					t.Error("processor.walker is nil")
				}
			}
		})
	}
}

func TestSetHints(t *testing.T) {
	tests := []struct {
		name     string
		specFile string
	}{
		{
			name:     "simple spec",
			specFile: "simple.yaml",
		},
		{
			name:     "nested objects",
			specFile: "nested.yaml",
		},
		{
			name:     "with parameters",
			specFile: "parameters.yaml",
		},
		{
			name:     "with request body",
			specFile: "request-body.yaml",
		},
		{
			name:     "empty spec",
			specFile: "empty.yaml",
		},
		{
			name:     "minimal spec",
			specFile: "minimal.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := loadTestSpec(t, tt.specFile)
			selector := &mockPromptSelector{}
			processor := NewEnrichProcessor(spec, selector)

			// SetHints should not panic
			processor.SetHints()

			// If we get here, the test passes
		})
	}
}

func TestSetHints_NilSpec(t *testing.T) {
	selector := &mockPromptSelector{}
	processor := NewEnrichProcessor(nil, selector)

	// SetHints should not panic with nil spec
	processor.SetHints()
}

func TestHintSetterVisitor_ImplementsVisitor(t *testing.T) {
	visitor := HintSetterVisitor{}

	// Verify all methods exist and can be called without panic
	t.Run("VisitPath", func(t *testing.T) {
		visitor.VisitPath("/test", nil)
	})

	t.Run("VisitOperation", func(t *testing.T) {
		visitor.VisitOperation("/test", "GET", nil)
	})

	t.Run("VisitParameter", func(t *testing.T) {
		param := &openapi3.Parameter{
			Name: "testParam",
		}
		visitor.VisitParameter("/test/param", param)
	})

	t.Run("VisitRequestBody", func(t *testing.T) {
		visitor.VisitRequestBody("/test/body", "application/json", nil)
	})

	t.Run("VisitResponse", func(t *testing.T) {
		visitor.VisitResponse("/test/response", "200", "application/json", nil)
	})

	t.Run("VisitComponentSchema", func(t *testing.T) {
		visitor.VisitComponentSchema("TestSchema", nil)
	})

	t.Run("VisitProperty", func(t *testing.T) {
		visitor.VisitProperty("/test.property", nil)
	})
}

func TestHintSetterVisitor_WithRealData(t *testing.T) {
	spec := loadTestSpec(t, "simple.yaml")
	visitor := HintSetterVisitor{}

	// Test visitor with actual path items
	for pathName, pathItem := range spec.Paths.Map() {
		if pathItem != nil {
			visitor.VisitPath(pathName, pathItem)

			for operationName, operation := range pathItem.Operations() {
				if operation != nil {
					visitor.VisitOperation(pathName+"/"+operationName, operationName, operation)

					for _, paramRef := range operation.Parameters {
						if paramRef != nil && paramRef.Value != nil {
							visitor.VisitParameter(pathName+"/parameter/"+paramRef.Value.Name, paramRef.Value)
						}
					}
				}
			}
		}
	}
}

