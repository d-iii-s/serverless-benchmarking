package dslvalidator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateDSL_Valid(t *testing.T) {
	ctx := context.Background()

	path := filepath.Join("testdata", "valid-dsl.yaml")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open valid DSL file: %v", err)
	}
	defer f.Close()

	var instance any
	if err := yaml.NewDecoder(f).Decode(&instance); err != nil {
		t.Fatalf("failed to decode valid DSL YAML: %v", err)
	}

	if err := ValidateDSL(ctx, instance); err != nil {
		t.Fatalf("expected no error for valid DSL, got: %v", err)
	}
}

func TestValidateDSL_Invalid(t *testing.T) {
	ctx := context.Background()

	path := filepath.Join("testdata", "invalid-dsl.yaml")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open invalid DSL file: %v", err)
	}
	defer f.Close()

	var instance any
	if err := yaml.NewDecoder(f).Decode(&instance); err != nil {
		t.Fatalf("failed to decode invalid DSL YAML: %v", err)
	}

	if err := ValidateDSL(ctx, instance); err == nil {
		t.Fatalf("expected validation error for invalid DSL, got nil")
	}
}
