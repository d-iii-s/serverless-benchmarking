package flowgen

import (
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
	if s1.Flow[0].OperationID != "createPet" {
		t.Errorf("expected operationId createPet, got %q", s1.Flow[0].OperationID)
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
