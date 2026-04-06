package harness

import (
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
