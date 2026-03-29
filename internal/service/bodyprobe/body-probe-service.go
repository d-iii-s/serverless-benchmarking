package bodyprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/d-iii-s/slsbench/internal/service/datagen"
	workdirsvc "github.com/d-iii-s/slsbench/internal/service/workdir"
)

const (
	defaultBasePathPrefix = "/petclinic/api"
	maxDebugPayloadChars  = 1600
	debugTruncatedSuffix  = "...<truncated>"
)

type generateChainsFn func(ctx context.Context, openAPILink, chain, baseURL string, debug bool) ([]datagen.StatefulChain, error)

type probeStats struct {
	generatedChains int
	acceptedChains  int
	totalSteps      int
	acceptedSteps   int
	retriedSteps    int
}

// Run probes generated request bodies against a running localhost app until
// enough 2xx-accepted bodies are collected for each body-bearing node.
func Run(ctx context.Context, chain, openAPILink, outputPath, dockerComposePath, serviceName string, port int, debug bool) error {
	return runWithGeneratorAndWorkdir(ctx, chain, openAPILink, outputPath, serviceName, port, datagen.GenerateStatefulChainsData, debug)
}

func runWithGeneratorAndWorkdir(
	ctx context.Context,
	chain, openAPILink, outputBasePath, serviceName string,
	port int,
	generateFn generateChainsFn,
	debug bool,
) error {
	runDir, err := workdirsvc.CreateResultSubdir(outputBasePath)
	if err != nil {
		return fmt.Errorf("failed to create probe result directory: %w", err)
	}
	if debug {
		fmt.Printf("[probe-bodies] output run directory: %s\n", runDir)
	}
	return runWithGenerator(ctx, chain, openAPILink, runDir, serviceName, port, generateFn, debug)
}

func runWithGenerator(
	ctx context.Context,
	chain, openAPILink, outputPath, serviceName string,
	port int,
	generateFn generateChainsFn,
	debug bool,
) error {
	if port <= 0 {
		return fmt.Errorf("port must be positive, got %d", port)
	}
	if strings.TrimSpace(chain) == "" {
		return fmt.Errorf("chain must be non-empty")
	}
	baseURL := fmt.Sprintf("http://localhost:%d%s", port, defaultBasePathPrefix)
	generatedChains, err := generateFn(ctx, openAPILink, chain, baseURL, debug)
	if err != nil {
		return fmt.Errorf("failed to generate stateful chains: %w", err)
	}
	acceptedChains, stats := filterAcceptedChains(generatedChains)
	if len(acceptedChains) == 0 {
		return fmt.Errorf("no 2xx-only stateful chains generated")
	}
	if debug {
		logChainsDebug("full-flow", acceptedChains, generatedChains, stats)
		if err := os.MkdirAll(outputPath, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory %q: %w", outputPath, err)
		}
		outputFile := fmt.Sprintf("%s/iterations.json", outputPath)
		serialized, err := json.MarshalIndent(datagen.ProjectMinimalIterations(acceptedChains), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal accepted chains: %w", err)
		}
		if err := os.WriteFile(outputFile, serialized, 0o644); err != nil {
			return fmt.Errorf("failed to write chain file %q: %w", outputFile, err)
		}
	}

	_ = serviceName // accepted by command signature and logs in CLI
	return nil
}

func filterAcceptedChains(generated []datagen.StatefulChain) ([]datagen.StatefulChain, probeStats) {
	accepted := make([]datagen.StatefulChain, 0, len(generated))
	stats := probeStats{}
	stats.generatedChains = len(generated)
	for _, chain := range generated {
		if len(chain.Steps) == 0 {
			continue
		}
		filtered := make([]datagen.StatefulStep, 0, len(chain.Steps))
		for _, step := range chain.Steps {
			stats.totalSteps++
			if step.Status < 200 || step.Status >= 300 {
				stats.retriedSteps++
			} else {
				stats.acceptedSteps++
				filtered = append(filtered, step)
			}
		}
		if len(filtered) > 0 {
			chain.Steps = filtered
			accepted = append(accepted, chain)
			stats.acceptedChains++
		}
	}
	return accepted, stats
}

func logChainsDebug(
	stageName string,
	accepted []datagen.StatefulChain,
	generated []datagen.StatefulChain,
	stats probeStats,
) {
	for _, chain := range generated {
		for stepIdx, step := range chain.Steps {
			prefix := "[probe-bodies] retry"
			if step.Status >= 200 && step.Status < 300 {
				prefix = "[probe-bodies] accepted"
			}
			fmt.Printf(
				"%s stage=%s iterationIndex=%d chainIndex=%d stepIndex=%d operationId=%s method=%s path=%s status=%d request=%s response=%s\n",
				prefix,
				stageName,
				chain.IterationID,
				chain.ChainIndex,
				stepIdx,
				step.OperationID,
				strings.ToUpper(step.Method),
				step.ResolvedPath,
				step.Status,
				formatDebugPayload(step.RequestBody),
				formatDebugPayload(step.ResponseBody),
			)
		}
	}
	fmt.Printf(
		"[probe-bodies] stage=%s generatedChains=%d acceptedChains=%d totalSteps=%d acceptedSteps=%d retriedSteps=%d\n",
		stageName, stats.generatedChains, stats.acceptedChains, stats.totalSteps, stats.acceptedSteps, stats.retriedSteps,
	)
	fmt.Printf("[probe-bodies] stage=%s persistedChains=%d\n", stageName, len(accepted))
}

func formatDebugPayload(payload any) string {
	if payload == nil {
		return "null"
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(fmt.Sprintf("%v", payload))
	}
	text := string(encoded)
	if len(text) <= maxDebugPayloadChars {
		return text
	}
	return text[:maxDebugPayloadChars] + debugTruncatedSuffix
}
