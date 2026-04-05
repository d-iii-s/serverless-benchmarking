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
	Short: "Run benchmark harness against a service using a scenario file",
	Long: `Run benchmark harness against a service using a scenario file.

This command orchestrates the benchmark execution by:
- Starting the service using Docker Compose
- Running workload tests using wrk2
- Collecting performance metrics
- Copying specified paths from the service container to results
- Cleaning up resources`,
	Example: `  slsbench harness -scenario-path ./scenario.json -service-name myapp -port 8080
  slsbench harness -scenario-path ./scenario.json -service-name myapp -wrk2params "-t4 -c200 -d60s -R5000"
  slsbench harness -s ./scenario.json -n myapp -c /var/log/app,/tmp/metrics`,
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
	harnessScenarioPath      string
	openApiSpecPath          string
	harnessPort              int
	harnessResultPath        string
	harnessDockerComposePath string
	harnessServiceName       string
	harnessCollectPaths      []string

	// Probe command flags
	probeFlowPath          string
	probeOpenAPILink       string
	probeOutputPath        string
	probeDockerComposePath string
	probeServiceName       string
	probePort              int
	probeDebug             bool
	probeNoRewriteLinked   bool
)

func init() {
	// Harness flags
	harnessCmd.Flags().StringVarP(&harnessScenarioPath, "scenario-path", "s", "", "Path to the scenario file")
	if err := harnessCmd.MarkFlagRequired("scenario-path"); err != nil {
		log.Fatalf("Failed to mark --scenario-path as required: %v", err)
	}

	harnessCmd.Flags().StringVarP(&openApiSpecPath, "openapi-spec-path", "o", "", "Path to the OpenAPI spec file (optional)")
	if err := harnessCmd.MarkFlagRequired("openapi-spec-path"); err != nil {
		log.Fatalf("Failed to mark --openapi-spec-path as required: %v", err)
	}

	harnessCmd.Flags().IntVarP(&harnessPort, "port", "p", 8080, "Local port  for the benchmark harness to use")

	harnessCmd.Flags().StringVarP(&harnessResultPath, "result-path", "r", "./result", "Path to save the results")

	harnessCmd.Flags().StringVarP(&harnessDockerComposePath, "docker-compose-path", "d", "", "Path to the docker-compose.yml file")
	if err := harnessCmd.MarkFlagRequired("docker-compose-path"); err != nil {
		log.Fatalf("Failed to mark --docker-compose-path as required: %v", err)
	}

	harnessCmd.Flags().StringVarP(&harnessServiceName, "service-name", "n", "", "Service name in the docker-compose file to benchmark (required)")
	if err := harnessCmd.MarkFlagRequired("service-name"); err != nil {
		log.Fatalf("Failed to mark --service-name as required: %v", err)
	}

	harnessCmd.Flags().StringSliceVarP(&harnessCollectPaths, "collect-paths", "c", []string{}, "Paths inside the service container to copy to results (e.g., /var/log/app,/tmp/metrics)")

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

	err := runValidateDSL(harnessScenarioPath)
	if err != nil {
		log.Fatalf("Scenario file validation failed: %v", err)
	}

	log.Printf("Running harness: scenario=%s result=%s docker-compose=%s port=%d service=%s",
		harnessScenarioPath, harnessResultPath, harnessDockerComposePath, harnessPort, harnessServiceName)
	if len(harnessCollectPaths) > 0 {
		log.Printf("Will collect paths from service container: %v", harnessCollectPaths)
	}

	harness.Run(ctx, harnessScenarioPath, harnessResultPath, harnessDockerComposePath, harnessServiceName, harnessPort, harnessCollectPaths, openApiSpecPath)
	return nil
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
	log.Printf("Running probe-bodies: flow=%s openapi=%s output=%s docker-compose=%s service=%s port=%d no-rewrite-linked-values=%t",
		probeFlowPath, probeOpenAPILink, probeOutputPath, probeDockerComposePath, probeServiceName, probePort, probeNoRewriteLinked)

	if err := bodyprobe.Run(
		ctx,
		probeFlowPath,
		probeOpenAPILink,
		probeOutputPath,
		probeDockerComposePath,
		probeServiceName,
		probePort,
		probeDebug,
		probeNoRewriteLinked,
	); err != nil {
		return err
	}
	return nil
}
