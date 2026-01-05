package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/d-iii-s/slsbench/internal/model"
	enricher "github.com/d-iii-s/slsbench/internal/service/enrich"
	"github.com/d-iii-s/slsbench/internal/service/harness"
	scenario_builder "github.com/d-iii-s/slsbench/internal/service/scenario-builder"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "slsbench",
	Short: "Serverless Benchmarking Tool",
	Long:  `Serverless Benchmarking Tool - A comprehensive framework for evaluating the performance of serverless and containerized Java workloads.`,
}

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich an OpenAPI specification file with additional metadata",
	Long: `Enrich an OpenAPI specification file with additional metadata.

This command processes an OpenAPI specification file and adds additional metadata
that is required for generating benchmark scenarios.`,
	Example: `  slsbench enrich -specification-path ./api.yaml -output-path ./enriched
  slsbench enrich -specification-path ./api.yaml`,
	RunE: runEnrich,
}

var scenarioCmd = &cobra.Command{
	Use:   "scenario",
	Short: "Generate a scenario file from an enriched OpenAPI specification",
	Long: `Generate a scenario file from an enriched OpenAPI specification.

This command takes an enriched OpenAPI specification and generates a scenario file
that can be used with the harness command to run benchmarks.`,
	Example: `  slsbench scenario -enriched-specification-path ./enriched-spec.yaml -output-path ./scenarios
  slsbench scenario -enriched-specification-path ./enriched-spec.yaml`,
	RunE: runScenario,
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

var (
	// Enrich flags
	enrichSpecificationPath string
	enrichOutputPath        string

	// Scenario flags
	scenarioEnrichedSpecPath string
	scenarioOutputPath       string

	// Harness flags
	harnessScenarioPath      string
	harnessWrk2Params        string
	harnessPort              int
	harnessResultPath        string
	harnessDockerComposePath string
	harnessServiceName       string
	harnessCollectPaths      []string
)

func init() {
	// Enrich command flags
	enrichCmd.Flags().StringVarP(&enrichSpecificationPath, "specification-path", "s", "", "Path to the OpenAPI specification file to enrich (required)")
	enrichCmd.MarkFlagRequired("specification-path")
	enrichCmd.Flags().StringVarP(&enrichOutputPath, "output-path", "o", "./enriched-spec.yaml", "Path to save the enriched OpenAPI specification file")

	// Scenario command flags
	scenarioCmd.Flags().StringVarP(&scenarioEnrichedSpecPath, "enriched-specification-path", "s", "", "Path to the OpenAPI Enriched Specification file (required)")
	scenarioCmd.MarkFlagRequired("enriched-specification-path")
	scenarioCmd.Flags().StringVarP(&scenarioOutputPath, "output-path", "o", "./scenario.json", "Path to save the generated scenario file")

	// Harness command flags
	harnessCmd.Flags().StringVarP(&harnessScenarioPath, "scenario-path", "s", "./scenario.json", "Path to the scenario file")
	harnessCmd.Flags().StringVarP(&harnessWrk2Params, "wrk2params", "w", "-t2 -c100 -d30s -R2000", "Additional wrk2 parameters")
	harnessCmd.Flags().IntVarP(&harnessPort, "port", "p", 8080, "Port for the benchmark harness to use")
	harnessCmd.Flags().StringVarP(&harnessResultPath, "result-path", "r", "./result", "Path to save the results")
	harnessCmd.Flags().StringVarP(&harnessDockerComposePath, "docker-compose-path", "d", "./docker-compose.yml", "Path to the docker-compose.yml file")
	harnessCmd.Flags().StringVarP(&harnessServiceName, "service-name", "n", "", "Service name in the docker-compose file to benchmark (required)")
	harnessCmd.MarkFlagRequired("service-name")
	harnessCmd.Flags().StringSliceVarP(&harnessCollectPaths, "collect-paths", "c", []string{}, "Paths inside the service container to copy to results (e.g., /var/log/app,/tmp/metrics)")

	// Add subcommands to root
	rootCmd.AddCommand(enrichCmd)
	rootCmd.AddCommand(scenarioCmd)
	rootCmd.AddCommand(harnessCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runEnrich(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log.Println("Enriching specification:", enrichSpecificationPath)

	specification := utils.OpenOpenApiSpecFile(ctx, enrichSpecificationPath)
	enrichProcessor := enricher.NewEnrichProcessor(specification, nil)
	enrichProcessor.SetHints()

	outputFile := enrichOutputPath + fmt.Sprintf("/enriched-spec-%s.yaml", time.Now().Format("2006-01-02-15:04:05"))
	err := utils.SaveYAML(outputFile, enrichProcessor.GetSpec())
	if err != nil {
		return fmt.Errorf("failed to save enriched specification: %w", err)
	}

	log.Printf("Enriched specification saved to: %s", outputFile)
	return nil
}

func runScenario(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log.Println("Generating scenario from specification:", scenarioEnrichedSpecPath)

	specification := utils.OpenOpenApiSpecFile(ctx, scenarioEnrichedSpecPath)
	scenario := scenario_builder.CreateScenarioFromSpecPath(ctx, specification, nil)

	outputFile := scenarioOutputPath + fmt.Sprintf("/scenario-%s.json", time.Now().Format("2006-01-02-15:04:05"))
	err := model.SerializeScenarioGraph(scenario, outputFile)
	if err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}

	log.Printf("Scenario saved to: %s", outputFile)
	return nil
}

func runHarness(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if harnessPort <= 0 {
		return fmt.Errorf("the --port flag must be a positive integer")
	}

	log.Printf("Running harness: scenario=%s wrk2=%s result=%s docker-compose=%s port=%d service=%s",
		harnessScenarioPath, harnessWrk2Params, harnessResultPath, harnessDockerComposePath, harnessPort, harnessServiceName)
	if len(harnessCollectPaths) > 0 {
		log.Printf("Will collect paths from service container: %v", harnessCollectPaths)
	}

	harness.Run(ctx, harnessScenarioPath, harnessWrk2Params, harnessResultPath, harnessDockerComposePath, harnessServiceName, harnessPort, harnessCollectPaths)
	return nil
}
