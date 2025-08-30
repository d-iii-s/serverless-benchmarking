# Quarkus Tika Benchmark

A document parsing microservice built with Quarkus and Apache Tika. It exposes endpoints for uploading and parsing documents (e.g., ODT, PDF), simulating serverless document processing workloads.

## Prebuilt images on DockerHub

- JVM: `aape2k/tika-jvm`
- Native: `aape2k/tika-native`

The application image was built on `Debian 6.1.140-1 (2025-05-22) x86_64 GNU/Linux` using 
`Oracle GraalVM 21.0.7` for Java. The build environment featured an Intel(R) Xeon(R) CPU @ 2.20GHz (8 cores) and 32GB RAM.

## Example Workload Configurations

- [tika-native](../../example-configs/tika-native.json)
- [tika-jvm](../../example-configs/tika-jvm.json)

## How to build JVM and NATIVE images?

Run script to build `native binary` and `JVM` files
```bash
./build.sh --maven-options="-Dquarkus.native.monitoring=jfr"
```
> **Note**: The user should be logged in to Docker.

> **Note**: `target` directory should be writable

### JVM

To build image execute following command:
```bash
docker build -f example-docker/jvm.Dockerfile -t aape2k/tika-jvm .
```

### Native

To build image execute following command:
```bash
docker build -f example-docker/native.Dockerfile -t aape2k/tika-native .
```
> **Note**: The image tag (`label`) can be changed

## Workloads

This benchmark provides the following example workloads, as defined in [`api/config.json`](./api/config.json):

### 1. `mixed-requests`

- **exampleName**: `sampleOdt`
- **wrkScripts**:
  - `mixed-requests.lua`

This workload issues a mix of different request types to the service, simulating a realistic and varied document processing usage pattern.

### 2. `odt-requests`

- **exampleName**: `sampleOdt`
- **wrkScripts**:
  - `odt-requests.lua`

This workload focuses on sending ODT document parsing requests to the service, 
useful for measuring performance and correctness for ODT files specifically.

### 3. `pdf-requests`

- **exampleName**: `samplePdf`
- **wrkScripts**:
  - `pdf-requests.lua`

This workload focuses on sending PDF document parsing requests to the service, 
useful for evaluating PDF-specific document processing.

You can select the desired workload by setting the `benchmarkConfigName` in your
harness configuration (see the example configs linked above).


