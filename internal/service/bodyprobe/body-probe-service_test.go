package bodyprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/d-iii-s/slsbench/internal/service/datagen"
)

func TestRunWithGenerator_DoesNotWriteIterationsWhenDebugDisabled(t *testing.T) {
	expectedChain := "createOwner"
	outDir := t.TempDir()

	generate := func(ctx context.Context, openAPILink, chain, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		if chain != expectedChain {
			t.Fatalf("unexpected chain %q", chain)
		}
		if debug {
			t.Fatalf("expected debug=false in generator call")
		}
		return []datagen.StatefulChain{
			{
				IterationID: 0,
				ChainIndex:  0,
				Steps: []datagen.StatefulStep{
					{Method: "POST", ResolvedPath: "/owners", Status: 201, RequestBody: map[string]any{"firstName": "A"}},
				},
			},
			{
				IterationID: 1,
				ChainIndex:  1,
				Steps: []datagen.StatefulStep{
					{Method: "POST", ResolvedPath: "/owners", Status: 201, RequestBody: map[string]any{"firstName": "B"}},
				},
			},
		}, nil
	}

	if err := runWithGenerator(context.Background(), expectedChain, "unused", outDir, "svc", 9966, generate, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "iterations.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no iterations file when debug=false, stat err=%v", err)
	}
}

func TestRunWithGenerator_WritesMinimalIterationsWhenDebugEnabled(t *testing.T) {
	chain := "createOwner"
	outDir := t.TempDir()
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		if chainArg != chain {
			t.Fatalf("expected chain=%q in generator call", chain)
		}
		if !debug {
			t.Fatalf("expected debug=true in generator call")
		}
		return []datagen.StatefulChain{
			{
				IterationID: 1,
				ChainIndex:  1,
				Steps: []datagen.StatefulStep{
					{
						FlowID:       "stage1/createOwner",
						Method:       "POST",
						Headers:      map[string]any{"content-type": "application/json"},
						Query:        map[string]any{"q": "abc"},
						ResolvedPath: "/owners",
						Status:       201,
						RequestBody:  map[string]any{"ownerId": "/id"},
						ResponseBody: map[string]any{"id": 10},
					},
				},
			},
		}, nil
	}

	if err := runWithGenerator(context.Background(), chain, "unused", outDir, "svc", 9966, generate, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "iterations.json"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	var iterations []datagen.MinimalIteration
	if err := json.Unmarshal(data, &iterations); err != nil {
		t.Fatalf("invalid output json: %v", err)
	}
	if len(iterations) != 1 || len(iterations[0].Steps) != 1 {
		t.Fatalf("unexpected minimal iterations shape: %+v", iterations)
	}
	step := iterations[0].Steps[0]
	if step.FlowID != "stage1/createOwner" {
		t.Fatalf("expected flowId in minimal output, got %q", step.FlowID)
	}
	if step.Method != "POST" {
		t.Fatalf("expected method in minimal output, got %q", step.Method)
	}
	if step.ResolvedPath != "/owners" {
		t.Fatalf("expected resolvedPath in minimal output, got %q", step.ResolvedPath)
	}
	if step.Headers["content-type"] != "application/json" {
		t.Fatalf("expected headers in minimal output, got %#v", step.Headers)
	}
	if step.Query["q"] != "abc" {
		t.Fatalf("expected query in minimal output, got %#v", step.Query)
	}
	body, ok := step.RequestBody.(map[string]any)
	if !ok || body["ownerId"] != "/id" {
		t.Fatalf("expected json pointer request body in minimal output, got: %#v", step.RequestBody)
	}
}

func TestRunWithGenerator_RequiresChain(t *testing.T) {
	outDir := t.TempDir()
	generate := func(ctx context.Context, openAPILink, chainArg, baseURL string, debug bool) ([]datagen.StatefulChain, error) {
		return []datagen.StatefulChain{
			{
				IterationID: 0,
				ChainIndex:  0,
				Steps:       []datagen.StatefulStep{{Method: "POST", ResolvedPath: "/owners", Status: 201}},
			},
		}, nil
	}

	err := runWithGenerator(context.Background(), " ", "unused", outDir, "svc", 9966, generate, false)
	if err == nil {
		t.Fatal("expected error for empty chain")
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

func TestRunWithGeneratorAndWorkdir_CreatesResultSubdir(t *testing.T) {
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

	if err := runWithGeneratorAndWorkdir(context.Background(), "createOwner", "unused", baseDir, "svc", 9966, generate, false); err != nil {
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
	outPath := filepath.Join(baseDir, name, "iterations.json")
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("expected no iterations file when debug=false, stat err: %v", err)
	}
}

func TestRunWithGenerator_DebugLogsContainChainStepFields(t *testing.T) {
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
	defer func() {
		os.Stdout = origStdout
	}()

	err = runWithGenerator(context.Background(), "createOwner", "unused", outDir, "svc", 9966, generate, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[probe-bodies] retry") || !strings.Contains(out, "status=400") {
		t.Fatalf("expected debug retry/status log, got: %s", out)
	}
	if !strings.Contains(out, "chainIndex=7") || !strings.Contains(out, "stepIndex=0") {
		t.Fatalf("expected chain step identifiers in log, got: %s", out)
	}
	if !strings.Contains(out, "request=") {
		t.Fatalf("expected debug request payload log, got: %s", out)
	}
	if !strings.Contains(out, "response=") {
		t.Fatalf("expected debug response payload log, got: %s", out)
	}
	if !strings.Contains(out, "message") {
		t.Fatalf("expected failure response body message in logs, got: %s", out)
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
