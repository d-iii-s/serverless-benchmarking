package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/d-iii-s/slsbench/internal/service/datagen"
	"github.com/d-iii-s/slsbench/internal/service/docker"
	"github.com/d-iii-s/slsbench/internal/service/flowgen"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	dockertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

const (
	defaultFirstResponseTimeout  = 120 * time.Second
	defaultFirstResponseInterval = 200 * time.Millisecond
	defaultAPIBasePathPrefix     = "/petclinic/api"
	wrkFlowImage                 = "aape2k/wrk2-flow:v3.0"
)

type EventProcessor struct{}

func newEventProcessor() EventProcessor {
	return EventProcessor{}
}

func (e EventProcessor) Start(_ context.Context, operation string) {
	log.Printf("Starting operation: %s\n", operation)
}

func (e EventProcessor) On(events ...api.Resource) {
	for _, event := range events {
		log.Printf("Resource ID: %s, Status: %v, Details: %s, Progress: %d/%d (%d%%)\n",
			event.ID, event.Status, event.Details, event.Current, event.Total, event.Percent)
	}
}

func (e EventProcessor) Done(operation string, success bool) {
	if success {
		log.Printf("Operation %s completed successfully\n", operation)
	} else {
		log.Printf("Operation %s failed\n", operation)
	}
}

// NewComposeService creates and initializes a Docker Compose service client.
func NewComposeService() (api.Compose, error) {
	return NewComposeServiceWithDockerHost("")
}

// NewComposeServiceWithDockerHost creates a Docker Compose service and optionally
// points it to a custom docker host (for example unix:///var/run/docker.sock).
func NewComposeServiceWithDockerHost(dockerHost string) (api.Compose, error) {
	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("error creating docker CLI: %w", err)
	}

	clientOptions := &flags.ClientOptions{}
	if strings.TrimSpace(dockerHost) != "" {
		clientOptions.Hosts = []string{dockerHost}
	}
	if err := dockerCLI.Initialize(clientOptions); err != nil {
		return nil, fmt.Errorf("failed to initialize docker CLI: %w", err)
	}

	composeService, err := compose.NewComposeService(dockerCLI, compose.WithEventProcessor(newEventProcessor()))
	if err != nil {
		return nil, fmt.Errorf("error creating compose service: %w", err)
	}

	return composeService, nil
}

func NewDockerClientWithSocket(socketPath string) (*client.Client, string, error) {
	host := DockerHostFromSocketPath(socketPath)
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create docker client: %w", err)
	}
	return cli, host, nil
}

type firstResponseResult struct {
	TargetURL        string    `json:"targetUrl"`
	StartedAt        time.Time `json:"startedAt"`
	FinishedAt       time.Time `json:"finishedAt"`
	DurationSeconds  float64   `json:"durationSeconds"`
	DurationMillis   int64     `json:"durationMillis"`
	Attempts         int       `json:"attempts"`
	StatusCode       int       `json:"statusCode"`
	ResolvedPathUsed string    `json:"resolvedPathUsed"`
}

type benchmarkContainerStatsSample struct {
	TimestampUTC     time.Time `json:"timestampUtc"`
	ContainerID      string    `json:"containerId"`
	ContainerName    string    `json:"containerName,omitempty"`
	CPUPercent       float64   `json:"cpuPercent"`
	MemoryUsageBytes uint64    `json:"memoryUsageBytes"`
	MemoryLimitBytes uint64    `json:"memoryLimitBytes"`
	MemoryPercent    float64   `json:"memoryPercent"`
	NetworkRxBytes   uint64    `json:"networkRxBytes"`
	NetworkTxBytes   uint64    `json:"networkTxBytes"`
	Pids             uint64    `json:"pids"`
	NumCPUs          uint32    `json:"numCpus"`
	OnlineCPUs       uint32    `json:"onlineCpus"`
	SystemUsage      uint64    `json:"systemUsage"`
	TotalCPUUsage    uint64    `json:"totalCpuUsage"`
	PreSystemUsage   uint64    `json:"preSystemUsage"`
	PreTotalCPUUsage uint64    `json:"preTotalCpuUsage"`
}

func Run(
	ctx context.Context,
	flowPath, resultPath, openAPISpecPath, dockerComposePath, serviceName string,
	port int,
	serviceMountPaths []string,
	probeBodiesPath, dockerSocketPath string,
	debugNon2xx bool,
) error {
	if strings.TrimSpace(flowPath) == "" {
		return fmt.Errorf("flow path must be non-empty")
	}
	if strings.TrimSpace(resultPath) == "" {
		return fmt.Errorf("result path must be non-empty")
	}
	if strings.TrimSpace(openAPISpecPath) == "" {
		return fmt.Errorf("openapi spec path must be non-empty")
	}
	if strings.TrimSpace(dockerComposePath) == "" {
		return fmt.Errorf("docker compose path must be non-empty")
	}
	if strings.TrimSpace(serviceName) == "" {
		return fmt.Errorf("service name must be non-empty")
	}
	if strings.TrimSpace(probeBodiesPath) == "" {
		return fmt.Errorf("probe-bodies path must be non-empty")
	}
	if port <= 0 {
		return fmt.Errorf("port must be positive, got %d", port)
	}
	if err := validateReadableFile(flowPath); err != nil {
		return fmt.Errorf("invalid flow path: %w", err)
	}
	if err := validateReadableFile(openAPISpecPath); err != nil {
		return fmt.Errorf("invalid openapi spec path: %w", err)
	}
	if err := validateReadableFile(dockerComposePath); err != nil {
		return fmt.Errorf("invalid docker compose path: %w", err)
	}
	if err := validateReadableDir(probeBodiesPath); err != nil {
		return fmt.Errorf("invalid probe-bodies path: %w", err)
	}
	if strings.TrimSpace(dockerSocketPath) != "" {
		if err := validateReadableFile(dockerSocketPath); err != nil {
			return fmt.Errorf("invalid docker socket path: %w", err)
		}
	}

	runDir, err := utils.CreateResultSubdirWithPrefix(resultPath, "harness-result")
	if err != nil {
		return fmt.Errorf("failed to create result directory: %w", err)
	}
	log.Printf("Harness output run directory: %s", runDir)

	dockerCli, dockerHost, err := NewDockerClientWithSocket(dockerSocketPath)
	if err != nil {
		return err
	}
	defer dockerCli.Close()

	composeService, err := NewComposeServiceWithDockerHost(dockerHost)
	if err != nil {
		return err
	}

	projectName := fmt.Sprintf("harness-%d", time.Now().UnixNano())
	log.Printf("[harness][compose] context project=%s composePath=%s service=%s dockerHost=%s", projectName, dockerComposePath, serviceName, dockerHost)

	loadStartedAt := time.Now()
	log.Printf("[harness][compose] phase=load begin project=%s", projectName)
	project, err := composeService.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{dockerComposePath},
		ProjectName: projectName,
	})
	if err != nil {
		log.Printf("[harness][compose] phase=load failed project=%s elapsed=%s error=%v", projectName, time.Since(loadStartedAt), err)
		return fmt.Errorf("failed to load compose project: %w", err)
	}
	log.Printf("[harness][compose] phase=load done project=%s elapsed=%s services=%d", projectName, time.Since(loadStartedAt), len(project.Services))
	if len(project.Services) == 0 {
		return fmt.Errorf("compose project %q has no services", projectName)
	}
	if !containsComposeService(project, serviceName) {
		return fmt.Errorf("service %q is not present in compose file %q", serviceName, dockerComposePath)
	}

	createStartedAt := time.Now()
	log.Printf("[harness][compose] phase=create begin project=%s", projectName)
	if err := composeService.Create(ctx, project, api.CreateOptions{Build: &api.BuildOptions{}}); err != nil {
		log.Printf("[harness][compose] phase=create failed project=%s elapsed=%s error=%v", projectName, time.Since(createStartedAt), err)
		return fmt.Errorf("failed to create compose resources: %w", err)
	}
	log.Printf("[harness][compose] phase=create done project=%s elapsed=%s", projectName, time.Since(createStartedAt))

	startStartedAt := time.Now()
	log.Printf("[harness][compose] phase=start begin project=%s", projectName)
	if err := composeService.Start(ctx, projectName, api.StartOptions{Project: project}); err != nil {
		log.Printf("[harness][compose] phase=start failed project=%s elapsed=%s error=%v", projectName, time.Since(startStartedAt), err)
		return fmt.Errorf("failed to start compose resources: %w", err)
	}
	log.Printf("[harness][compose] phase=start done project=%s elapsed=%s", projectName, time.Since(startStartedAt))

	var runErr error
	defer func() {
		downStartedAt := time.Now()
		reason := "success"
		if runErr != nil {
			reason = "failure"
		}
		log.Printf("[harness][compose] phase=down begin project=%s reason=%s", projectName, reason)
		downErr := composeService.Down(context.Background(), projectName, api.DownOptions{Project: project})
		if downErr != nil {
			log.Printf("[harness][compose] phase=down failed project=%s elapsed=%s error=%v", projectName, time.Since(downStartedAt), downErr)
		} else {
			log.Printf("[harness][compose] phase=down done project=%s elapsed=%s", projectName, time.Since(downStartedAt))
		}
		if downErr != nil && runErr == nil {
			runErr = fmt.Errorf("failed to tear down compose project: %w", downErr)
		}
	}()

	networkName := getProjectNetworkName(project)
	log.Printf("[harness][compose] resolved network project=%s network=%s", projectName, networkName)
	serviceContainerID, err := findContainerIDByServiceName(ctx, dockerCli, projectName, serviceName)
	if err != nil {
		return fmt.Errorf("failed to find service container id: %w", err)
	}
	statsOutputPath := filepath.Join(runDir, "benchmark-container-stats.jsonl")
	log.Printf("[harness][stats] streaming begin container=%s output=%s", serviceContainerID, statsOutputPath)
	stopStatsCollector, err := startBenchmarkContainerStatsCollector(ctx, dockerCli, serviceContainerID, statsOutputPath)
	if err != nil {
		return fmt.Errorf("failed to start benchmark container stats collector: %w", err)
	}
	defer func() {
		if stopErr := stopStatsCollector(); stopErr != nil {
			log.Printf("[harness][stats] streaming failed container=%s error=%v", serviceContainerID, stopErr)
			if runErr == nil {
				runErr = fmt.Errorf("failed to finalize benchmark container stats collector: %w", stopErr)
			}
			return
		}
		log.Printf("[harness][stats] streaming done container=%s output=%s", serviceContainerID, statsOutputPath)
	}()

	dsl, err := flowgen.ParseDSL(flowPath)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}
	if len(dsl.Stages) == 0 {
		return fmt.Errorf("flow has no stages")
	}

	apiBasePath := deriveAPIBasePath(openAPISpecPath)
	firstPath := firstResolvedPathFromProbeData(probeBodiesPath)
	firstResult, err := measureFirstResponse(ctx, serviceName, port, apiBasePath, firstPath)
	if err != nil {
		return fmt.Errorf("failed to measure first response: %w", err)
	}
	if err := writeJSON(filepath.Join(runDir, "first_request_result.json"), firstResult); err != nil {
		return fmt.Errorf("failed to write first request result: %w", err)
	}

	stageNames := sortedStageNames(dsl)
	for _, stageName := range stageNames {
		stage := dsl.Stages[stageName]
		if _, err := flowgen.ParseWrk2Params(stage.Wrk2Params); err != nil {
			return fmt.Errorf("stage %q has invalid wrk2params: %w", stageName, err)
		}
		stageIterations, err := loadStageIterations(probeBodiesPath, stageName)
		if err != nil {
			return err
		}
		stageRoot := filepath.Join(runDir, "wrk2-input", sanitizePathPart(stageName))
		stageDataDir := filepath.Join(stageRoot, stageName)
		if err := os.MkdirAll(stageDataDir, 0o755); err != nil {
			return fmt.Errorf("failed to create stage data directory: %w", err)
		}
		if err := writeIterations(stageDataDir, stageIterations); err != nil {
			return fmt.Errorf("failed to write stage iterations for stage=%s: %w", stageName, err)
		}

		stageOutputDir := filepath.Join(runDir, "wrk2-results", sanitizePathPart(stageName))
		if err := os.MkdirAll(stageOutputDir, 0o755); err != nil {
			return fmt.Errorf("failed to create stage output directory: %w", err)
		}
		log.Printf("Starting wrk2-flow run for stage=%s", stageName)
		log.Printf("Stage wrk2 debug mode stage=%s flowDebugNon2xx=%t", stageName, debugNon2xx)
		if err := runWrk2FlowContainer(
			ctx,
			dockerCli,
			networkName,
			stage.Wrk2Params,
			stageName,
			serviceName,
			port,
			stageRoot,
			stageOutputDir,
			debugNon2xx,
		); err != nil {
			return err
		}
		log.Printf("Completed wrk2-flow run for stage=%s", stageName)
	}

	for _, serviceMountPath := range serviceMountPaths {
		serviceMountPath = strings.TrimSpace(serviceMountPath)
		if serviceMountPath == "" {
			continue
		}
		if err := docker.CopyFromContainer(ctx, dockerCli, serviceContainerID, serviceMountPath, runDir); err != nil {
			log.Printf("Warning: failed to copy %q from service container: %v", serviceMountPath, err)
		}
	}

	runErr = nil
	return nil
}

func runWrk2FlowContainer(
	ctx context.Context,
	dockerCli *client.Client,
	networkName, wrk2Params, stageName, serviceName string,
	port int,
	dataRootPath, outputPath string,
	debugNon2xx bool,
) error {
	args := buildWrk2Args(wrk2Params)
	if len(args) == 0 {
		return fmt.Errorf("empty wrk2 params for stage %q", stageName)
	}
	args = append(args, fmt.Sprintf("http://%s:%d/", serviceName, port))

	containerConfig := &dockertypes.Config{
		Image: wrkFlowImage,
		Env: []string{
			"FLOW_EXECUTOR_MODE=templates",
			"FLOW_DATA_DIR=/flowdata",
			fmt.Sprintf("FLOW_STAGE=%s", stageName),
			"FLOW_STATS_OUT_DIR=/stats",
			fmt.Sprintf("FLOW_DEBUG_NON2XX=%d", boolToInt(debugNon2xx)),
		},
		Cmd: args,
	}
	hostConfig := &dockertypes.HostConfig{
		NetworkMode: dockertypes.NetworkMode(networkName),
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   dataRootPath,
				Target:   "/flowdata",
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: outputPath,
				Target: "/stats",
			},
		},
	}

	containerName := fmt.Sprintf("harness-%s-%d", sanitizePathPart(stageName), time.Now().UnixNano())
	resp, err := dockerCli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create wrk2 container for stage=%s: %w", stageName, err)
	}

	defer func() {
		_ = dockerCli.ContainerRemove(context.Background(), resp.ID, dockertypes.RemoveOptions{Force: true})
	}()

	if err := dockerCli.ContainerStart(ctx, resp.ID, dockertypes.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start wrk2 container for stage=%s: %w", stageName, err)
	}

	statusCh, errCh := dockerCli.ContainerWait(ctx, resp.ID, dockertypes.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("wrk2 container wait failed for stage=%s: %w", stageName, waitErr)
		}
	case status := <-statusCh:
		if status.Error != nil {
			return fmt.Errorf("wrk2 container exited with error for stage=%s: %s", stageName, status.Error.Message)
		}
		exitCode = status.StatusCode
	}

	if err := writeContainerLogs(ctx, dockerCli, resp.ID, filepath.Join(outputPath, "wrk_container.log")); err != nil {
		return fmt.Errorf("failed to write wrk2 container logs for stage=%s: %w", stageName, err)
	}
	if err := os.WriteFile(filepath.Join(outputPath, "exit_code.txt"), []byte(fmt.Sprintf("%d\n", exitCode)), 0o644); err != nil {
		return fmt.Errorf("failed to write wrk2 exit code for stage=%s: %w", stageName, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("wrk2 container failed for stage=%s with exit code %d", stageName, exitCode)
	}
	return nil
}

func buildWrk2Args(wrk2Params string) []string {
	args := strings.Fields(strings.TrimSpace(wrk2Params))
	if len(args) == 0 {
		return nil
	}
	for _, arg := range args {
		if arg == "--latency" {
			return args
		}
	}
	return append(args, "--latency")
}

func writeContainerLogs(ctx context.Context, dockerCli *client.Client, containerID, outputPath string) error {
	reader, err := dockerCli.ContainerLogs(ctx, containerID, dockertypes.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}

func startBenchmarkContainerStatsCollector(
	ctx context.Context,
	dockerCli *client.Client,
	containerID, outputPath string,
) (func() error, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, err
	}
	collectorCtx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamBenchmarkContainerStats(collectorCtx, dockerCli, containerID, outputPath)
	}()
	return func() error {
		cancel()
		err := <-errCh
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}, nil
}

func streamBenchmarkContainerStats(
	ctx context.Context,
	dockerCli *client.Client,
	containerID, outputPath string,
) error {
	statsResp, err := dockerCli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return err
	}
	defer statsResp.Body.Close()

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(statsResp.Body)
	encoder := json.NewEncoder(file)
	for {
		var payload dockertypes.StatsResponse
		if err := decoder.Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return nil
			}
			return err
		}
		sample := statsSampleFromDockerPayload(containerID, payload)
		if err := encoder.Encode(sample); err != nil {
			return err
		}
	}
}

func statsSampleFromDockerPayload(containerID string, payload dockertypes.StatsResponse) benchmarkContainerStatsSample {
	onlineCPUs := payload.CPUStats.OnlineCPUs
	if onlineCPUs == 0 {
		onlineCPUs = uint32(len(payload.CPUStats.CPUUsage.PercpuUsage))
	}
	memoryLimit := payload.MemoryStats.Limit
	memoryUsage := payload.MemoryStats.Usage
	memoryPercent := 0.0
	if memoryLimit > 0 {
		memoryPercent = (float64(memoryUsage) / float64(memoryLimit)) * 100.0
	}

	rxBytes := uint64(0)
	txBytes := uint64(0)
	for _, stats := range payload.Networks {
		rxBytes += stats.RxBytes
		txBytes += stats.TxBytes
	}

	cpuPercent := calcCPUPercent(payload)
	ts := payload.Read.UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return benchmarkContainerStatsSample{
		TimestampUTC:     ts,
		ContainerID:      containerID,
		ContainerName:    strings.TrimPrefix(payload.Name, "/"),
		CPUPercent:       cpuPercent,
		MemoryUsageBytes: memoryUsage,
		MemoryLimitBytes: memoryLimit,
		MemoryPercent:    memoryPercent,
		NetworkRxBytes:   rxBytes,
		NetworkTxBytes:   txBytes,
		Pids:             payload.PidsStats.Current,
		NumCPUs:          uint32(len(payload.CPUStats.CPUUsage.PercpuUsage)),
		OnlineCPUs:       onlineCPUs,
		SystemUsage:      payload.CPUStats.SystemUsage,
		TotalCPUUsage:    payload.CPUStats.CPUUsage.TotalUsage,
		PreSystemUsage:   payload.PreCPUStats.SystemUsage,
		PreTotalCPUUsage: payload.PreCPUStats.CPUUsage.TotalUsage,
	}
}

func calcCPUPercent(payload dockertypes.StatsResponse) float64 {
	cpuDelta := float64(payload.CPUStats.CPUUsage.TotalUsage - payload.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(payload.CPUStats.SystemUsage - payload.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0.0
	}
	onlineCPUs := payload.CPUStats.OnlineCPUs
	if onlineCPUs == 0 {
		onlineCPUs = uint32(len(payload.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}
	return (cpuDelta / systemDelta) * float64(onlineCPUs) * 100.0
}

func measureFirstResponse(
	ctx context.Context,
	serviceName string,
	port int,
	apiBasePath string,
	resolvedPath string,
) (*firstResponseResult, error) {
	path := readinessPath(apiBasePath, resolvedPath)
	log.Printf("Harness first-response readiness path: %s", path)

	targets := []string{
		fmt.Sprintf("http://localhost:%d%s", port, path),
		fmt.Sprintf("http://127.0.0.1:%d%s", port, path),
		fmt.Sprintf("http://%s:%d%s", serviceName, port, path),
	}

	var lastErr error
	for _, target := range targets {
		result, err := measureFirstResponseURL(ctx, target, path)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no targets to check")
	}
	return nil, lastErr
}

func measureFirstResponseURL(ctx context.Context, targetURL, resolvedPath string) (*firstResponseResult, error) {
	startedAt := time.Now().UTC()
	deadlineCtx, cancel := context.WithTimeout(ctx, defaultFirstResponseTimeout)
	defer cancel()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(defaultFirstResponseInterval)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-deadlineCtx.Done():
			return nil, fmt.Errorf("timeout waiting for first successful response at %s", targetURL)
		case <-ticker.C:
			attempts++
			req, err := http.NewRequestWithContext(deadlineCtx, http.MethodGet, targetURL, nil)
			if err != nil {
				return nil, err
			}
			resp, err := clientHTTP.Do(req)
			if err != nil {
				continue
			}
			_ = resp.Body.Close()
			// Treat any non-5xx response as service readiness to avoid false
			// negatives on endpoints that may legitimately return 4xx.
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				finishedAt := time.Now().UTC()
				duration := finishedAt.Sub(startedAt)
				return &firstResponseResult{
					TargetURL:        targetURL,
					StartedAt:        startedAt,
					FinishedAt:       finishedAt,
					DurationSeconds:  duration.Seconds(),
					DurationMillis:   duration.Milliseconds(),
					Attempts:         attempts,
					StatusCode:       resp.StatusCode,
					ResolvedPathUsed: resolvedPath,
				}, nil
			}
		}
	}
}

func deriveAPIBasePath(openAPISpecPath string) string {
	raw, err := os.ReadFile(openAPISpecPath)
	if err != nil {
		return defaultAPIBasePathPrefix
	}
	var spec struct {
		Servers []struct {
			URL string `json:"url" yaml:"url"`
		} `json:"servers" yaml:"servers"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil || len(spec.Servers) == 0 {
		if yamlErr := yaml.Unmarshal(raw, &spec); yamlErr != nil || len(spec.Servers) == 0 {
			return defaultAPIBasePathPrefix
		}
	}
	serverURL := strings.TrimSpace(spec.Servers[0].URL)
	if serverURL == "" {
		return defaultAPIBasePathPrefix
	}
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return defaultAPIBasePathPrefix
	}
	return normalizeAPIBasePath(parsed.Path)
}

func readinessPath(apiBasePath, resolvedPath string) string {
	basePath := normalizeAPIBasePath(apiBasePath)
	path := strings.TrimSpace(resolvedPath)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if basePath == "/" {
		return path
	}
	if path == basePath || strings.HasPrefix(path, basePath+"/") {
		return path
	}
	return basePath + path
}

func normalizeAPIBasePath(path string) string {
	base := strings.TrimSpace(path)
	if base == "" || base == "/" {
		return defaultAPIBasePathPrefix
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return defaultAPIBasePathPrefix
	}
	return base
}

func loadStageIterations(probeBodiesPath, stageName string) ([]datagen.MinimalIteration, error) {
	stageDir := filepath.Join(probeBodiesPath, stageName)
	if err := validateReadableDir(stageDir); err != nil {
		return nil, fmt.Errorf("probe-bodies stage directory %q is invalid: %w", stageDir, err)
	}

	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage dir %q: %w", stageDir, err)
	}
	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "iteration-") && strings.HasSuffix(name, ".json") {
			fileNames = append(fileNames, name)
		}
	}
	sort.Strings(fileNames)
	if len(fileNames) == 0 {
		return nil, fmt.Errorf("no iteration files found in %q", stageDir)
	}

	iterations := make([]datagen.MinimalIteration, 0, len(fileNames))
	for _, name := range fileNames {
		path := filepath.Join(stageDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %q: %w", path, err)
		}
		var iteration datagen.MinimalIteration
		if err := json.Unmarshal(raw, &iteration); err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", path, err)
		}
		iterations = append(iterations, iteration)
	}
	return iterations, nil
}

func writeIterations(stageDir string, iterations []datagen.MinimalIteration) error {
	for i, iteration := range iterations {
		fileName := fmt.Sprintf("iteration-%06d.json", i+1)
		outPath := filepath.Join(stageDir, fileName)
		serialized, err := json.MarshalIndent(iteration, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, serialized, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func firstResolvedPathFromProbeData(probeBodiesPath string) string {
	stageEntries, err := os.ReadDir(probeBodiesPath)
	if err != nil {
		return "/"
	}
	stageNames := make([]string, 0, len(stageEntries))
	for _, entry := range stageEntries {
		if entry.IsDir() {
			stageNames = append(stageNames, entry.Name())
		}
	}
	sort.Strings(stageNames)
	for _, stage := range stageNames {
		stageDir := filepath.Join(probeBodiesPath, stage)
		entries, err := os.ReadDir(stageDir)
		if err != nil {
			continue
		}
		fileNames := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, "iteration-") && strings.HasSuffix(name, ".json") {
				fileNames = append(fileNames, name)
			}
		}
		sort.Strings(fileNames)
		for _, name := range fileNames {
			raw, err := os.ReadFile(filepath.Join(stageDir, name))
			if err != nil {
				continue
			}
			var iteration datagen.MinimalIteration
			if err := json.Unmarshal(raw, &iteration); err != nil {
				continue
			}
			for _, step := range iteration.Steps {
				if strings.TrimSpace(step.ResolvedPath) != "" {
					return step.ResolvedPath
				}
				if strings.TrimSpace(step.PathTemplate) != "" {
					return step.PathTemplate
				}
			}
		}
	}
	return "/"
}

func sortedStageNames(dsl *flowgen.DSL) []string {
	names := make([]string, 0, len(dsl.Stages))
	for name := range dsl.Stages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeJSON(path string, payload any) error {
	serialized, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, serialized, 0o644)
}

func DockerHostFromSocketPath(socketPath string) string {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return ""
	}
	if strings.HasPrefix(socketPath, "unix://") {
		return socketPath
	}
	return "unix://" + socketPath
}

func validateReadableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", path)
	}
	return nil
}

func validateReadableDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}

func sanitizePathPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", ".", "_")
	return replacer.Replace(v)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func containsComposeService(project *types.Project, serviceName string) bool {
	for _, svc := range project.Services {
		if svc.Name == serviceName {
			return true
		}
	}
	return false
}

func getProjectNetworkName(project *types.Project) string {
	if project.Networks != nil {
		for key, networkConfig := range project.Networks {
			if strings.TrimSpace(networkConfig.Name) != "" {
				return networkConfig.Name
			}
			if key == "default" {
				return fmt.Sprintf("%s_default", project.Name)
			}
		}
	}
	return fmt.Sprintf("%s_default", project.Name)
}

// findContainerIDByServiceName finds the container ID for a given service name in a Docker Compose project.
func findContainerIDByServiceName(ctx context.Context, cli *client.Client, projectName, serviceName string) (string, error) {
	filterArgs := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", "com.docker.compose.project", projectName)),
		filters.Arg("label", fmt.Sprintf("%s=%s", "com.docker.compose.service", serviceName)),
	)

	containers, err := cli.ContainerList(ctx, dockertypes.ListOptions{
		Filters: filterArgs,
		All:     false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("no container found for service %s in project %s", serviceName, projectName)
	}
	return containers[0].ID, nil
}
