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
- **What's included:**
  - Benchmark harness for orchestrating and measuring serverless workloads
  - Sample workloads for three major Java frameworks
  - Data collection and reporting of key metrics
- **What's not included:**
  - Built-in analysis tools
- **Intended audience:**
  - Researchers, engineers, and performance testers interested in serverless Java performance

## Features
- **Harness:** Unified tool for launching, managing, and measuring serverless workloads
- **Data Collection:** Automatic collection of startup time, throughput, resource usage, and other metrics
- **Metrics:** Startup time, time to first response, throughput, CPU/memory usage, and more
- **Sample Benchmarks:** Ready-to-use benchmarks for Spring, Micronaut, Quarkus, and more (see [Benchmark Details](benchmark/README.md) for specifics)
- **Easy Extension:** Add your own benchmark with minimal effort (see [Extension Guide](docs/extension-guide.md))

## Project **Structure**
- `/benchmark` – Sample bencharmks ([details](benchmark/README.md))
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
  See [Benhcmarks Details](benchmark/README.md) for descriptions of the sample benchmarks and their endpoints.

- **For technical architecture and development:**  
  Visit [Development & Architecture](docs/development-architecture.md) for an overview of the harness, workflow, and extension points.

- **Demo:**
  Visit [Sample demo README](demo/README.md)
