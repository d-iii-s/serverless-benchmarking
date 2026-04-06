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

The `harness` command orchestrates benchmark execution using flow stages,
`probe-bodies` generated iterations, Docker Compose, and `wrk2-flow` containers.

#### Basic Usage

```bash
slsbench harness \
  --flow-path ./workdir/spring-petclinic-rest/flow.yaml \
  --probe-bodies-path ./workdir/spring-petclinic-rest/result-2026-04-03-14:45:00 \
  --openapi-spec-path ./workdir/spring-petclinic-rest/openapi.yml \
  --docker-compose-path ./workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  --service-name petclinic \
  --port 9966 \
  --result-path ./workdir/spring-petclinic-rest/results \
  --service-mount-path /var/log/app \
  --docker-socket-path /var/run/docker.sock
```

#### Flags

- `-f, --flow-path` (required): Flow DSL YAML path.
- `-b, --probe-bodies-path` (required): Path to `probe-bodies` output root (contains `<stage>/iteration-*.json`).
- `-o, --openapi-spec-path` (required): OpenAPI spec path.
- `-d, --docker-compose-path` (required): Application docker-compose path.
- `-n, --service-name` (required): Service name in docker-compose.
- `-p, --port`: Service port inside Docker network (default: `8080`).
- `-r, --result-path`: Base output path (a timestamped run directory is created inside it).
- `-m, --service-mount-path`: Optional paths inside service container to copy to results (repeat `-m` or use comma-separated values).
- `--docker-socket-path`: Docker socket path for DooD mode (default: `/var/run/docker.sock`).

#### What It Does

1. Starts your service with Docker Compose.
2. Measures time to first successful response and writes `first_request_result.json`.
3. For each flow stage, starts one dedicated `wrk2-flow` container run.
4. Forces `wrk2` latency histogram output via `--latency`.
5. Mounts stage probe data into the benchmark container and saves stage-level wrk output artifacts.
6. Optionally copies one or more mounted paths from the service container to results.
7. Always tears down compose resources.

#### Example

```bash
# Run benchmark from probe-bodies generated iterations
slsbench harness \
  -f ./workdir/spring-petclinic-rest/flow.yaml \
  -b ./workdir/spring-petclinic-rest/result-2026-04-03-14:45:00 \
  -o ./workdir/spring-petclinic-rest/openapi.yml \
  -d ./workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  -n petclinic \
  -p 9966 \
  -r ./workdir/spring-petclinic-rest/results

# Run benchmark and collect logs/metrics from the service container
slsbench harness \
  -f ./flow.yaml \
  -b ./result-probe/result-2026-04-03-14:45:00 \
  -o ./openapi.yml \
  -d ./docker-compose.yml \
  -n myapp \
  -p 8080 \
  -m /var/log/app
```

#### Collecting Files from Service Container

Use `--service-mount-path` (`-m`) to copy one or more files/directories from the service container to your results folder after benchmark completion. This is useful for:

- Application logs
- JFR (Java Flight Recorder) recordings
- Custom metrics files
- Debug output
- Profiling data

Paths should be absolute paths inside the container:

```bash
# Collect application logs
slsbench harness -n myapp -m /var/log/app

# Collect multiple paths
slsbench harness -n myapp -m /var/log/app -m /tmp/metrics

# Collect JFR recordings
slsbench harness -n myapp -m /tmp/recording.jfr
```

Collected files are saved in a `collected/` subdirectory within the results folder.

#### Output

Results are saved in a timestamped directory:
- Probe output format: `probe-bodies-result-YYYY-MM-DD-HH:MM:SS/`
- Harness output format: `harness-result-YYYY-MM-DD-HH:MM:SS/`
- Contents:
  - `first_request_result.json`: First request latency data
  - `wrk2-input/<stage>/<stage>/iteration-*.json`: Stage input data used for one run per stage
  - `wrk2-results/<stage>/`: wrk output files and container logs for each stage
  - `collected/`: Files copied from service container (if `--service-mount-path` was used)

### Complete Workflow Example

Here's a complete example from start to finish:

```bash
# 1. Enrich your OpenAPI specification
slsbench enrich -s ./api/openapi.yaml -o ./enriched

# 2. Generate probe-bodies data
slsbench probe-bodies \
  -f ./workdir/spring-petclinic-rest/flow.yaml \
  -o ./workdir/spring-petclinic-rest/openapi.yml \
  -r ./workdir/spring-petclinic-rest \
  -d ./workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  -n petclinic \
  -p 9966

# 3. Run harness benchmark
slsbench harness \
  -f ./workdir/spring-petclinic-rest/flow.yaml \
  -b ./workdir/spring-petclinic-rest/result-2026-04-03-14:45:00 \
  -o ./workdir/spring-petclinic-rest/openapi.yml \
  -d ./workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  -n petclinic \
  -p 9966 \
  -r ./results

# 4. Check results
ls -la ./results/harness-result-*/
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

### Run in Docker (DooD)

You can run the full `slsbench` CLI in a container and still control the host Docker daemon by mounting the Docker socket (Docker-out-of-Docker pattern).

#### Build the Image

```bash
docker build -t slsbench:dood .
```

#### Probe-Bodies in DooD

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":/workspace \
  -w /workspace \
  slsbench:dood probe-bodies \
  --flow-path /workspace/workdir/spring-petclinic-rest/flow.yaml \
  --openapi-link /workspace/workdir/spring-petclinic-rest/openapi.yml \
  --output-path /workspace/workdir/spring-petclinic-rest \
  --docker-compose-path /workspace/workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  --service-name petclinic \
  --port 9966 \
  --docker-socket-path /var/run/docker.sock
```

#### Harness in DooD

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":/workspace \
  -w /workspace \
  slsbench:dood harness \
  --flow-path /workspace/workdir/spring-petclinic-rest/flow.yaml \
  --probe-bodies-path /workspace/workdir/spring-petclinic-rest/probe-bodies-result-2026-04-03-14:45:00 \
  --openapi-spec-path /workspace/workdir/spring-petclinic-rest/openapi.yml \
  --docker-compose-path /workspace/workdir/spring-petclinic-rest/docker-compose.petclinic.yml \
  --service-name petclinic \
  --port 9966 \
  --result-path /workspace/workdir/spring-petclinic-rest/results \
  --docker-socket-path /var/run/docker.sock
```

Required mounts:
- Docker socket (`/var/run/docker.sock`) from host to container.
- Project/workdir directory that contains `flow`, OpenAPI spec, compose file, and output directories.

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
