package datagen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// todo: generate_bodies should take instruction in which order generate bodies.

// scriptRelPath is the path from the project root to the Python helper script.
const scriptRelPath = "scripts/generate_bodies.py"

type StatefulStep struct {
	IterationID  int            `json:"iterationIndex"`
	Stage        string         `json:"stage,omitempty"`
	FlowID       string         `json:"flowId,omitempty"`
	OperationID  string         `json:"operationId,omitempty"`
	Method       string         `json:"method"`
	PathTemplate string         `json:"pathTemplate"`
	ResolvedPath string         `json:"resolvedPath"`
	PathParams   map[string]any `json:"pathParameters,omitempty"`
	Query        map[string]any `json:"query,omitempty"`
	Headers      map[string]any `json:"headers,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty"`
	Status       int            `json:"status"`
	ResponseBody any            `json:"responseBody,omitempty"`
}

type StatefulChain struct {
	IterationID int            `json:"iterationIndex"`
	ChainIndex  int            `json:"chainIndex"`
	Steps       []StatefulStep `json:"steps"`
}

type MinimalIterationStep struct {
	FlowID       string         `json:"flowId"`
	Method       string         `json:"method"`
	Headers      map[string]any `json:"headers"`
	Query        map[string]any `json:"query"`
	ResolvedPath string         `json:"resolvedPath"`
	RequestBody  any            `json:"requestBody"`
}

type MinimalIteration struct {
	IterationID int                    `json:"iterationIndex"`
	Steps       []MinimalIterationStep `json:"steps"`
}

func ProjectMinimalIterations(chains []StatefulChain) []MinimalIteration {
	minimal := make([]MinimalIteration, 0, len(chains))
	for _, chain := range chains {
		steps := make([]MinimalIterationStep, 0, len(chain.Steps))
		for _, step := range chain.Steps {
			headers := step.Headers
			if headers == nil {
				headers = map[string]any{}
			}
			query := step.Query
			if query == nil {
				query = map[string]any{}
			}
			steps = append(steps, MinimalIterationStep{
				FlowID:       step.FlowID,
				Method:       step.Method,
				Headers:      headers,
				Query:        query,
				ResolvedPath: step.ResolvedPath,
				RequestBody:  step.RequestBody,
			})
		}
		minimal = append(minimal, MinimalIteration{
			IterationID: chain.IterationID,
			Steps:       steps,
		})
	}
	return minimal
}

// GenerateRequestBodies invokes the Schemathesis-based Python script to
// generate random request bodies for the given OpenAPI endpoint and writes
// the result (a JSON array) to outputPath.
//
// Parameters:
//   - specPath:   path to the OpenAPI specification file (YAML or JSON)
//   - endpoint:   API endpoint path, e.g. "/pets"
//   - method:     HTTP method, e.g. "POST"
//   - count:      number of request bodies to generate (must be > 0)
//   - outputPath: file path where the JSON array of bodies will be written
func GenerateRequestBodies(ctx context.Context, specPath, endpoint, method string, count int, outputPath string) error {
	bodies, err := GenerateRequestBodiesData(ctx, specPath, endpoint, method, count)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(bodies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal generated bodies: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write generated bodies to %q: %w", outputPath, err)
	}
	return nil
}

// GenerateRequestBodiesData invokes the Schemathesis-based Python script and
// returns generated request bodies as in-memory objects.
func GenerateRequestBodiesData(ctx context.Context, openAPILink, endpoint, method string, count int) ([]map[string]any, error) {
	_ = ctx
	_ = openAPILink
	_ = endpoint
	_ = method
	_ = count
	return nil, fmt.Errorf("per-operation body generation is removed; use GenerateStatefulChainsData")
}

// GenerateStatefulChainsData runs Schemathesis stateful mode and returns
// link-driven execution chains.
func GenerateStatefulChainsData(
	ctx context.Context,
	openAPILink string,
	chain string,
	baseURL string,
	debug bool,
) ([]StatefulChain, error) {
	if !isRemoteOpenAPILink(openAPILink) {
		if _, err := os.Stat(openAPILink); err != nil {
			return nil, fmt.Errorf("OpenAPI spec file not found: %w", err)
		}
	}
	if strings.TrimSpace(chain) == "" {
		return nil, fmt.Errorf("chain is required")
	}

	scriptPath, err := resolveScriptPath()
	if err != nil {
		return nil, fmt.Errorf("failed to locate generate_bodies.py: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "slsbench-chains-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	args := []string{
		scriptPath,
		"--openapi-link", openAPILink,
		"--chain", chain,
		"--output", tmpPath,
		"--base-url", baseURL,
	}
	if debug {
		args = append(args, "--debug")
	}
	cmd := exec.CommandContext(ctx, "python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("generate_bodies.py stateful failed (exit %v): %s", err, string(output))
	}
	if debug && len(output) > 0 {
		// Forward python debug logs (stderr captured in CombinedOutput).
		_, _ = fmt.Fprint(os.Stderr, string(output))
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated chains temp file: %w", err)
	}
	var chains []StatefulChain
	if err := json.Unmarshal(data, &chains); err != nil {
		return nil, fmt.Errorf("failed to parse generated chains JSON: %w", err)
	}
	return chains, nil
}

// resolveScriptPath tries to find the Python helper script relative to the
// Go source file (works during `go test`) and then relative to the working
// directory (works when running the built binary from the project root).
func resolveScriptPath() (string, error) {
	// 1. Try relative to this Go source file (useful during `go test`).
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		// thisFile is .../internal/service/datagen/datagen.go
		// project root is three directories up.
		projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
		candidate := filepath.Join(projectRoot, scriptRelPath)
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs, nil
		}
	}

	// 2. Try relative to the current working directory.
	if _, err := os.Stat(scriptRelPath); err == nil {
		abs, _ := filepath.Abs(scriptRelPath)
		return abs, nil
	}

	// 3. Allow override via environment variable.
	if envPath := os.Getenv("SLSBENCH_SCRIPT_DIR"); envPath != "" {
		candidate := filepath.Join(envPath, "generate_bodies.py")
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find %s in any known location", scriptRelPath)
}

func isRemoteOpenAPILink(v string) bool {
	lower := strings.ToLower(v)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
