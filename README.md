# Serverless Benchmark Suite

## Introduction

The **Serverless Benchmark Suite** is a comprehensive framework designed to evaluate the performance of serverless and containerized Java workloads. At its core is a powerful **harness** that automates the process of launching, managing, and measuring serverless applications under realistic workloads. The suite provides a standardized environment for running benchmarks, collecting key metrics (such as startup time, throughput, and resource usage), and ensuring repeatable, reliable results across different frameworks and deployment modes. By aggregating a variety of sample workloads and offering easy extensibility, the Serverless Benchmark Suite enables researchers and engineers to systematically compare serverless platforms and optimize application performance.

## Goals

The primary goal of this project is to provide a comprehensive, extensible suite for benchmarking serverless Java workloads. Specifically, we aim to:

- Design and implement a robust, extensible harness for executing and measuring serverless workloads in both JVM and native (AOT) modes.
- Integrate a diverse set of real-world benchmarks, covering multiple Java frameworks and typical serverless use cases.
- Enable easy addition of new workloads and frameworks by users.
- Collect and report key performance metrics, such as startup time, throughput, and resource usage, in a consistent and repeatable manner.

## Scope

For more information on project motivation and goals, see the [Abstract](docs/abstract.md).

For a detailed requirements analysis, see [Requirements Analysis](docs/requirements_analysis.md).

- **What's included:**
  - Benchmark harness for orchestrating and measuring serverless workloads
  - Sample workloads for three major Java frameworks
  - Data collection and reporting of key metrics
- **What's not included:**
  - Built-in analysis tools
- **Intended audience:**
  - Researchers, engineers, and performance testers interested in serverless Java performance

### **Use Case Example**

**Scenario:** A development team is evaluating whether to deploy their Spring Boot microservice as a traditional JVM application or as a GraalVM native image in a serverless environment. They need to understand the trade-offs between startup time, memory usage, and throughput.

**Solution:** Using the Serverless Benchmark Suite, the team can:
1. Configure the harness to run their Spring Boot application in both JVM and native modes
2. Execute identical workloads against both versions using the same API endpoints
3. Collect comprehensive metrics including:
   - Cold start time (time to first request)
   - Memory consumption during startup and steady-state operation
   - Throughput under various load conditions
   - CPU utilization patterns
4. Compare results to make an informed decision about deployment strategy

**Outcome:** The team gains quantitative data showing that while the native image has a 3x faster startup time and 40% lower memory usage, the JVM version provides 15% higher throughput under sustained load, enabling them to choose the optimal deployment strategy based on their specific requirements.



## Features
- **Harness:** Unified tool for launching, managing, and measuring serverless workloads
- **Data Collection:** Automatic collection of startup time, throughput, resource usage, and other metrics
- **Metrics:** Startup time, time to first response, throughput, CPU/memory usage, and more
- **Sample Benchmarks:** Ready-to-use benchmarks for Spring, Micronaut, Quarkus, and more (see [Benchmark Details](benchmark/README.md) for specifics)
- **Easy Extension:** Add your own benchmark with minimal effort (see [Extension Guide](docs/extension-guide.md))

## Project **Structure**
- `/benchmark` – Sample benchmarks ([details](benchmark/README.md))
- `/harness` – Benchmark harness source code
- `/demo` – Demo and sample analysis ([sample demo](demo/report.pdf))

## Where to Find More Information

- **To run benchmarks and collect data:**  
  See the [Running & Data Collection Guide](docs/running-and-collection.md) for step-by-step instructions on executing workloads and gathering results.

- **To add or extend benchmarks:**  
  Refer to the [Extension Guide](docs/extension-guide.md) for details on integrating your own workloads or extending the suite.

- **For project motivation and goals:**  
  Read the [Abstract](docs/abstract.md) for background, motivation, and a summary of the suite's objectives.

- **For details on included benchmarks:**  
  See [Benchmarks Details](benchmark/README.md) for descriptions of the sample benchmarks and their endpoints.

- **For technical architecture and development:**  
  Visit [Development & Architecture](docs/development-architecture.md) for an overview of the harness, workflow, and extension points.

- **Demo:**
  Visit [Sample demo README](demo/README.md)

## Technologies Used

This project leverages a diverse technology stack carefully chosen to address the specific requirements of serverless Java benchmarking. Each technology serves a distinct purpose in the overall architecture:

### **Core Technologies**

#### **Go (Golang) - Benchmark Harness**
- **Version:** Go 1.23.2
- **Purpose:** The main orchestration and management system
- **Why Go:** 
  - Excellent concurrency support
  - Strong Docker integration capabilities
  - Fast compilation and execution for real-time monitoring
  - Cross-platform compatibility for diverse deployment environments
  - Minimal runtime overhead for accurate performance measurements

#### **Docker - Containerization**
- **Purpose:** Standardized deployment and isolation of benchmark workloads
- **Why Docker:**
  - Ensures consistent runtime environments across different systems
  - Enables easy deployment of both JVM and native Java applications
  - Provides resource isolation for accurate performance measurements
  - Simplifies dependency management and environment setup

### **Java Frameworks & Runtimes**

#### **Spring Boot 3.0.4**
- **Purpose:** Traditional JVM-based serverless workload
- **Why Spring Boot:**
  - Integrated from an existing project

#### **Micronaut 4.3.1**
- **Purpose:** Lightweight, cloud-native Java framework
- **Why Micronaut:**
  - Integrated from an existing project

#### **Quarkus 3.7.1**
- **Purpose:** Kubernetes-native Java framework optimized for containers
- **Why Quarkus:**
  - Integrated from an existing project

#### **GraalVM Native Image**
- **Purpose:** Ahead-of-Time (AOT) compilation for native executables
- **Why GraalVM:**
  - The most widely used AOT compiler for Java

### **Workload Generation & Testing**

#### **wrk2 - HTTP Load Testing**
- **Purpose:** High-performance HTTP benchmarking tool
- **Why wrk2:**
  - Generates consistent, repeatable load patterns
  - Low overhead for accurate performance measurements
  - Supports custom Lua scripts for complex request patterns
  - Provides detailed latency statistics and percentiles

#### **Lua Scripting**
- **Purpose:** Custom request generation and API testing
- **Why Lua:**
  - Lightweight scripting language with minimal overhead
  - Excellent integration with wrk2 for dynamic request generation
  - Easy to write and maintain test scenarios

#### **OpenAPI Specification**
- **Purpose:** API contract definition and request generation
- **Why OpenAPI:**
  - Standardized API documentation format
  - Ensures consistent API testing across different frameworks

### **Data Analysis & Visualization**

#### **Python Ecosystem**
- **Libraries:** pandas, matplotlib, numpy, jupyter
- **Purpose:** Data analysis, visualization, and reporting
- **Why Python:**
  - Rich ecosystem for data analysis and scientific computing
  - Excellent visualization capabilities with matplotlib
  - Jupyter notebooks for interactive analysis and documentation
  - pandas for efficient data manipulation and aggregation
  - Easy integration with various data formats (CSV, JSON, etc.)

#### **Jupyter Notebooks**
- **Purpose:** Interactive data analysis and report generation
- **Why Jupyter:**
  - Combines code, analysis, and documentation in one environment
  - Enables reproducible research and analysis
  - Interactive visualization capabilities
  - Support for multiple output formats (HTML, PDF, etc.)

### **Build & Dependency Management**

#### **Maven**
- **Purpose:** Java project build and dependency management
- **Why Maven:**
  - Standard build tool for Java projects

#### **Docker Multi-stage Builds**
- **Purpose:** Optimized container image creation
- **Why Multi-stage:**
  - Reduces final image size by excluding build dependencies
  - Enables different base images for build and runtime
  - Supports both JVM and native compilation in same Dockerfile
  - Improves security by minimizing attack surface
  - Faster deployment due to smaller image sizes

### **Monitoring & Observability**

#### **Java Flight Recorder (JFR)**
- **Purpose:** Low-overhead performance monitoring
- **Why JFR:**
  - Minimal performance impact on application execution
  - Detailed insights into JVM behavior and performance
  - Built-in support in modern JVMs and GraalVM
  - Rich profiling data for analysis
  - Standard tool for Java performance analysis

### **Architecture Decisions**

The technology choices reflect several key architectural principles:

1. **Performance First:** Every technology is selected to minimize overhead and provide accurate measurements
2. **Reproducibility:** Docker and standardized build processes ensure consistent results across environments
3. **Extensibility:** Modular design allows easy addition of new frameworks and workloads
4. **Industry Standards:** Use of widely-adopted tools and frameworks for broad applicability
