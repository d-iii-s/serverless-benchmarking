# Serverless Benchmark Suite

## Installation

### Prerequisites

- **Go 1.24+** - [Install Go](https://go.dev/doc/install)
- **Docker** - Required for running benchmarks (the tool uses Docker Compose)
- **Docker Compose** - Usually included with Docker Desktop

### Install the CLI Tool

> **Note:** This is a private repository. You'll need appropriate access and authentication configured.

#### Build from Source

Clone the repository and build locally:

```bash
git clone <repository-url>
cd serverless-benchmarking
git checkout extension-mechanism-redesign
```

**Option 1: Use the build script (Recommended)**

The `build.sh` script will check all prerequisites and build the binary:

```bash
./build.sh
```

This script will:
- Check for Go 1.24+, Git, Docker, and Docker Compose
- Verify Docker is running
- Build the `slsbench` binary if all requirements are met

**Option 2: Manual build**

If you prefer to build manually:

```bash
go build -o slsbench ./cmd/slsbench
```

**Install the binary**

After building, you can move the binary to a directory in your PATH:

```bash
sudo mv slsbench /usr/local/bin/
# or
mv slsbench ~/go/bin/
```

### Verify Installation

Check that the tool is installed correctly:

```bash
slsbench help
```

You should see the help menu with available commands.

## Quick Start

The tool provides three main commands for the benchmarking workflow:

### 1. Enrich OpenAPI Specification

Enrich an OpenAPI specification file with additional metadata:

```bash
slsbench enrich -specification-path ./api.yaml -output-path ./enriched
```

### 2. Generate Scenario

Generate a scenario file from an enriched OpenAPI specification:

```bash
slsbench scenario -enriched-specification-path ./enriched/enriched-spec-*.yaml -output-path ./scenarios
```

### 3. Run Benchmark Harness

Run the benchmark harness against a service:

```bash
slsbench harness \
  -scenario-path ./scenarios/scenario-*.json \
  -service-name myapp \
  -docker-compose-path ./docker-compose.yml \
  -port 8080 \
  -result-path ./results
```

### Getting Help

- Show general help: `slsbench help` or `slsbench`
- Show command-specific help: `slsbench help <command>` or `slsbench <command> -help`

Examples:
```bash
slsbench help enrich
slsbench scenario -help
slsbench harness -help
```

## User Guide

### Overview

The benchmarking workflow consists of three main steps:

1. **Enrich** - Add metadata hints to your OpenAPI specification
2. **Scenario** - Generate a benchmark scenario from the enriched specification
3. **Harness** - Run the benchmark against your service using Docker Compose

### Step 1: Enrich OpenAPI Specification

The `enrich` command processes your OpenAPI specification and adds metadata hints that guide data generation for benchmark scenarios.

#### Basic Usage

```bash
slsbench enrich -s ./api.yaml -o ./enriched
```

#### Flags

- `-s, --specification-path` (required): Path to your OpenAPI specification file (YAML or JSON)
- `-o, --output-path`: Directory where the enriched specification will be saved (default: `./enriched-spec.yaml`)

#### What It Does

- Reads your OpenAPI specification
- Prompts you to select data generation hints for each parameter and property
- Adds `x-user-hint` extensions to guide test data generation
- Saves the enriched specification with a timestamp

#### Example

```bash
# Enrich a specification file
slsbench enrich -s ./openapi.yaml -o ./enriched

# Output: ./enriched/enriched-spec-2025-01-15-14:30:45.yaml
```

#### Output

The enriched specification is saved with a timestamp in the filename:
- Format: `enriched-spec-YYYY-MM-DD-HH:MM:SS.yaml`
- Location: The directory specified by `--output-path`

### Step 2: Generate Scenario

The `scenario` command generates a benchmark scenario file from the enriched OpenAPI specification.

#### Basic Usage

```bash
slsbench scenario -s ./enriched/enriched-spec-*.yaml -o ./scenarios
```

#### Flags

- `-s, --enriched-specification-path` (required): Path to the enriched OpenAPI specification file
- `-o, --output-path`: Directory where the scenario file will be saved (default: `./scenario.json`)

#### What It Does

- Reads the enriched OpenAPI specification
- Builds a data model from all endpoints, operations, and parameters
- Creates a scenario graph representing the API structure
- Performs topological sorting to determine endpoint execution order
- Saves the scenario as a JSON file

#### Example

```bash
# Generate scenario from enriched specification
slsbench scenario -s ./enriched/enriched-spec-2025-01-15-14:30:45.yaml -o ./scenarios

# Output: ./scenarios/scenario-2025-01-15-14:35:12.json
```

#### Output

The scenario file is saved with a timestamp:
- Format: `scenario-YYYY-MM-DD-HH:MM:SS.json`
- Contains: Endpoint definitions, data structures, and execution order

### Step 3: Run Benchmark Harness

The `harness` command orchestrates the complete benchmark execution using Docker Compose and wrk2.

#### Basic Usage

```bash
slsbench harness \
  -s ./scenarios/scenario-*.json \
  -n myapp \
  -d ./docker-compose.yml \
  -p 8080 \
  -r ./results
```

#### Flags

- `-s, --scenario-path`: Path to the scenario file (default: `./scenario.json`)
- `-n, --service-name` (required): Name of the service in docker-compose.yml to benchmark
- `-d, --docker-compose-path`: Path to docker-compose.yml file (default: `./docker-compose.yml`)
- `-p, --port`: Port number for the service (default: `8080`)
- `-r, --result-path`: Directory to save benchmark results (default: `./result`)
- `-w, --wrk2params`: wrk2 parameters (default: `-t2 -c100 -d30s -R2000`)

#### What It Does

1. Creates a workload container with wrk2
2. Starts your service using Docker Compose
3. Connects the workload container to the service network
4. Runs benchmark tests against all endpoints in the scenario
5. Collects performance metrics:
   - Container statistics (CPU, memory)
   - RSS (Resident Set Size) information
   - First request latency
   - wrk2 benchmark results
6. Saves all results to the specified directory
7. Cleans up containers and networks

#### wrk2 Parameters

Common wrk2 parameters you can customize:

- `-t`: Number of threads (e.g., `-t4`)
- `-c`: Number of connections (e.g., `-c200`)
- `-d`: Duration (e.g., `-d60s`, `-d5m`)
- `-R`: Request rate per second (e.g., `-R5000`)

#### Example

```bash
# Run benchmark with default settings
slsbench harness \
  -s ./scenarios/scenario-2025-01-15-14:35:12.json \
  -n myapp \
  -p 8080

# Run benchmark with custom wrk2 parameters
slsbench harness \
  -s ./scenarios/scenario-2025-01-15-14:35:12.json \
  -n myapp \
  -d ./docker-compose.yml \
  -p 8080 \
  -w "-t4 -c200 -d60s -R5000" \
  -r ./benchmark-results
```

#### Output

Results are saved in a timestamped directory:
- Format: `result-YYYY-MM-DD-HH:MM:SS/`
- Contents:
  - `container-stats.csv`: Container resource usage over time
  - `rss_info.json`: Memory usage information
  - `first_request_result.json`: First request latency data
  - `wrk2_results_*.txt`: Detailed wrk2 benchmark output

### Complete Workflow Example

Here's a complete example from start to finish:

```bash
# 1. Enrich your OpenAPI specification
slsbench enrich -s ./api/openapi.yaml -o ./enriched

# 2. Generate scenario from enriched spec
slsbench scenario \
  -s ./enriched/enriched-spec-2025-01-15-14:30:45.yaml \
  -o ./scenarios

# 3. Run benchmark
slsbench harness \
  -s ./scenarios/scenario-2025-01-15-14:35:12.json \
  -n myapp \
  -d ./docker-compose.yml \
  -p 8080 \
  -w "-t2 -c100 -d30s -R2000" \
  -r ./results

# 4. Check results
ls -la ./results/result-*/
```

### Docker Compose Requirements

Your `docker-compose.yml` file should:

1. Define the service you want to benchmark
2. Expose the service on the port specified with `--port`
3. Be compatible with Docker Compose v2

Example `docker-compose.yml`:

```yaml
version: '3.8'
services:
  myapp:
    image: myapp:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
```

### Troubleshooting

#### Enrich Command Issues

**Problem**: "Failed to open OpenAPI specification"
- **Solution**: Ensure the file path is correct and the file is valid YAML or JSON

**Problem**: "Failed to save enriched specification"
- **Solution**: Check that the output directory exists and is writable

#### Scenario Command Issues

**Problem**: "Failed to generate scenario"
- **Solution**: Ensure you're using an enriched specification (from the `enrich` command), not the original

**Problem**: "Topological sort failed"
- **Solution**: This is a warning, not an error. The scenario will still be generated, but endpoint order may be undefined if there are circular dependencies

#### Harness Command Issues

**Problem**: "Docker is not running"
- **Solution**: Start Docker daemon: `sudo systemctl start docker` (Linux) or start Docker Desktop

**Problem**: "Service not found in docker-compose.yml"
- **Solution**: Verify the `--service-name` matches exactly with the service name in your docker-compose.yml

**Problem**: "Failed to connect to network"
- **Solution**: Ensure Docker Compose can create networks. Try running `docker network ls` to check

**Problem**: "Port already in use"
- **Solution**: Either stop the service using the port or change the `--port` value

### Tips and Best Practices

1. **Organize Your Files**: Keep enriched specs, scenarios, and results in separate directories
2. **Use Timestamps**: The tool automatically adds timestamps to outputs, making it easy to track versions
3. **Test Incrementally**: Start with a small API subset before benchmarking the entire API
4. **Monitor Resources**: Check `container-stats.csv` to understand resource usage patterns
5. **Customize wrk2**: Adjust `--wrk2params` based on your performance testing needs
6. **Clean Up**: The harness automatically cleans up containers, but you can manually clean with `docker compose down`
