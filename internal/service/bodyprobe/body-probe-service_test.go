package bodyprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/d-iii-s/slsbench/internal/service/datagen"
	"github.com/d-iii-s/slsbench/internal/service/flowgen"
)

func TestRunWithGenerator_RequiresFlowPath(t *testing.T) {
	outDir := t.TempDir()
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		return nil, nil
	}

	err := runWithGenerator(context.Background(), " ", "unused", outDir, 9966, generate, false)
	if err == nil {
		t.Fatal("expected error for empty flow path")
	}
}

func TestRequestTargetWithMargin(t *testing.T) {
	if got := requestTargetWithMargin(100); got != 111 {
		t.Fatalf("expected 111, got %d", got)
	}
	if got := requestTargetWithMargin(1); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := requestTargetWithMargin(0); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestStageTraverser_WeightedRoundRobin(t *testing.T) {
	stage := flowgen.Stage{
		Flow: []flowgen.FlowNode{
			{
				Name:        "entry",
				OperationID: "opEntry",
				EntryNode:   true,
				Edges: []flowgen.Edge{
					{To: "a", Weight: 0.75},
					{To: "b", Weight: 0.25},
				},
			},
			{Name: "a", OperationID: "opA"},
			{Name: "b", OperationID: "opB"},
		},
	}
	traverser, err := newStageTraverser("stage1", stage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var secondOps []string
	for i := 0; i < 4; i++ {
		ops, err := traverser.NextChainOperationIDs()
		if err != nil {
			t.Fatalf("traversal failed: %v", err)
		}
		if len(ops) != 2 {
			t.Fatalf("expected 2 operations in chain, got %d", len(ops))
		}
		secondOps = append(secondOps, ops[1])
	}
	if !slices.Equal(secondOps, []string{"opA", "opA", "opB", "opA"}) {
		t.Fatalf("unexpected WRR sequence: %#v", secondOps)
	}
}

func TestRunWithGenerator_WritesPerStageIterations(t *testing.T) {
	flowPath := writeTempFlow(t, `
stages:
  alpha:
    wrk2params: -t1 -c1 -d1s -R2
    flow:
      - start:
        operationId: addOwner
        entrynode: true
  beta:
    wrk2params: -t1 -c1 -d1s -R1
    flow:
      - begin:
        operationId: addOwner
        entrynode: true
`)
	outDir := t.TempDir()
	callCount := 0
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		callCount++
		return []datagen.StatefulChain{
			{
				IterationID: callCount,
				ChainIndex:  callCount,
				Steps: []datagen.StatefulStep{
					{
						FlowID:       fmt.Sprintf("flow-%d-a", callCount),
						Method:       "POST",
						PathTemplate: "/owners/{ownerId}",
						PathParams:   map[string]any{"ownerId": "addOwner.responseBody#/id"},
						Headers:      map[string]any{"content-type": "application/json"},
						Query:        map[string]any{"q": "x"},
						ResolvedPath: "/owners",
						RequestBody: map[string]any{
							"n":            callCount,
							"step":         1,
							"ownerPointer": "addOwner.requestBody#/owner/id",
						},
						Status:       201,
					},
					{
						FlowID:       fmt.Sprintf("flow-%d-b", callCount),
						Method:       "GET",
						PathTemplate: "/owners/{ownerId}",
						PathParams:   map[string]any{"ownerId": "/id"},
						Headers:      map[string]any{"accept": "application/json"},
						Query:        map[string]any{"q2": "y"},
						ResolvedPath: "/owners/1",
						RequestBody:  nil,
						Status:       201,
					},
				},
			},
		}, nil
	}
	if err := runWithGenerator(context.Background(), flowPath, "unused", outDir, 9966, generate, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	alphaFiles := listIterationFiles(t, filepath.Join(outDir, "alpha"))
	betaFiles := listIterationFiles(t, filepath.Join(outDir, "beta"))
	// alpha target: ceil(2*1*1.1)=3; beta target: ceil(1*1*1.1)=2.
	// Each accepted iteration in this test has 2 steps, so alpha needs 2 files and beta needs 1 file.
	if len(alphaFiles) != 2 {
		t.Fatalf("expected 2 alpha iteration files, got %d", len(alphaFiles))
	}
	if len(betaFiles) != 1 {
		t.Fatalf("expected 1 beta iteration file, got %d", len(betaFiles))
	}
	if got := totalStepCountFromFiles(t, filepath.Join(outDir, "alpha"), alphaFiles); got < 3 {
		t.Fatalf("expected alpha total step count >= 3, got %d", got)
	}
	if got := totalStepCountFromFiles(t, filepath.Join(outDir, "beta"), betaFiles); got < 2 {
		t.Fatalf("expected beta total step count >= 2, got %d", got)
	}
	iteration := readIterationFile(t, filepath.Join(outDir, "alpha", alphaFiles[0]))
	if len(iteration.Steps) != 2 {
		t.Fatalf("expected one full chain iteration with 2 steps, got %d", len(iteration.Steps))
	}
	sample := iteration.Steps[0]
	if sample.Method == "" || sample.ResolvedPath == "" {
		t.Fatalf("missing wrk2 required fields in output: %+v", sample)
	}
	if sample.PathTemplate != "/owners/{ownerId}" {
		t.Fatalf("missing pathTemplate in output: %+v", sample)
	}
	if sample.PathParams["ownerId"] != "alpha.addOwner.responseBody#/id" {
		t.Fatalf("missing pathParameters in output: %#v", sample.PathParams)
	}
	if _, ok := sample.Headers["content-type"]; !ok {
		t.Fatalf("missing headers in output: %+v", sample.Headers)
	}
	if _, ok := sample.Query["q"]; !ok {
		t.Fatalf("missing query in output: %+v", sample.Query)
	}
	if sample.RequestBody == nil {
		t.Fatal("missing requestBody in output")
	}
	requestBody, ok := sample.RequestBody.(map[string]any)
	if !ok {
		t.Fatalf("unexpected requestBody shape: %#v", sample.RequestBody)
	}
	if requestBody["ownerPointer"] != "alpha.addOwner.requestBody#/owner/id" {
		t.Fatalf("missing stage-scoped request pointer in output: %#v", requestBody)
	}
}

func TestRunWithGeneratorAndWorkdir_CreatesResultSubdir(t *testing.T) {
	flowPath := writeTempFlow(t, `
stages:
  stage1:
    wrk2params: -t1 -c1 -d1s -R1
    flow:
      - createOwner:
        operationId: addOwner
        entrynode: true
`)
	baseDir := t.TempDir()
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		return []datagen.StatefulChain{
			{
				IterationID: 0,
				ChainIndex:  0,
				Steps:       []datagen.StatefulStep{{Method: "POST", ResolvedPath: "/owners", Status: 201}},
			},
		}, nil
	}

	if err := runWithGeneratorAndWorkdir(context.Background(), flowPath, "unused", baseDir, 9966, generate, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("failed to list base dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one result subdir, got %d", len(entries))
	}
	name := entries[0].Name()
	if ok, _ := regexp.MatchString(`^result-\d{4}-\d{2}-\d{2}-\d{2}:\d{2}:\d{2}$`, name); !ok {
		t.Fatalf("unexpected result directory name: %s", name)
	}
	stageDir := filepath.Join(baseDir, name, "stage1")
	files := listIterationFiles(t, stageDir)
	if len(files) != 2 {
		t.Fatalf("expected 2 stage iteration files, got %d", len(files))
	}
}

func TestRunWithGenerator_DebugLogsContainStageAndChain(t *testing.T) {
	flowPath := writeTempFlow(t, `
stages:
  stage1:
    wrk2params: -t1 -c1 -d1s -R1
    flow:
      - createOwner:
        operationId: addOwner
        entrynode: true
`)
	outDir := t.TempDir()
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		return []datagen.StatefulChain{
			{
				IterationID: 7,
				ChainIndex:  7,
				Steps: []datagen.StatefulStep{
					{IterationID: 7, Stage: "stage1", OperationID: "addOwner", Method: "POST", ResolvedPath: "/owners", Status: 400, RequestBody: map[string]any{"a": 1}, ResponseBody: map[string]any{"message": "bad input"}},
					{IterationID: 7, Stage: "stage1", OperationID: "addOwner", Method: "POST", ResolvedPath: "/owners", Status: 201, RequestBody: map[string]any{"a": 2}, ResponseBody: map[string]any{"id": 1}},
				},
			},
		}, nil
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	err = runWithGenerator(context.Background(), flowPath, "unused", outDir, 9966, generate, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "stage=stage1") {
		t.Fatalf("expected stage log, got: %s", out)
	}
	if !strings.Contains(out, "chain=addOwner") {
		t.Fatalf("expected chain log, got: %s", out)
	}
}

func TestFilterAcceptedChains_RejectsNon2xxChains(t *testing.T) {
	accepted, stats := filterAcceptedChains([]datagen.StatefulChain{
		{
			ChainIndex: 0,
			Steps: []datagen.StatefulStep{
				{Method: "POST", Status: 201},
				{Method: "GET", Status: 200},
			},
		},
		{
			ChainIndex: 1,
			Steps: []datagen.StatefulStep{
				{Method: "POST", Status: 400},
				{Method: "GET", Status: 200},
			},
		},
	})
	if len(accepted) != 2 {
		t.Fatalf("expected 2 accepted chains after filtering, got %d", len(accepted))
	}
	if len(accepted[1].Steps) != 1 || accepted[1].Steps[0].Status != 200 {
		t.Fatalf("expected non-2xx steps to be filtered out, got %+v", accepted[1].Steps)
	}
	if stats.generatedChains != 2 || stats.acceptedChains != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestFormatDebugPayload_TruncatesLongJSON(t *testing.T) {
	longValue := strings.Repeat("x", maxDebugPayloadChars+200)
	out := formatDebugPayload(map[string]any{"payload": longValue})
	if !strings.Contains(out, debugTruncatedSuffix) {
		t.Fatalf("expected truncation suffix %q in output", debugTruncatedSuffix)
	}
	if len(out) <= maxDebugPayloadChars {
		t.Fatalf("expected output longer than cap due to suffix, got len=%d", len(out))
	}
}

func readIterationFile(t *testing.T, path string) datagen.MinimalIteration {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file %s: %v", path, err)
	}
	var iteration datagen.MinimalIteration
	if err := json.Unmarshal(data, &iteration); err != nil {
		t.Fatalf("invalid output json: %v", err)
	}
	return iteration
}

func listIterationFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to list dir %s: %v", dir, err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "iteration-") && strings.HasSuffix(name, ".json") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files
}

func totalStepCountFromFiles(t *testing.T, dir string, files []string) int {
	t.Helper()
	total := 0
	for _, name := range files {
		iteration := readIterationFile(t, filepath.Join(dir, name))
		total += len(iteration.Steps)
	}
	return total
}

func writeTempFlow(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "flow.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp flow: %v", err)
	}
	return p
}
