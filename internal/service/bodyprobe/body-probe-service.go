package bodyprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/d-iii-s/slsbench/internal/service/datagen"
	"github.com/d-iii-s/slsbench/internal/service/flowgen"
	"github.com/d-iii-s/slsbench/internal/service/harness"
	utils "github.com/d-iii-s/slsbench/internal/utils"
	"github.com/docker/compose/v5/pkg/api"
	"golang.org/x/sync/errgroup"
)

const (
	defaultBasePathPrefix = "/petclinic/api"
	defaultReadyPath      = "/owners"
	defaultReadyTimeout   = 120 * time.Second
	defaultReadyInterval  = 1 * time.Second
	maxDebugPayloadChars  = 1600
	debugTruncatedSuffix  = "...<truncated>"
)

type generateChainsFn func(ctx context.Context, openAPILink, chain, baseURL string, debug bool) ([]datagen.StatefulChain, error)
type composeStarterFn func(ctx context.Context, dockerComposePath, serviceName string, debug bool) (func(context.Context) error, error)
type readinessWaitFn func(ctx context.Context, url string, timeout, interval time.Duration) error

var (
	startComposeForProbeFn composeStarterFn = startComposeForProbe
	waitForHTTPReadyFn     readinessWaitFn  = waitForHTTPReady
)

type probeStats struct {
	generatedChains int
	acceptedChains  int
	totalSteps      int
	acceptedSteps   int
	retriedSteps    int
}

// Run probes generated request bodies against a running localhost app until
// enough 2xx-accepted bodies are collected for each body-bearing node.
func Run(
	ctx context.Context,
	flowPath, openAPILink, outputPath, dockerComposePath, serviceName string,
	port int,
	debug bool,
	noRewriteLinkedValues bool,
) error {
	return runWithManagedDocker(ctx, dockerComposePath, serviceName, port, debug, func(runCtx context.Context) error {
		generateFn := func(
			generateCtx context.Context,
			generateOpenAPILink, chain, baseURL string,
			generateDebug bool,
		) ([]datagen.StatefulChain, error) {
			return datagen.GenerateStatefulChainsData(
				generateCtx,
				generateOpenAPILink,
				chain,
				baseURL,
				generateDebug,
				noRewriteLinkedValues,
			)
		}
		return runWithGeneratorAndWorkdir(runCtx, flowPath, openAPILink, outputPath, port, generateFn, debug)
	})
}

func runWithManagedDocker(
	ctx context.Context,
	dockerComposePath, serviceName string,
	port int,
	debug bool,
	runFn func(context.Context) error,
) error {
	if strings.TrimSpace(dockerComposePath) == "" {
		return fmt.Errorf("docker compose path must be non-empty")
	}
	if strings.TrimSpace(serviceName) == "" {
		return fmt.Errorf("service name must be non-empty")
	}
	if port <= 0 {
		return fmt.Errorf("port must be positive, got %d", port)
	}
	if runFn == nil {
		return fmt.Errorf("run function must be provided")
	}

	teardown, err := startComposeForProbeFn(ctx, dockerComposePath, serviceName, debug)
	if err != nil {
		return err
	}

	readinessURL := fmt.Sprintf("http://localhost:%d%s%s", port, defaultBasePathPrefix, defaultReadyPath)
	if err := waitForHTTPReadyFn(ctx, readinessURL, defaultReadyTimeout, defaultReadyInterval); err != nil {
		_ = teardown(context.Background())
		return fmt.Errorf("service did not become ready at %s: %w", readinessURL, err)
	}

	runErr := runFn(ctx)
	downErr := teardown(context.Background())
	if runErr != nil {
		if downErr != nil {
			return fmt.Errorf("probe failed: %w; additionally failed to tear down compose project: %v", runErr, downErr)
		}
		return runErr
	}
	if downErr != nil {
		return fmt.Errorf("failed to tear down compose project: %w", downErr)
	}
	return nil
}

func startComposeForProbe(
	ctx context.Context,
	dockerComposePath, serviceName string,
	debug bool,
) (func(context.Context) error, error) {
	composeService, err := harness.NewComposeService()
	if err != nil {
		return nil, err
	}

	projectName := fmt.Sprintf("probe-bodies-%d", time.Now().UnixNano())
	project, err := composeService.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{dockerComposePath},
		ProjectName: projectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load compose project from %q: %w", dockerComposePath, err)
	}
	if len(project.Services) == 0 {
		return nil, fmt.Errorf("compose project %q has no services", projectName)
	}
	if !containsComposeService(project, serviceName) {
		return nil, fmt.Errorf("compose service %q is not defined in %q", serviceName, dockerComposePath)
	}

	if err := composeService.Create(ctx, project, api.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("failed to create compose project %q: %w", projectName, err)
	}
	if err := composeService.Start(ctx, projectName, api.StartOptions{Project: project}); err != nil {
		return nil, fmt.Errorf("failed to start compose project %q: %w", projectName, err)
	}
	if debug {
		fmt.Printf("[probe-bodies] started compose project=%s service=%s\n", projectName, serviceName)
	}

	return func(cleanupCtx context.Context) error {
		if debug {
			fmt.Printf("[probe-bodies] stopping compose project=%s\n", projectName)
		}
		return composeService.Down(cleanupCtx, projectName, api.DownOptions{Project: project})
	}, nil
}

func containsComposeService(project *types.Project, serviceName string) bool {
	for _, svc := range project.Services {
		if svc.Name == serviceName {
			return true
		}
	}
	return false
}

func waitForHTTPReady(ctx context.Context, url string, timeout, interval time.Duration) error {
	if timeout <= 0 {
		timeout = defaultReadyTimeout
	}
	if interval <= 0 {
		interval = defaultReadyInterval
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	tryOnce := func() bool {
		req, err := http.NewRequestWithContext(deadlineCtx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode >= 200 && resp.StatusCode < 500
	}

	if tryOnce() {
		return nil
	}

	for {
		select {
		case <-deadlineCtx.Done():
			return fmt.Errorf("timeout after %s", timeout)
		case <-ticker.C:
			if tryOnce() {
				return nil
			}
		}
	}
}

func runWithGeneratorAndWorkdir(
	ctx context.Context,
	flowPath, openAPILink, outputBasePath string,
	port int,
	generateFn generateChainsFn,
	debug bool,
) error {
	runDir, err := utils.CreateResultSubdir(outputBasePath)
	if err != nil {
		return fmt.Errorf("failed to create probe result directory: %w", err)
	}
	if debug {
		fmt.Printf("[probe-bodies] output run directory: %s\n", runDir)
	}
	return runWithGenerator(ctx, flowPath, openAPILink, runDir, port, generateFn, debug)
}

func runWithGenerator(
	ctx context.Context,
	flowPath, openAPILink, outputPath string,
	port int,
	generateFn generateChainsFn,
	debug bool,
) error {
	if port <= 0 {
		return fmt.Errorf("port must be positive, got %d", port)
	}
	if strings.TrimSpace(flowPath) == "" {
		return fmt.Errorf("flow path must be non-empty")
	}
	dsl, err := flowgen.ParseDSL(flowPath)
	if err != nil {
		return fmt.Errorf("failed to parse flow DSL: %w", err)
	}
	if len(dsl.Stages) == 0 {
		return fmt.Errorf("flow contains no stages")
	}

	baseURL := fmt.Sprintf("http://localhost:%d%s", port, defaultBasePathPrefix)
	stageNames := make([]string, 0, len(dsl.Stages))
	for stageName := range dsl.Stages {
		stageNames = append(stageNames, stageName)
	}
	sort.Strings(stageNames)

	g, ctx := errgroup.WithContext(ctx)
	for _, stageName := range stageNames {
		stageName := stageName
		stage := dsl.Stages[stageName]
		g.Go(func() error {
			return runStageProbe(ctx, stageName, stage, outputPath, openAPILink, baseURL, generateFn, debug)
		})
	}
	return g.Wait()
}

// runStageProbe collects accepted chain iterations for one flow stage. Stages run concurrently
// in runWithGenerator; each stage writes only under outputPath/<stageName>.
func runStageProbe(
	ctx context.Context,
	stageName string,
	stage flowgen.Stage,
	outputPath, openAPILink, baseURL string,
	generateFn generateChainsFn,
	debug bool,
) error {
	cfg, err := flowgen.ParseWrk2Params(stage.Wrk2Params)
	if err != nil {
		return fmt.Errorf("stage %q: invalid wrk2params: %w", stageName, err)
	}
	target := requestTargetWithMargin(cfg.TotalRequests())
	if target <= 0 {
		return fmt.Errorf("stage %q: computed non-positive target %d", stageName, target)
	}
	traverser, err := newStageTraverser(stageName, stage)
	if err != nil {
		return err
	}

	acceptedIterations := make([]datagen.MinimalIteration, 0, target)
	acceptedRequestCount := 0
	attemptLimit := maxInt(target*10, 50)
	attempts := 0
	for acceptedRequestCount < target && attempts < attemptLimit {
		operationIDs, err := traverser.NextChainOperationIDs()
		if err != nil {
			return err
		}
		chain := strings.Join(operationIDs, ",")
		if debug {
			fmt.Printf("[probe-bodies] stage=%s attempt=%d chain=%s\n", stageName, attempts+1, chain)
		}
		generatedChains, err := generateFn(ctx, openAPILink, chain, baseURL, debug)
		attempts++
		if err != nil {
			if debug {
				fmt.Printf("[probe-bodies] stage=%s generation error: %v\n", stageName, err)
			}
			continue
		}
		acceptedChains, stats := filterAcceptedChains(generatedChains)
		if debug {
			logChainsDebug(stageName, acceptedChains, generatedChains, stats)
		}
		for _, chainResult := range acceptedChains {
			for idx := range chainResult.Steps {
				step := &chainResult.Steps[idx]
				if step.Stage == "" {
					step.Stage = stageName
				}
			}
			iterations := datagen.ProjectMinimalIterations([]datagen.StatefulChain{chainResult})
			for _, iteration := range iterations {
				if len(iteration.Steps) == 0 {
					continue
				}
				acceptedIterations = append(acceptedIterations, iteration)
				acceptedRequestCount += len(iteration.Steps)
			}
			if acceptedRequestCount >= target {
				break
			}
		}
	}

	if acceptedRequestCount < target {
		return fmt.Errorf("stage %q: collected %d/%d accepted steps", stageName, acceptedRequestCount, target)
	}

	stageDir := filepath.Join(outputPath, stageName)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create stage output directory %q: %w", stageDir, err)
	}
	if err := writeStageIterationFiles(stageName, stageDir, acceptedIterations); err != nil {
		return err
	}
	return nil
}

func writeStageIterationFiles(stageName, stageDir string, iterations []datagen.MinimalIteration) error {
	for i, iteration := range iterations {
		serialized, err := json.MarshalIndent(iteration, "", "  ")
		if err != nil {
			return fmt.Errorf("stage %q: failed to marshal iteration %d: %w", stageName, i+1, err)
		}
		fileName := fmt.Sprintf("iteration-%06d.json", i+1)
		outputFile := filepath.Join(stageDir, fileName)
		if err := os.WriteFile(outputFile, serialized, 0o644); err != nil {
			return fmt.Errorf("stage %q: failed to write iteration file %q: %w", stageName, outputFile, err)
		}
	}
	return nil
}

func requestTargetWithMargin(total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) * 1.10))
}

type stageTraverser struct {
	stageName string
	nodes     map[string]flowgen.FlowNode
	entryName string
	choosers  map[string]*weightedRoundRobinChooser
	maxDepth  int
}

func newStageTraverser(stageName string, stage flowgen.Stage) (*stageTraverser, error) {
	if len(stage.Flow) == 0 {
		return nil, fmt.Errorf("stage %q: flow is empty", stageName)
	}
	nodes := make(map[string]flowgen.FlowNode, len(stage.Flow))
	entryName := ""
	for _, node := range stage.Flow {
		if node.Name == "" {
			return nil, fmt.Errorf("stage %q: flow node has empty name", stageName)
		}
		nodes[node.Name] = node
		if node.EntryNode {
			if entryName != "" {
				return nil, fmt.Errorf("stage %q: multiple entry nodes are not supported", stageName)
			}
			entryName = node.Name
		}
	}
	if entryName == "" {
		return nil, fmt.Errorf("stage %q: no entry node", stageName)
	}

	choosers := make(map[string]*weightedRoundRobinChooser)
	for name, node := range nodes {
		if len(node.Edges) == 0 {
			continue
		}
		for _, edge := range node.Edges {
			if _, ok := nodes[edge.To]; !ok {
				return nil, fmt.Errorf("stage %q: node %q has edge to unknown node %q", stageName, name, edge.To)
			}
		}
		chooser, err := newWeightedRoundRobinChooser(node.Edges)
		if err != nil {
			return nil, fmt.Errorf("stage %q node %q: %w", stageName, name, err)
		}
		choosers[name] = chooser
	}
	return &stageTraverser{
		stageName: stageName,
		nodes:     nodes,
		entryName: entryName,
		choosers:  choosers,
		maxDepth:  len(nodes)*4 + 4,
	}, nil
}

func (t *stageTraverser) NextChainOperationIDs() ([]string, error) {
	current := t.entryName
	ops := make([]string, 0, t.maxDepth)
	for depth := 0; depth < t.maxDepth; depth++ {
		node, ok := t.nodes[current]
		if !ok {
			return nil, fmt.Errorf("stage %q: traversal reached unknown node %q", t.stageName, current)
		}
		if strings.TrimSpace(node.OperationID) == "" {
			return nil, fmt.Errorf("stage %q: node %q missing operationId", t.stageName, node.Name)
		}
		ops = append(ops, node.OperationID)
		if len(node.Edges) == 0 {
			return ops, nil
		}
		chooser := t.choosers[node.Name]
		if chooser == nil {
			return nil, fmt.Errorf("stage %q: missing chooser for node %q", t.stageName, node.Name)
		}
		current = chooser.Next()
	}
	return nil, fmt.Errorf("stage %q: traversal exceeded max depth %d (possible cycle)", t.stageName, t.maxDepth)
}

type weightedRoundRobinChooser struct {
	edges   []flowgen.Edge
	current []float64
	total   float64
}

func newWeightedRoundRobinChooser(edges []flowgen.Edge) (*weightedRoundRobinChooser, error) {
	if len(edges) == 0 {
		return nil, fmt.Errorf("cannot build weighted chooser with no edges")
	}
	total := 0.0
	for _, edge := range edges {
		if edge.Weight <= 0 {
			return nil, fmt.Errorf("edge to %q has non-positive weight %v", edge.To, edge.Weight)
		}
		total += edge.Weight
	}
	return &weightedRoundRobinChooser{
		edges:   edges,
		current: make([]float64, len(edges)),
		total:   total,
	}, nil
}

func (w *weightedRoundRobinChooser) Next() string {
	best := 0
	for i := range w.edges {
		w.current[i] += w.edges[i].Weight
		if w.current[i] > w.current[best] {
			best = i
		}
	}
	w.current[best] -= w.total
	return w.edges[best].To
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
