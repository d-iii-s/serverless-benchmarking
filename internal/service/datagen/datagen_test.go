package datagen

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// skipIfNoPython skips the test when python3 or schemathesis is not available.
func skipIfNoPython(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found in PATH – skipping")
	}

	cmd := exec.Command("python3", "-c", "import schemathesis")
	if err := cmd.Run(); err != nil {
		t.Skip("schemathesis Python package not installed – skipping")
	}
}

func TestGenerateRequestBodies_Valid(t *testing.T) {
	skipIfNoPython(t)

	ctx := context.Background()
	specPath := filepath.Join("testdata", "petstore.yaml")
	outputPath := filepath.Join(t.TempDir(), "bodies.json")

	err := GenerateRequestBodies(ctx, specPath, "/pets", "POST", 3, outputPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and parse the output file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var bodies []map[string]any
	if err := json.Unmarshal(data, &bodies); err != nil {
		t.Fatalf("output is not a valid JSON array of objects: %v", err)
	}

	if len(bodies) != 3 {
		t.Fatalf("expected 3 bodies, got %d", len(bodies))
	}

	// Each body must have the required "name" field
	for i, body := range bodies {
		if _, ok := body["name"]; !ok {
			t.Errorf("body[%d] missing required field 'name': %v", i, body)
		}
	}
}

func TestGenerateRequestBodies_InvalidSpec(t *testing.T) {
	ctx := context.Background()
	outputPath := filepath.Join(t.TempDir(), "bodies.json")

	err := GenerateRequestBodies(ctx, "/nonexistent/openapi.yaml", "/pets", "POST", 3, outputPath)
	if err == nil {
		t.Fatal("expected error for non-existent spec path, got nil")
	}
}

func TestGenerateRequestBodies_InvalidCount(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "petstore.yaml")
	outputPath := filepath.Join(t.TempDir(), "bodies.json")

	err := GenerateRequestBodies(ctx, specPath, "/pets", "POST", 0, outputPath)
	if err == nil {
		t.Fatal("expected error for count=0, got nil")
	}

	err = GenerateRequestBodies(ctx, specPath, "/pets", "POST", -1, outputPath)
	if err == nil {
		t.Fatal("expected error for count=-1, got nil")
	}
}

func TestGenerateRequestBodies_UnknownEndpoint(t *testing.T) {
	skipIfNoPython(t)

	ctx := context.Background()
	specPath := filepath.Join("testdata", "petstore.yaml")
	outputPath := filepath.Join(t.TempDir(), "bodies.json")

	err := GenerateRequestBodies(ctx, specPath, "/nonexistent", "POST", 3, outputPath)
	if err == nil {
		t.Fatal("expected error for unknown endpoint, got nil")
	}
}
