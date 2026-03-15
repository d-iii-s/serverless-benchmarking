package flowgen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseWrk2Params tests
// ---------------------------------------------------------------------------

func TestParseWrk2Params_Basic(t *testing.T) {
	cfg, err := ParseWrk2Params("-t2 -c100 -d30s -R2000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Rate != 2000 {
		t.Errorf("expected rate 2000, got %d", cfg.Rate)
	}
	if cfg.Duration != 30 {
		t.Errorf("expected duration 30, got %d", cfg.Duration)
	}
	if cfg.TotalRequests() != 60000 {
		t.Errorf("expected total 60000, got %d", cfg.TotalRequests())
	}
}

func TestParseWrk2Params_Minutes(t *testing.T) {
	cfg, err := ParseWrk2Params("-d2m -R500")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Duration != 120 {
		t.Errorf("expected 120s, got %d", cfg.Duration)
	}
	if cfg.TotalRequests() != 60000 {
		t.Errorf("expected 60000, got %d", cfg.TotalRequests())
	}
}

func TestParseWrk2Params_Hours(t *testing.T) {
	cfg, err := ParseWrk2Params("-d1h -R10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Duration != 3600 {
		t.Errorf("expected 3600s, got %d", cfg.Duration)
	}
}

func TestParseWrk2Params_MissingRate(t *testing.T) {
	_, err := ParseWrk2Params("-d30s")
	if err == nil {
		t.Fatal("expected error for missing -R")
	}
}

func TestParseWrk2Params_MissingDuration(t *testing.T) {
	_, err := ParseWrk2Params("-R2000")
	if err == nil {
		t.Fatal("expected error for missing -d")
	}
}

// ---------------------------------------------------------------------------
// ParseDSL tests
// ---------------------------------------------------------------------------

func TestParseDSL(t *testing.T) {
	dsl, err := ParseDSL(filepath.Join("testdata", "test-dsl.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dsl.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(dsl.Stages))
	}

	s1, ok := dsl.Stages["stage1"]
	if !ok {
		t.Fatal("stage1 not found")
	}
	if len(s1.Flow) != 3 {
		t.Fatalf("stage1: expected 3 flow nodes, got %d", len(s1.Flow))
	}

	// Check entry node.
	if s1.Flow[0].Name != "node1" {
		t.Errorf("expected node name 'node1', got %q", s1.Flow[0].Name)
	}
	if !s1.Flow[0].EntryNode {
		t.Error("expected node1 to be entry node")
	}
	if s1.Flow[0].Endpoint != "/pets" {
		t.Errorf("expected endpoint /pets, got %q", s1.Flow[0].Endpoint)
	}
	if s1.Flow[0].Method != "POST" {
		t.Errorf("expected method POST, got %q", s1.Flow[0].Method)
	}
	if len(s1.Flow[0].Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(s1.Flow[0].Edges))
	}
	if s1.Flow[0].Edges[0].To != "node2" || s1.Flow[0].Edges[0].Weight != 0.3 {
		t.Errorf("unexpected edge[0]: %+v", s1.Flow[0].Edges[0])
	}
	if s1.Flow[0].Edges[1].To != "node3" || s1.Flow[0].Edges[1].Weight != 0.7 {
		t.Errorf("unexpected edge[1]: %+v", s1.Flow[0].Edges[1])
	}

	// Non-entry node.
	if s1.Flow[1].EntryNode {
		t.Error("node2 should not be entry node")
	}
}

// ---------------------------------------------------------------------------
// ComputeBodyCounts tests
// ---------------------------------------------------------------------------

func TestComputeBodyCounts_Stage1(t *testing.T) {
	dsl, err := ParseDSL(filepath.Join("testdata", "test-dsl.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stage := dsl.Stages["stage1"]
	// R=2000, d=30s => total=60000
	counts, err := ComputeBodyCounts(stage, 60000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 3 {
		t.Fatalf("expected 3 counts, got %d", len(counts))
	}

	// node1 (POST, entry) -> 60000 bodies.
	assertCount(t, counts, "node1", 60000)
	// node2 (GET) -> 0 bodies (GET has no body).
	assertCount(t, counts, "node2", 0)
	// node3 (GET) -> 0 bodies.
	assertCount(t, counts, "node3", 0)
}

func TestComputeBodyCounts_Stage2(t *testing.T) {
	dsl, err := ParseDSL(filepath.Join("testdata", "test-dsl.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stage := dsl.Stages["stage2"]
	// R=1000, d=10s => total=10000
	counts, err := ComputeBodyCounts(stage, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCount(t, counts, "node1", 10000)
	assertCount(t, counts, "node2", 0) // GET
}

func TestComputeBodyCounts_NoEntryNode(t *testing.T) {
	stage := Stage{
		Flow: []FlowNode{
			{Name: "n1", Endpoint: "/a", Method: "POST", EntryNode: false},
		},
	}
	_, err := ComputeBodyCounts(stage, 100)
	if err == nil {
		t.Fatal("expected error when no entry node")
	}
}

func assertCount(t *testing.T, counts []NodeBodyCount, name string, expected int) {
	t.Helper()
	for _, c := range counts {
		if c.NodeName == name {
			if c.Count != expected {
				t.Errorf("node %q: expected count %d, got %d", name, expected, c.Count)
			}
			return
		}
	}
	t.Errorf("node %q not found in counts", name)
}

// ---------------------------------------------------------------------------
// GenerateFlowBodies integration test (with mock generator)
// ---------------------------------------------------------------------------

func TestGenerateFlowBodies_WithMock(t *testing.T) {
	ctx := context.Background()
	dslPath := filepath.Join("testdata", "test-dsl.yaml")
	specPath := "unused_in_mock.yaml" // the mock doesn't use this
	outputDir := t.TempDir()

	// Mock generator: writes a JSON array with `count` dummy objects.
	mockGen := func(_ context.Context, _, endpoint, method string, count int, outputPath string) error {
		bodies := make([]map[string]any, count)
		for i := range bodies {
			bodies[i] = map[string]any{
				"endpoint": endpoint,
				"method":   method,
				"index":    i,
			}
		}
		data, _ := json.Marshal(bodies)
		return os.WriteFile(outputPath, data, 0o644)
	}

	err := GenerateFlowBodies(ctx, dslPath, specPath, outputDir, mockGen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify stage1 output.
	verifyFile(t, filepath.Join(outputDir, "stage1", "node1.json"), 60000)
	verifyEmptyArray(t, filepath.Join(outputDir, "stage1", "node2.json"))
	verifyEmptyArray(t, filepath.Join(outputDir, "stage1", "node3.json"))

	// Verify stage2 output.
	verifyFile(t, filepath.Join(outputDir, "stage2", "node1.json"), 10000)
	verifyEmptyArray(t, filepath.Join(outputDir, "stage2", "node2.json"))
}

func verifyFile(t *testing.T, path string, expectedCount int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var arr []any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("%s: not a valid JSON array: %v", path, err)
	}
	if len(arr) != expectedCount {
		t.Errorf("%s: expected %d items, got %d", path, expectedCount, len(arr))
	}
}

func verifyEmptyArray(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != "[]" {
		t.Errorf("%s: expected empty array '[]', got %q", path, string(data))
	}
}

// ---------------------------------------------------------------------------
// Integration test with real Python/Schemathesis (skipped if unavailable)
// ---------------------------------------------------------------------------

func skipIfNoPython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found – skipping")
	}
	cmd := exec.Command("python3", "-c", "import schemathesis")
	if err := cmd.Run(); err != nil {
		t.Skip("schemathesis not installed – skipping")
	}
}

func TestGenerateFlowBodies_RealPython(t *testing.T) {
	skipIfNoPython(t)

	ctx := context.Background()
	tmpDir := t.TempDir()
	dslPath := filepath.Join(tmpDir, "test-dsl-small.yaml")
	// Use the petstore spec from datagen testdata.
	specPath := filepath.Join("..", "datagen", "testdata", "petstore.yaml")
	outputDir := filepath.Join(tmpDir, "output")

	// Write a small DSL for the real test (keep counts low).
	smallDSL := `stages:
  stage1:
    wrk2params: -t1 -c1 -d1s -R3
    flow:
      - node1:
        endpoint: /pets
        method: POST
        entrynode: true
        edges:
          - to: node2
            weight: 1.0
      - node2:
        endpoint: /pets
        method: GET
`
	if err := os.WriteFile(dslPath, []byte(smallDSL), 0o644); err != nil {
		t.Fatalf("failed to write small DSL: %v", err)
	}

	// Use the real datagen.GenerateRequestBodies via import.
	// We can't directly import datagen here without a circular dependency
	// concern, so we replicate the Python call inline.
	realGen := func(ctx context.Context, specP, endpoint, method string, count int, outputPath string) error {
		scriptPath := filepath.Join("..", "..", "..", "scripts", "generate_bodies.py")
		cmd := exec.CommandContext(ctx, "python3",
			scriptPath,
			"--spec-path", specP,
			"--endpoint", endpoint,
			"--method", method,
			"--count", fmt.Sprintf("%d", count),
			"--output", outputPath,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("python script failed: %w\n%s", err, string(out))
		}
		return nil
	}

	err := GenerateFlowBodies(ctx, dslPath, specPath, outputDir, realGen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// node1 is POST /pets with count=3 -> should have 3 bodies.
	verifyFile(t, filepath.Join(outputDir, "stage1", "node1.json"), 3)
	// node2 is GET -> empty.
	verifyEmptyArray(t, filepath.Join(outputDir, "stage1", "node2.json"))
}

