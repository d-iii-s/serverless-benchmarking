package enricher

import (
	"context"
	"github.com/d-iii-s/slsbench/internal/utils"
	"path/filepath"
	"testing"
)

// multiCallSelector handles hint selection calls
type multiCallSelector struct {
	hintIdx int
}

func (m *multiCallSelector) Run() (int, string, error) {
	return m.hintIdx, utils.HintOptions[m.hintIdx], nil
}

func TestSetHints_ExtensionNames(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, filepath.Join("..", "walker", "testdata", "simple.yaml"))
	
	// Create a selector for hints
	// For id: hint=uuid (index 23)
	// For name: hint=string (index 18)
	// For email: hint=email (index 4)
	selector := &multiCallSelector{
		hintIdx: 23, // uuid
	}
	
	processor := NewEnrichProcessor(spec, selector)
	processor.SetHints()
	
	// Check that extensions were set on response properties
	pathItem := spec.Paths.Map()["/api/users"]
	if pathItem == nil {
		t.Fatal("path not found")
	}
	
	operation := pathItem.Get
	if operation == nil {
		t.Fatal("operation not found")
	}
	
	response := operation.Responses.Map()["200"]
	if response == nil || response.Value == nil {
		t.Fatal("response not found")
	}
	
	mediaType := response.Value.Content["application/json"]
	if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
		t.Fatal("schema not found")
	}
	
	schema := mediaType.Schema.Value
	
	// Check id property
	idProp := schema.Properties["id"]
	if idProp == nil || idProp.Value == nil {
		t.Fatal("id property not found")
	}
	
	if idProp.Value.Extensions == nil {
		t.Error("Extensions map not initialized for id")
	} else {
		if _, ok := idProp.Value.Extensions["x-user-hint"]; !ok {
			t.Error("x-user-hint extension not set on id")
		}
	}
	
	// Check name property
	nameProp := schema.Properties["name"]
	if nameProp == nil || nameProp.Value == nil {
		t.Fatal("name property not found")
	}
	
	if nameProp.Value.Extensions == nil {
		t.Error("Extensions map not initialized for name")
	} else {
		if _, ok := nameProp.Value.Extensions["x-user-hint"]; !ok {
			t.Error("x-user-hint extension not set on name")
		}
	}
}

func TestSetHints_ArraysApplyToItems(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, filepath.Join("..", "walker", "testdata", "arrays.yaml"))

	selector := &multiCallSelector{
		hintIdx: 23, // uuid
	}

	processor := NewEnrichProcessor(spec, selector)
	processor.SetHints()

	pathItem := spec.Paths.Map()["/api/users"]
	if pathItem == nil {
		t.Fatal("path not found")
	}

	op := pathItem.Get
	if op == nil {
		t.Fatal("get operation not found")
	}

	resp := op.Responses.Map()["200"]
	if resp == nil || resp.Value == nil {
		t.Fatal("response not found")
	}

	media := resp.Value.Content["application/json"]
	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		t.Fatal("schema not found")
	}

	// Root is an array; ensure hints are applied to items, not the array container
	rootSchema := media.Schema.Value
	if rootSchema.Extensions != nil {
		if _, hasHint := rootSchema.Extensions["x-user-hint"]; hasHint {
			t.Error("array container should not have x-user-hint")
		}
	}

	if rootSchema.Items == nil || rootSchema.Items.Value == nil {
		t.Fatal("array items schema not found")
	}

	itemSchema := rootSchema.Items.Value
	idProp := itemSchema.Properties["id"]
	if idProp == nil || idProp.Value == nil {
		t.Fatal("id property on items not found")
	}
	if idProp.Value.Extensions == nil {
		t.Fatal("id property extensions not set")
	}
	if _, ok := idProp.Value.Extensions["x-user-hint"]; !ok {
		t.Error("id property missing x-user-hint")
	}
}

func TestSetHints_ObjectsApplyToFieldsNotContainer(t *testing.T) {
	ctx := context.Background()
	spec := utils.OpenOpenApiSpecFile(ctx, filepath.Join("..", "walker", "testdata", "nested.yaml"))

	selector := &multiCallSelector{
		hintIdx: 23, // uuid
	}

	processor := NewEnrichProcessor(spec, selector)
	processor.SetHints()

	pathItem := spec.Paths.Map()["/api/users"]
	if pathItem == nil {
		t.Fatal("path not found")
	}

	op := pathItem.Get
	if op == nil {
		t.Fatal("get operation not found")
	}

	resp := op.Responses.Map()["200"]
	if resp == nil || resp.Value == nil {
		t.Fatal("response not found")
	}

	media := resp.Value.Content["application/json"]
	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		t.Fatal("schema not found")
	}

	root := media.Schema.Value
	addressProp := root.Properties["address"]
	if addressProp == nil || addressProp.Value == nil {
		t.Fatal("address property not found")
	}

	// Object container (address) should not have hints
	if addressProp.Value.Extensions != nil {
		if _, hasHint := addressProp.Value.Extensions["x-user-hint"]; hasHint {
			t.Error("object container should not have x-user-hint")
		}
	}

	// Its scalar fields should have hints
	street := addressProp.Value.Properties["street"]
	if street == nil || street.Value == nil {
		t.Fatal("street property not found on address")
	}
	if street.Value.Extensions == nil {
		t.Fatal("street property extensions not set")
	}
	if _, ok := street.Value.Extensions["x-user-hint"]; !ok {
		t.Error("street property missing x-user-hint")
	}
}


