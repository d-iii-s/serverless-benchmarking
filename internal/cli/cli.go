package cli

import (
	"context"
	"fmt"
	"log"
	"os"

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

var validateDslCmd = &cobra.Command{
	Use:   "validate-dsl",
	Short: "Validate a DSL YAML file against the built-in JSON Schema",
	Long: `Validate a DSL YAML file against the built-in JSON Schema.

This command reads a DSL definition from a YAML file and validates it
against the built-in DSL JSON Schema. It prints a human-readable result
and exits with code 0 on success, 1 on validation error.`,
	RunE: runValidateDSL,
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
	// Harness flags
	harnessScenarioPath      string
	harnessWrk2Params        string
	harnessPort              int
	harnessResultPath        string
	harnessDockerComposePath string
	harnessServiceName       string
	harnessCollectPaths      []string

	// Validate DSL flags
	validateDSLPath string
)

func init() {
	// Validate DSL command flags
	validateDslCmd.Flags().StringVarP(&validateDSLPath, "dsl-path", "d", "", "Path to the DSL YAML file to validate (required)")
	validateDslCmd.MarkFlagRequired("dsl-path")

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
	rootCmd.AddCommand(validateDslCmd)
	rootCmd.AddCommand(harnessCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runValidateDSL(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log.Println("Validating DSL file:", validateDSLPath)

	file, err := os.Open(validateDSLPath)
	if err != nil {
		return fmt.Errorf("failed to open DSL file %q: %w", validateDSLPath, err)
	}
	defer file.Close()

	var doc any
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("failed to parse DSL YAML %q: %w", validateDSLPath, err)
	}

	if err := dslvalidator.ValidateDSL(ctx, doc); err != nil {
		// Try to pretty-print jsonschema validation errors if possible.
		log.Printf("DSL validation failed for %s", validateDSLPath)
		utils.PrintJSON(err)
		return fmt.Errorf("DSL validation failed: %w", err)
	}

	log.Printf("DSL validation passed for %s", validateDSLPath)
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
