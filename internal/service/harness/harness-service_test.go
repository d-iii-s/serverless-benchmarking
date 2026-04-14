package harness

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const latencyArg = "--latency"

func TestBuildWrk2ArgsAddsLatencyWhenMissing(t *testing.T) {
	args := buildWrk2Args("-t2 -c100 -d30s -R2000")
	if !slices.Contains(args, latencyArg) {
		t.Fatalf("expected %s to be injected, got args=%v", latencyArg, args)
	}
	if args[len(args)-1] != latencyArg {
		t.Fatalf("expected %s as trailing arg, got args=%v", latencyArg, args)
	}
}

func TestBuildWrk2ArgsKeepsSingleLatencyWhenAlreadyPresent(t *testing.T) {
	args := buildWrk2Args("-t2 -c100 --latency -d30s -R2000")
	latencyCount := 0
	for _, arg := range args {
		if arg == latencyArg {
			latencyCount++
		}
	}
	if latencyCount != 1 {
		t.Fatalf("expected one %s argument, got %d in args=%v", latencyArg, latencyCount, args)
	}
}

func TestBuildWrk2ArgsEmptyInput(t *testing.T) {
	args := buildWrk2Args("   ")
	if args != nil {
		t.Fatalf("expected nil args for empty wrk2 params, got %v", args)
	}
}

func TestDeriveAPIBasePath_FromYAMLSpec(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yml")
	if err := os.WriteFile(specPath, []byte("servers:\n  - url: http://localhost:9966/petclinic/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := DeriveAPIBasePath(specPath)
	if got != "/petclinic/api" {
		t.Fatalf("expected /petclinic/api, got %q", got)
	}
}

func TestDeriveAPIBasePath_FallbackToSlash(t *testing.T) {
	got := DeriveAPIBasePath("/nonexistent/openapi.yml")
	if got != "/" {
		t.Fatalf("expected /, got %q", got)
	}
}

func TestDeriveAPIBasePath_EmptyServersURL(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yml")
	if err := os.WriteFile(specPath, []byte("servers:\n  - url: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := DeriveAPIBasePath(specPath)
	if got != "/" {
		t.Fatalf("expected /, got %q", got)
	}
}

func TestNormalizeAPIBasePath_Defaults(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/api", "/api"},
		{"api/", "/api"},
		{"/api/v1/", "/api/v1"},
	}
	for _, tc := range cases {
		got := normalizeAPIBasePath(tc.input)
		if got != tc.want {
			t.Errorf("normalizeAPIBasePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
