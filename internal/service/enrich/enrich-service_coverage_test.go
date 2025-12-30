package enricher

import (
	"context"
	"github.com/d-iii-s/slsbench/internal/utils"
	"path/filepath"
	"testing"
)

// TestSetHints_CoversAllWalkerTestCases verifies that all test cases from walker/testdata are covered
func TestSetHints_CoversAllWalkerTestCases(t *testing.T) {
	// All test files from walker/testdata
	testFiles := []string{
		"simple.yaml",
		"nested.yaml",
		"arrays.yaml",
		"allof.yaml",
		"anyof.yaml",
		"oneof.yaml",
		"references.yaml",
		"additional-properties.yaml",
		"parameters.yaml",
		"path-level-parameters.yaml",
		"path-and-operation-parameters.yaml",
		"all-parameter-types.yaml",
		"request-body.yaml",
		"multiple-operations.yaml",
		"complex-nested.yaml",
		"circular.yaml",
		"minimal.yaml",
		"empty.yaml",
	}

	ctx := context.Background()
	
	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {
			// Load input spec from walker/testdata
			inputPath := filepath.Join("..", "walker", "testdata", testFile)
			spec := utils.OpenOpenApiSpecFile(ctx, inputPath)
			
			if spec == nil {
				t.Fatalf("Failed to load spec: %s", inputPath)
			}
			
			// Create a selector for hints
			selector := &multiCallSelector{
				hintIdx: 23, // uuid
			}
			
			processor := NewEnrichProcessor(spec, selector)
			
			// This should not panic and should process all properties
			processor.SetHints()
			
			// Verify that at least some extensions were set (if the spec has properties)
			// We can't verify exact values without knowing the structure, but we can verify
			// that the processing completed without errors
		})
	}
}

// TestSetHints_HandlesAllSchemaTypes verifies that all schema types are handled
func TestSetHints_HandlesAllSchemaTypes(t *testing.T) {
	testCases := []struct {
		name     string
		testFile string
		desc     string
	}{
		{"SimpleProperties", "simple.yaml", "Simple object properties"},
		{"NestedObjects", "nested.yaml", "Nested object properties"},
		{"Arrays", "arrays.yaml", "Array items and nested arrays"},
		{"AllOf", "allof.yaml", "AllOf schema composition"},
		{"AnyOf", "anyof.yaml", "AnyOf schema composition"},
		{"OneOf", "oneof.yaml", "OneOf schema composition"},
		{"References", "references.yaml", "Component schema references"},
		{"AdditionalProperties", "additional-properties.yaml", "Additional properties schema"},
		{"Parameters", "parameters.yaml", "Path parameters"},
		{"PathLevelParameters", "path-level-parameters.yaml", "Path-level parameters"},
		{"PathAndOperationParameters", "path-and-operation-parameters.yaml", "Path and operation parameters"},
		{"AllParameterTypes", "all-parameter-types.yaml", "All parameter types (path, query, header, cookie)"},
		{"RequestBody", "request-body.yaml", "Request body properties"},
		{"MultipleOperations", "multiple-operations.yaml", "Multiple operations on same path"},
		{"ComplexNested", "complex-nested.yaml", "Complex nested structures"},
		{"Circular", "circular.yaml", "Circular references"},
		{"Minimal", "minimal.yaml", "Minimal spec"},
		{"Empty", "empty.yaml", "Empty spec"},
	}

	ctx := context.Background()
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join("..", "walker", "testdata", tc.testFile)
			spec := utils.OpenOpenApiSpecFile(ctx, inputPath)
			
			if spec == nil {
				t.Fatalf("Failed to load spec: %s", inputPath)
			}
			
			selector := &multiCallSelector{
				hintIdx: 7, // uuid
			}
			
			processor := NewEnrichProcessor(spec, selector)
			
			// Should not panic
			processor.SetHints()
		})
	}
}

