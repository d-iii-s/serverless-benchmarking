## Installation & Prerequisites
- Docker
  - Docker Engine minimal version is `27.3.1`
  - Docker API version must not exceed `1.47`
- GraalVM JDK (to compile benchmarks)
  -  minimal version is `21.0.2`
- Go SDK (to compile harness)
  - minimal version `1.23.2`

> **Note**: This harness has been developed and tested on Linux. Its behavior on macOS and Windows has not been evaluated.

## How to run benchmark

### Building binary

The binary file can be downloaded from the latest release or built locally. To build the binary locally, a minimum Go version of 1.23.2 is required. Assuming you are in the root of the project, the following command will produce a binary named serverless-benchmark:

```bash
 go build -o serverless-benchmark ./cmd
```

The following guidance assumes that the binary file named serverless-benchmark is located in the root of the project.

### Adjusting config

The only required argument for the harness is the path to a valid configuration file. This file must be in JSON format and contain the following fields *(below is an example of a config for micronaut-shopcart benchmark)*:

```json
{
  "benchmarkName": "micronaut-shopcart",
  "workloadImage": "aape2k/workload-generator",
  "benchmarkImage": "aape2k/shopcart-jvm",
  "hostPort": "8003",
  "resultPath": "./resultPath",
  "benchmarksRootPath": "./benchmark",
  "benchmarkConfigName": "create-populate-read-pipeline",
  "wrk2params": "-t2 -c50 -d30s -R2000 --latency"
}
```

#### Mandatory fields:

- **benchmarkName** - Name of the benchmark (list of the available benchmarks is provided bellow)
- **workloadImage** - Name of the Docker image used for the workload generator.
- **benchmarkImage** - Name of the Docker image containing the benchmarked application.
- **hostPort** - Port exposed on the host to access the application.
- **resultPath** - Host directory where logs and results will be saved.
- **benchmarksRootPath** - Path to the benchmark source directory.
- **benchmarkConfigName** - Name of the workload within the benchmark (a list of available workloads for each benchmark is provided below).
- **wrk2params** - Parameters passed to wrk2 (e.g., concurrency, duration, rate).

#### Optional fields:

- **networkName** - If specified, creates a named Docker network for benchmark and workload communication. If not provided, a name is generated from the config hash.
- **benchmarkContainerName** - Custom name for the benchmark container. If omitted, one is derived from the config hash.
- **javaOptions** - Additional JVM options to be passed to the Java process running inside the benchmark container. This string is injected as the value of the `HARNESS_JAVA_OPTS` environment variable and can be used to tune memory settings (e.g., `-Xmx2g`), enable debugging, set system properties, or adjust garbage collection parameters. If omitted, no extra options are provided.

> **Note:** Paths can be absolute or relative to the harness binary execution path.

> **Note:** Example configurations for all available benchmarks are located in the /example-configs folder.

> **Note:** JFR logs can be collected by specifying `-XX:StartFlightRecording=filename=/app/logs/result.jfr,settings=profile,dumponexit=true` in javaOptions

Available Benchmarks: (**benchmarkName** and *benchmarkConfigName*):
- **micronaut-shopcart**
  - *mixed-requests*
  - *create-populate-read-pipeline*
- **quarkus-tika**
  - *mixed-requests*
  - *odt-requests*
  - *pdf-requests*
- **spring-petclinic**
  - *mixed-requests*
  - *create-read-pipeline*

All benchmark images are publicly available on Docker Hub and can be used to run the harness without building any image:

- `aape2k/petclinic-jvm` – Spring Petclinic (JVM)
- `aape2k/petclinic-native` – Spring Petclinic (Native)
- `aape2k/shopcart-jvm` – Micronaut Shopcart (JVM)
- `aape2k/shopcart-native` – Micronaut Shopcart (Native)
- `aape2k/tika-jvm` – Quarkus Tika (JVM)
- `aape2k/tika-native` – Quarkus Tika (Native, experimental)

More info about benchmarks, images and how to build benchmarks localy can be found at [Benchmark README](../benchmark/README.md) 

To successfully run the harness, **benchmarkName** and **benchmarkConfigName** must match the **benchmarkImage** specified in the configuration.

### Workload Generator

The harness uses WRK2 as its workload generator. WRK2 enables the generation of HTTP requests against the benchmarked application with configurable parameters such as duration, requests per second, number of connections, and more. In the harness configuration, you must specify the name of the workload generator image. Like the benchmark images, the workload generator image is prebuilt and publicly available on Docker Hub under the following name:

- `aape2k/workload-image`

You can also build the workload generator image locally. The Dockerfile is located at `harness/docker/workload-generator.docker`. To build the Docker image, run the following command from the root of the project:

```bash
docker build -f example-docker/jvm.Dockerfile -t aape2k/workload-generator .
```

### Running Harness

After succesfully adjusting config based on your preferences and local environment harness can be
started by executing following command:

```bash
./serverless-benchmark --config-path /path/to/config/file.json
```

The harness requires only one parameter - `config-path`, which is the path to the adjusted configuration file. The output from the command should look similar to the example below:

```bash
API server listening at: 127.0.0.1:42599
2025/06/19 11:56:52 Connected to docker host with cient version - 1.48
2025/06/19 11:56:52 Network created with ID: 37df5f401f7d6664c8fe0ff1e401c052abf997411d7dbcacee8c58d3f9795d93
2025/06/19 11:56:52 Benchmark container created with ID: fa51c094264c46ae7352365ecf2c94217f11a8009f332052769b7d7ba3695f77
2025/06/19 11:56:53 Benchamrk container with ID: fa51c094264c46ae7352365ecf2c94217f11a8009f332052769b7d7ba3695f77 started successfully
2025/06/19 11:56:53 Measuring first time request...
2025/06/19 11:57:01 First time request successfully measured
2025/06/19 11:57:01 Collecting container stats...
2025/06/19 11:57:01 Workload container created: aape2k/workload-generator, ID: 0bbc970360ed2ed1c6748418a5a2de0840dbdb83fe06e06102a1df3c37ae087d
2025/06/19 11:57:01 Starting workload container...
2025/06/19 11:57:32 Workload container exited with status code: 0
2025/06/19 11:57:33 Workload container created: aape2k/workload-generator, ID: 5b8a919db5b1bc05b49b59e1f09e26d225b2acf4d695f97da262566c3a7af73b
2025/06/19 11:57:33 Starting workload container...
2025/06/19 11:58:04 Workload container exited with status code: 0
2025/06/19 11:58:04 Workload container created: aape2k/workload-generator, ID: a4ec78dc67a3632200e0181c44ee05548fa7c8668cead62f13067848dea67ed6
2025/06/19 11:58:04 Starting workload container...
2025/06/19 11:58:33 Workload container exited with status code: 134
2025/06/19 11:58:33 Container with ID: 0bbc970360ed2ed1c6748418a5a2de0840dbdb83fe06e06102a1df3c37ae087d removed successfully
2025/06/19 11:58:33 Container with ID: 5b8a919db5b1bc05b49b59e1f09e26d225b2acf4d695f97da262566c3a7af73b removed successfully
2025/06/19 11:58:33 Container with ID: a4ec78dc67a3632200e0181c44ee05548fa7c8668cead62f13067848dea67ed6 removed successfully
2025/06/19 11:58:34 Stream ended.
2025/06/19 11:58:34 Error fetching top: Error response from daemon: container fa51c094264c46ae7352365ecf2c94217f11a8009f332052769b7d7ba3695f77 is not running
2025/06/19 11:58:34 Container with ID: fa51c094264c46ae7352365ecf2c94217f11a8009f332052769b7d7ba3695f77 removed successfully
2025/06/19 11:58:35 Network with ID: 37df5f401f7d6664c8fe0ff1e401c052abf997411d7dbcacee8c58d3f9795d93 removed successfully
```

In the result path should appear folder with results and current timestamp (for example: `result-2025-04-19-12:03:44`). 

The result folder structer should look like this:

```bash
result-2025-04-19-12:03:44/
|--- jfr/
     |--- result.jfr
|--- 0-create-shop-carts.lua.out
|--- 1-populate-shop-carts.lua.out
|--- 2-read-shop-carts.lua.out
|--- api.yaml
|--- container-stats.csv
|--- create-shop-carts.lua
|--- first-response.csv
|--- parse.lua
|--- populate-shop-carts.lua
|--- read-shop-carts.lua
|--- rss_info.json
```
