package dslvalidator

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// ensure embed is referenced so linters don't complain
var _ embed.FS

//go:embed schema/dsl.schema.json
var dslSchemaBytes []byte

var compiledSchema *jsonschema.Schema

func init() {
	var schemaDoc any
	if err := json.Unmarshal(dslSchemaBytes, &schemaDoc); err != nil {
		panic(fmt.Errorf("failed to unmarshal embedded DSL schema: %w", err))
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("dsl.schema.json", schemaDoc); err != nil {
		panic(fmt.Errorf("failed to register embedded DSL schema: %w", err))
	}

	schema, err := compiler.Compile("dsl.schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to compile embedded DSL schema: %w", err))
	}
	compiledSchema = schema
}

// ValidateDSL validates a parsed YAML/JSON document against the embedded DSL schema.
// The instance should be the result of unmarshalling YAML/JSON into interface{}.
func ValidateDSL(_ context.Context, instance any) error {
	if compiledSchema == nil {
		return fmt.Errorf("DSL schema is not initialized")
	}
	return compiledSchema.Validate(instance)
}
