# Extension Guide

## How to Add a New Benchmark

Adding a new benchmark to the Serverless Benchmark Suite is designed to be straightforward and modular. 
Each benchmark is self-contained and follows a standard structure, making it easy to integrate new workloads
and ensure compatibility with the harness.

### 1. Create a New Benchmark Directory

Inside the `/benchmark` directory, create a new folder for your benchmark. The 
folder name should be unique and descriptive (e.g., `myframework-foobar`). This name
will be used as **benchmarkName** in application configuration file. 

### 2. Required Subfolders and Files

Each benchmark directory should contain a `README.md` file that provides a brief 
overview of the benchmark and step-by-step instructions for building images locally.

Another mandatory element is the `api/` subfolder. This folder contains the [OpenAPI](https://www.openapis.org/) specification, Lua scripts, sample resources, and the workload configuration file. Files from this folder are mounted into containers, so it's best practice to include only essential files here. Details on each file are provided below:

- `api.yaml` — the OpenAPI (Swagger) specification. This file must be named consistently across benchmarks and must describe the benchmark’s endpoints. It should follow certain rules to be parsed correctly by the harness. See more on these constraints in the [api.yaml section](#requirements-for-apiyaml).

- `resources/` — a subfolder that contains sample files referenced in the OpenAPI specification (`api.yaml`). More information is available in the [api.yaml section](#requirements-for-apiyaml).

- `config.json` — the workload configuration file that defines benchmark parameters.

> **Note**: `api.yaml`, `config.json` files and `resources/` folder must be named consistently across benchmarks. Your benchmark must also adhere to this convention. Any folders or files other than these may be named arbitrarily.

#### Requirements for api.yaml

For `api.yaml` to be parsed by the harness, it must follow certain rules:

The servers section must include a URL in the format `http://localhost:{port}`,
where `{port}` has a defined default value. This port will be used to connect to
the benchmarking application.

```yaml
servers:
- url: http://localhost:{port}
    variables:
        port:
          default: '8001'
          description: The port number of the server
```
The harness parses the `api.yaml` file to automatically discover endpoints and construct HTTP requests for benchmarking. It does this by looking for `examples` defined under each endpoint's `requestBody` in the OpenAPI specification. Each example is given a unique name (for example, `newClient`), which is then referenced in the workload configuration (such as in `config.json` or the harness config files).

When running a workload, the harness uses the example name to:
- Locate the corresponding endpoint (path and HTTP method) in the OpenAPI spec.
- Extract the example request body and any required headers or parameters.
- Substitute any path or query parameters with the values provided in the example or parameter definitions.
- Use the example's value (or external file, if `externalValue` is specified) as the request payload.

**Key points:**
- Each `requestBody` should define at least one named example under the `examples` field.
- The example name acts as a unique identifier for that type of request and is used in workload scripts and configuration.
- The harness expects the OpenAPI spec to be structured so that it can unambiguously map an example name to a specific endpoint and request format.

**Example snippet from `api.yaml`:**

```yaml
# example from micronaut-shopcart/api/api.yaml
...
  /:
    post:
      summary: Create a new client
      operationId: addClient
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - username
                - name
              properties:
                username:
                  type: string
                  description: The username of the client
                name:
                  type: string
                  description: The name of the client
            examples:
              newClient:
                value: '{"username": "john_doe", "name": "John Doe"}'
```

If endpoint does not require a body, you should specify empty value:

```yaml
# example from micronaut-shopcart/api/api.yaml
...
  /cart/{username}:
    get:
      summary: Get a client's shopping cart
      operationId: getCart
      parameters:
        - in: path
          name: username
          required: true
          schema:
            type: string
          example: 'john_doe'
          description: The username of the client whose cart is to be retrieved
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
            examples:
              getCartExample:
                value: ''
```

The harness also supports using files as example values.
To add a file as an example, the user should create a subfolder named `resources/` 
inside the `api/` directory and place the file there. Then, the user should reference this file using the externalValue field, as shown in the example below:

```yaml
# example from micronaut-shopcart/api/api.yaml
...
  /parse/text:
    post:
      summary: Extract text from a document
      description: Extracts plain text from a PDF or ODT document.
      requestBody:
        required: true
        content:
          application/pdf:
            schema:
              type: string
              format: binary
            examples:
              samplePDF:
                summary: Example PDF file
                externalValue: "quarkus.pdf"
```

Resources will be copied to working directory in the runtime.

#### Requirements for config.json

The `config.json` file defines the available workloads and example data for your benchmark. It is placed in the `api/` directory alongside your `api.yaml`. Each entry in the JSON array represents a workload configuration that the harness can use to run benchmarks.

A typical `config.json` entry includes:
- `name`: A unique identifier for the workload (e.g., `"pdf-requests"`).
- `exampleName`: The name of the example defined in your OpenAPI spec (under `examples:` in `api.yaml`). This links the workload to a specific example input, which can be a file referenced via `externalValue`.
- `wrkScripts`: A list of Lua script filenames (used by the harness) that define the request patterns for this workload.

> **NOTE**: Config can include one or more scripts, but in practice, multiple scripts are used only when the scripts depend on the result of each other's execution.

> **NOTE**: For more information, examine the `config.json` files in existing benchmarks.

#### WRK2 scripts

The `wrkScripts` field in your `config.json` specifies one or more Lua scripts that define the request patterns for the [WRK2](https://github.com/giltene/wrk2) benchmarking tool. These scripts allow you to simulate different types of client behavior and workloads against your API endpoints. More on WRK2 scripting can be found [here](https://github.com/giltene/wrk2/blob/master/SCRIPTING).

To simplify the integration of new benchmarks, the harness provides a parser for the OpenAPI file, which, given a unique example name, returns all necessary information for the request: path, method, headers, and example body. Example of usage is bellow:

```lua
-- example from benchmark/quarkus-tika/api/requests.lua
local OpeanApiParse = require("parser")

function init(args)
   local path, method, headers, body = OpeanApiParse.getRequestParameters(os.getenv("SAMPLE_NAME"))
   wrk.path = path
   wrk.method = method
   wrk.headers = headers
   wrk.body = body
end
```
> **Note**: In this example, the example name is received from an environment variable. This allows you to reuse the script for different examples by passing different environment variables to the container. More information about available environment variables is provided below.

#### Available environment varaibles in lua script

The following environment variables are available to your Lua WRK2 scripts:

- `OUTPUT_DIR`:  
  The path inside the container where all benchmark resources are mounted. This directory contains files such as `api.yaml`, `config.json`, example files referenced by `externalValue`, and any output files your scripts may generate.  
  **Example:** `/workspace/benchmark/quarkus-tika/api`

- `HOST_URL`:  
  The base URL of the API endpoint being benchmarked. Since benchmarking occurs inside a Docker private network, the benchmark's internal domain name is used in this URL.
  **Example:** `http://benchmark:8001`

- `SAMPLE_NAME`:  
  The name of the example defined in your OpenAPI spec (under `examples:` in `api.yaml`). This is typically used to select which example input to use for the request.  
  **Example:** `samplePDF`

You can access these variables in your Lua scripts using `os.getenv`, for example:

**For more information on how parser, environment variables and scripts can be used, refer to the example scripts provided in the benchamrks api repositories.**

### 3. Implement the Benchmark Application

- Implement your application according to the API contract defined in `api/api.yaml`.
- Ensure the application can be built and run in a container using the provided Dockerfiles.

### 4. Build and Publish Docker Images

- Build the Docker images for both JVM and native modes (if applicable).
- Push the images to a container registry (e.g., Docker Hub) if you want to use them in CI or share with others.

> **Note**: If you want to collect Java Flight Recorder (JFR) data during benchmarking, ensure that JFR is enabled in your application image and
that the resulting `.jfr` file is saved to `/app/logs/`. Files in this directory will be included in the output directory after the benchmark run.

### 5. Add Example Configuration

- Provide example harness configuration files (JSON) in the `/example-configs` directory at the project root. These should demonstrate how to run your benchmark with the harness, specifying the correct `benchmarkName`, `benchmarkImage`, and workload scripts.

### 6. Register the Benchmark

- Update the documentation (e.g., `/benchmark/README.md`) to include your new benchmark, its description, and available workloads.
- Optionally, submit a pull request if you want your benchmark included in the main repository.

### 7. Test with the Harness

- Use the harness as described in [Running & Data Collection Guide](./running-and-collection.md) to verify that your benchmark runs end-to-end, collects results, and produces the expected output.

---










