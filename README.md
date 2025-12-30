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
