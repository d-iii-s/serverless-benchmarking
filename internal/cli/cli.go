package cli

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/d-iii-s/slsbench/internal/service/bodyprobe"
	"github.com/d-iii-s/slsbench/internal/service/dslvalidator"
	"github.com/d-iii-s/slsbench/internal/service/harness"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "slsbench",
	Short: "Serverless Benchmarking Tool",
	Long:  `Serverless Benchmarking Tool - A comprehensive framework for evaluating the performance of serverless and containerized Java workloads.`,
}

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Run flow-driven benchmark harness against a service",
	Long: `Run benchmark harness against a service using flow stages and
probe-bodies generated iterations.

This command orchestrates the benchmark execution by:
- Starting the service using Docker Compose
- Measuring first successful response time
- Running wrk2-flow workloads per flow step
- Copying an optional mounted path from the service container to results
- Cleaning up resources`,
	Example: `  slsbench harness \
    --flow-path ./flow.yaml \
    --probe-bodies-path ./result-probe/result-2026-04-03-14:45:00 \
    --openapi-spec-path ./openapi.yml \
    --docker-compose-path ./docker-compose.yml \
    --service-name petclinic \
    --port 9966 \
    --result-path ./result \
    --service-mount-path /var/log/app \
    --docker-socket-path /var/run/docker.sock`,
	RunE: runHarness,
}

var probeBodiesCmd = &cobra.Command{
	Use:   "probe-bodies",
	Short: "Generate and probe stateful link-aware API chains",
	Long: `Generate Schemathesis stateful chains (OpenAPI links-aware) and execute
them against the running application, then persist accepted chain artifacts.`,
	RunE: runProbeBodies,
}

var (
	// Harness flags
	harnessFlowPath          string
	harnessProbeBodiesPath   string
	openApiSpecPath          string
	harnessPort              int
	harnessResultPath        string
	harnessDockerComposePath string
	harnessServiceName       string
	harnessServiceMountPaths []string
	harnessDockerSocketPath  string
	harnessDebugNon2xx       bool
	harnessReadinessPath     string

	// Probe command flags
	probeFlowPath          string
	probeOpenAPILink       string
	probeOutputPath        string
	probeDockerComposePath string
	probeDockerSocketPath  string
	probeServiceName       string
	probePort              int
	probeDebug             bool
	probeNoRewriteLinked   bool
	probeReadinessPath     string
	probeMaxTarget         int
)

func init() {
	// Harness flags
	harnessCmd.Flags().StringVarP(&harnessFlowPath, "flow-path", "f", "", "Path to the flow DSL YAML file")
	if err := harnessCmd.MarkFlagRequired("flow-path"); err != nil {
		log.Fatalf("Failed to mark --flow-path as required: %v", err)
	}

	harnessCmd.Flags().StringVarP(&harnessProbeBodiesPath, "probe-bodies-path", "b", "", "Path to probe-bodies result root containing stage iteration files")
	if err := harnessCmd.MarkFlagRequired("probe-bodies-path"); err != nil {
		log.Fatalf("Failed to mark --probe-bodies-path as required: %v", err)
	}

	harnessCmd.Flags().StringVarP(&openApiSpecPath, "openapi-spec-path", "o", "", "Path to the OpenAPI spec file")
	if err := harnessCmd.MarkFlagRequired("openapi-spec-path"); err != nil {
		log.Fatalf("Failed to mark --openapi-spec-path as required: %v", err)
	}

	harnessCmd.Flags().IntVarP(&harnessPort, "port", "p", 8080, "Application service port inside docker network")

	harnessCmd.Flags().StringVarP(&harnessResultPath, "result-path", "r", "./result", "Path to save the results")

	harnessCmd.Flags().StringVarP(&harnessDockerComposePath, "docker-compose-path", "d", "", "Path to the docker-compose.yml file")
	if err := harnessCmd.MarkFlagRequired("docker-compose-path"); err != nil {
		log.Fatalf("Failed to mark --docker-compose-path as required: %v", err)
	}

	harnessCmd.Flags().StringVarP(&harnessServiceName, "service-name", "n", "", "Service name in the docker-compose file to benchmark (required)")
	if err := harnessCmd.MarkFlagRequired("service-name"); err != nil {
		log.Fatalf("Failed to mark --service-name as required: %v", err)
	}

	harnessCmd.Flags().StringSliceVarP(&harnessServiceMountPaths, "service-mount-path", "m", []string{}, "Optional paths inside service container to copy to results (repeat flag or use comma-separated values)")
	harnessCmd.Flags().StringVar(&harnessDockerSocketPath, "docker-socket-path", "/var/run/docker.sock", "Path to Docker socket for DooD mode")
	harnessCmd.Flags().BoolVar(&harnessDebugNon2xx, "debug-non2xx", false, "Enable FLOW_DEBUG_NON2XX=1 in wrk2 container for non-2xx debug capture")
	harnessCmd.Flags().StringVar(&harnessReadinessPath, "readiness-path", "", "Explicit readiness probe path (auto-derived from OpenAPI if empty)")

	// Probe-bodies flags
	probeBodiesCmd.Flags().StringVarP(&probeFlowPath, "flow-path", "f", "", "Path to flow DSL YAML file")
	if err := probeBodiesCmd.MarkFlagRequired("flow-path"); err != nil {
		log.Fatalf("Failed to mark --flow-path as required: %v", err)
	}

	probeBodiesCmd.Flags().StringVarP(&probeOpenAPILink, "openapi-link", "o", "", "OpenAPI file path or URL")
	if err := probeBodiesCmd.MarkFlagRequired("openapi-link"); err != nil {
		log.Fatalf("Failed to mark --openapi-link as required: %v", err)
	}

	probeBodiesCmd.Flags().StringVarP(&probeOutputPath, "output-path", "r", "./result-probe", "Output path for accepted generated bodies")
	if err := probeBodiesCmd.MarkFlagRequired("output-path"); err != nil {
		log.Fatalf("Failed to mark --output-path as required: %v", err)
	}

	probeBodiesCmd.Flags().StringVarP(&probeDockerComposePath, "docker-compose-path", "d", "", "Path to the docker-compose.yml file for benchmark application")
	if err := probeBodiesCmd.MarkFlagRequired("docker-compose-path"); err != nil {
		log.Fatalf("Failed to mark --docker-compose-path as required: %v", err)
	}
	probeBodiesCmd.Flags().StringVar(&probeDockerSocketPath, "docker-socket-path", "/var/run/docker.sock", "Path to Docker socket for DooD mode")

	probeBodiesCmd.Flags().StringVarP(&probeServiceName, "service-name", "n", "", "Service name in docker-compose to probe")
	if err := probeBodiesCmd.MarkFlagRequired("service-name"); err != nil {
		log.Fatalf("Failed to mark --service-name as required: %v", err)
	}

	probeBodiesCmd.Flags().IntVarP(&probePort, "port", "p", 8080, "Local running service port to probe")
	if err := probeBodiesCmd.MarkFlagRequired("port"); err != nil {
		log.Fatalf("Failed to mark --port as required: %v", err)
	}
	probeBodiesCmd.Flags().BoolVar(&probeDebug, "debug", false, "Enable detailed probe debug logs")
	probeBodiesCmd.Flags().BoolVar(
		&probeNoRewriteLinked,
		"no-rewrite-linked-values",
		false,
		"Disable replacing linked values with JSON pointers in generated output",
	)
	probeBodiesCmd.Flags().StringVar(&probeReadinessPath, "readiness-path", "", "Explicit readiness probe path (auto-derived from OpenAPI if empty)")
	probeBodiesCmd.Flags().IntVar(&probeMaxTarget, "max-probe-target", 0, "Cap the number of generated iterations per stage (0 = unlimited)")

	// Adding commands to root
	rootCmd.AddCommand(harnessCmd)
	rootCmd.AddCommand(probeBodiesCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runValidateDSL(dslPath string) error {
	ctx := context.Background()
	log.Println("Validating DSL file:", dslPath)

	file, err := os.Open(dslPath)
	if err != nil {
		return fmt.Errorf("failed to open DSL file %q: %w", dslPath, err)
	}
	defer file.Close()

	var doc any
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("failed to parse DSL YAML %q: %w", dslPath, err)
	}

	if err := dslvalidator.ValidateDSL(ctx, doc); err != nil {
		// Try to pretty-print jsonschema validation errors if possible.
		log.Printf("DSL validation failed for %s", dslPath)
		utils.PrintJSON(err)
		return fmt.Errorf("DSL validation failed: %w", err)
	}

	log.Printf("DSL validation passed for %s", dslPath)
	return nil
}

func runHarness(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validation input arguments
	if harnessPort <= 0 {
		return fmt.Errorf("the --port flag must be a positive integer")
	}

	if err := runValidateDSL(harnessFlowPath); err != nil {
		return fmt.Errorf("flow file validation failed: %w", err)
	}

	log.Printf("Running harness: flow=%s probe-bodies=%s openapi=%s result=%s docker-compose=%s service=%s port=%d docker-socket=%s service-mount-paths=%v debug-non2xx=%t readiness-path=%q",
		harnessFlowPath, harnessProbeBodiesPath, openApiSpecPath, harnessResultPath, harnessDockerComposePath, harnessServiceName, harnessPort, harnessDockerSocketPath, harnessServiceMountPaths, harnessDebugNon2xx, harnessReadinessPath)

	return harness.Run(
		ctx,
		harnessFlowPath,
		harnessResultPath,
		openApiSpecPath,
		harnessDockerComposePath,
		harnessServiceName,
		harnessPort,
		harnessServiceMountPaths,
		harnessProbeBodiesPath,
		harnessDockerSocketPath,
		harnessDebugNon2xx,
		harnessReadinessPath,
	)
}

func runProbeBodies(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if probePort <= 0 {
		return fmt.Errorf("the --port flag must be a positive integer")
	}
	if probeDockerComposePath == "" {
		return fmt.Errorf("the --docker-compose-path flag must be provided")
	}
	if probeServiceName == "" {
		return fmt.Errorf("the --service-name flag must be provided")
	}
	if err := runValidateDSL(probeFlowPath); err != nil {
		return fmt.Errorf("flow file validation failed: %w", err)
	}
	log.Printf("Running probe-bodies: flow=%s openapi=%s output=%s docker-compose=%s docker-socket=%s service=%s port=%d no-rewrite-linked-values=%t readiness-path=%q max-probe-target=%d",
		probeFlowPath, probeOpenAPILink, probeOutputPath, probeDockerComposePath, probeDockerSocketPath, probeServiceName, probePort, probeNoRewriteLinked, probeReadinessPath, probeMaxTarget)

	if err := bodyprobe.Run(
		ctx,
		probeFlowPath,
		probeOpenAPILink,
		probeOutputPath,
		probeDockerComposePath,
		probeDockerSocketPath,
		probeServiceName,
		probePort,
		probeDebug,
		probeNoRewriteLinked,
		probeReadinessPath,
		probeMaxTarget,
	); err != nil {
		return err
	}
	return nil
}
