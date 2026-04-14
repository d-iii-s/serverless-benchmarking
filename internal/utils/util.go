package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DumpJSON returns a JSON-formatted string representation of any struct.
func DumpJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling to JSON: %v", err)
	}
	return string(data)
}

// PrintJSON prints any struct in a readable JSON format.
func PrintJSON(v any) {
	fmt.Println(DumpJSON(v))
}

func CreateResultSubdir(path string) (string, error) {
	return CreateResultSubdirWithPrefix(path, "result")
}

func CreateResultSubdirWithPrefix(path, prefix string) (string, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("failed to create base directory %q: %w", path, err)
	}
	if prefix == "" {
		prefix = "result"
	}

	dirName := fmt.Sprintf("%s-%s", prefix, time.Now().Format("2006-01-02-15:04:05"))
	resultDir := filepath.Join(path, dirName)

	if err := os.Mkdir(resultDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create result subdirectory %q: %w", resultDir, err)
	}

	return resultDir, nil
}
