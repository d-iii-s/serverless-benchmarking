# slsbench v3.0.0

A scenario-based benchmarking framework for serverless and containerized HTTP applications. Instead of isolated endpoint probes or aggregate latency numbers, `slsbench` derives stateful, link-aware API request chains from OpenAPI specifications and replays them through a Flow DSL that models realistic usage patterns under controlled load.

## What's New in v3.0.0

### New Commands

- **`probe-bodies`** — Generates stateful API chains using [Schemathesis](https://schemathesis.readthedocs.io/) and OpenAPI Links against a running application. Only 2xx-accepted chains are persisted as reusable `iteration-*.json` artifacts, separating scenario validity from performance measurement.

- **`harness`** — Replays pre-generated probe-bodies iterations under rate-controlled `wrk2-flow` load. Measures time-to-first-response, streams container-level resource stats (CPU, memory, network I/O), and executes one `wrk2-flow` container per flow stage. Supports optional file collection from the service container.

### Flow DSL

Benchmark scenarios are defined in YAML with JSON Schema validation. Each stage combines a load profile (`wrk2params`) and a directed usage model (`flow`) with weighted round-robin transitions, entry nodes, terminal nodes, and optional field mappings between steps.

### Docker-out-of-Docker (DooD)

The recommended execution model. The `slsbench` container mounts the host Docker socket and controls sibling application/benchmark containers. All flow, OpenAPI, compose, and output paths use container-side mount locations.

### Other Changes

- `--max-probe-target` flag to cap iterations generated per stage (avoids long probe runs)
- `--service-mount-path` flag to copy files from the service container to results
- `--debug-non2xx` flag for non-2xx response capture in wrk2 containers
- Removed legacy commands (`enrich`, `scenario`, `walker`) and all associated dead code
- Dropped unused dependencies (`promptui`, `kin-openapi`), reducing `go.sum` by ~40%
- Fixed CI pipelines (`.gitlab-ci.yml` and `.github/workflows/ci.yml`) to test current packages
- Expanded README with Mermaid architecture/lifecycle/topology diagrams and thesis-oriented framing

## Docker Image

```bash
docker pull aape2k/slsbench:v3.0.0
```

## Quick Start

```bash
# Step 1: Generate stateful probe bodies
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":/workspace \
  aape2k/slsbench:v3.0.0 probe-bodies \
    --flow-path /workspace/flow.yaml \
    --openapi-link /workspace/openapi.yml \
    --output-path /workspace/probe-output \
    --docker-compose-path /workspace/docker-compose.yml \
    --service-name petclinic \
    --port 9966 \
    --max-probe-target 500

# Step 2: Run the benchmark harness
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":/workspace \
  aape2k/slsbench:v3.0.0 harness \
    --flow-path /workspace/flow.yaml \
    --probe-bodies-path /workspace/probe-output/probe-bodies-result-<timestamp> \
    --openapi-spec-path /workspace/openapi.yml \
    --docker-compose-path /workspace/docker-compose.yml \
    --service-name petclinic \
    --port 9966 \
    --result-path /workspace/results
```

## Full Documentation

See the [README](https://github.com/d-iii-s/serverless-benchmarking/blob/v3.0.0/README.md) for complete documentation including Flow DSL reference, OpenAPI requirements, output structure, and troubleshooting.
