package datagen

import (
	"context"
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
	// --- input validation ---
	if count <= 0 {
		return fmt.Errorf("count must be a positive integer, got %d", count)
	}
	if _, err := os.Stat(specPath); err != nil {
		return fmt.Errorf("OpenAPI spec file not found: %w", err)
	}

	// --- locate the Python script ---
	scriptPath, err := resolveScriptPath()
	if err != nil {
		return fmt.Errorf("failed to locate generate_bodies.py: %w", err)
	}

	// --- build and run the command ---
	args := []string{
		scriptPath,
		"--spec-path", specPath,
		"--endpoint", endpoint,
		"--method", strings.ToUpper(method),
		"--count", fmt.Sprintf("%d", count),
		"--output", outputPath,
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	/*
		we should compute map endpoint -> number of bodies
		mount value structure:
		/stage1
		DSL: would be -> endpoint -> endpoint

	*/
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("generate_bodies.py failed (exit %v): %s", err, string(output))
	}

	return nil
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
