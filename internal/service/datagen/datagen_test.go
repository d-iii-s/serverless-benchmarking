package datagen

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestGenerateRequestBodiesData_ReturnsRemovedError(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "petstore.yaml")
	_, err := GenerateRequestBodiesData(ctx, specPath, "/pets", "POST", 1)
	if err == nil {
		t.Fatal("expected removed-per-operation error, got nil")
	}
}

func TestGenerateStatefulChainsData_RequiresChain(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("testdata", "petstore.yaml")
	_, err := GenerateStatefulChainsData(ctx, specPath, "", "http://localhost:9966/petclinic/api", false)
	if err == nil {
		t.Fatal("expected error for missing chain, got nil")
	}
}

func TestGenerateStatefulChainsData_BasicShape(t *testing.T) {
	if os.Getenv("SLSBENCH_RUN_STATEFUL_TESTS") == "" {
		t.Skip("set SLSBENCH_RUN_STATEFUL_TESTS=1 to run stateful datagen integration test")
	}
	skipIfNoPython(t)
	resp, err := http.Get("http://localhost:9966/petclinic/api/owners")
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 500 {
		t.Skip("localhost petclinic is not available for stateful integration test")
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	ctx := context.Background()
	specPath := filepath.Join("..", "..", "..", "workdir", "spring-petclinic-rest", "openapi.yml")
	chains, err := GenerateStatefulChainsData(ctx, specPath, "addOwner", "http://localhost:9966/petclinic/api", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(chains) == 0 {
		t.Fatal("expected at least one generated chain")
	}
	if len(chains[0].Steps) == 0 {
		t.Fatal("expected generated chain to include steps")
	}
	if chains[0].IterationID != 0 {
		t.Fatalf("expected iterationIndex 0, got %d", chains[0].IterationID)
	}
	if chains[0].Steps[0].FlowID == "" {
		t.Fatal("expected generated step to include flowId")
	}
}

func TestProjectMinimalIterations_Wrk2ReplayFields(t *testing.T) {
	minimal := ProjectMinimalIterations([]StatefulChain{
		{
			IterationID: 9,
			ChainIndex:  2,
			Steps: []StatefulStep{
				{
					FlowID:       "lifecycle/createOwner",
					Method:       "POST",
					PathTemplate: "/owners/{ownerId}",
					PathParams:   map[string]any{"ownerId": "/id"},
					Headers:      map[string]any{"content-type": "application/json"},
					Query:        map[string]any{"q": "abc"},
					ResolvedPath: "/owners/10",
					RequestBody:  map[string]any{"ownerId": "/id"},
					Status:       201,
				},
			},
		},
	})
	if len(minimal) != 1 || len(minimal[0].Steps) != 1 {
		t.Fatalf("unexpected projected size: %+v", minimal)
	}
	if minimal[0].Steps[0].FlowID != "lifecycle/createOwner" {
		t.Fatalf("unexpected flowId in projection: %q", minimal[0].Steps[0].FlowID)
	}
	if minimal[0].Steps[0].ResolvedPath != "/owners/10" {
		t.Fatalf("unexpected resolvedPath in projection: %q", minimal[0].Steps[0].ResolvedPath)
	}
	if minimal[0].Steps[0].PathTemplate != "/owners/{ownerId}" {
		t.Fatalf("unexpected pathTemplate in projection: %q", minimal[0].Steps[0].PathTemplate)
	}
	if minimal[0].Steps[0].PathParams["ownerId"] != "/id" {
		t.Fatalf("unexpected pathParameters in projection: %#v", minimal[0].Steps[0].PathParams)
	}
	raw, err := json.Marshal(minimal[0].Steps[0])
	if err != nil {
		t.Fatalf("failed to marshal minimal step: %v", err)
	}
	text := string(raw)
	if !containsAll(text, []string{"flowId", "method", "pathTemplate", "pathParameters", "headers", "query", "resolvedPath", "requestBody"}) {
		t.Fatalf("expected minimal keys in output json, got %s", text)
	}
	if containsAny(text, []string{"status", "operationId"}) {
		t.Fatalf("unexpected rich fields in minimal output json: %s", text)
	}
	if strings.Contains(text, "$response.") {
		t.Fatalf("unexpected response template in projected output json: %s", text)
	}
	if !strings.Contains(text, "/id") {
		t.Fatalf("expected json pointer replacement in projected output json: %s", text)
	}
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
