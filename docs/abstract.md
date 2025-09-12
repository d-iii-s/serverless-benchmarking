# Motivation

Nowadays, a growing number of software development companies are increasingly
adopting serverless computing due to its higher level of server abstraction, enabling
hassle-free development without concerns over underlying infrastructure. Serverless
computing primarily consists of Backend-as-a-Service (BaaS) and
Function-as-a-Service (FaaS). BaaS offers a unified API for third-party services,
while FaaS encapsulates server-side logic for efficient scaling and execution.
Java, a widely used language, is popular for serverless applications. However, its
traditional drawbacks in terms of execution time and resource consumption present a
problem. Native image compilation has emerged as a solution, significantly
enhancing Java's performance in serverless environments.
However, existing benchmarking suites for modern serverless frameworks lack
support for executing JVM programs and native compiled programs within the same
benchmark. This feature is critical for obtaining accurate performance comparisons.
Additionally, many existing serverless benchmarks suites do not offer a
comprehensive set of benchmarks or lack mechanisms to collect new metrics.

# Similar projects

The following projects are among the most prominent open-source serverless benchmarking suites:

---

## [Barista](https://github.com/barista-benchmarks/barista/tree/master/benchmarks)

**Overview:**

Barista is a benchmarking suite focused on cloud microservices, providing a set 
of representative workloads and a harness for evaluating distributed systems. 
It is designed to be extensible and supports a variety of microservice architectures,
primarily targeting JVM-based services.

**Advantages:**  
- Provides a diverse set of microservice benchmarks.
- Includes a harness for orchestrating and measuring distributed workloads.
- Supports containerized execution for reproducibility.

**Disadvantages:**  
- Primarily targets JVM-based microservices; does not natively support native-image
(AOT) Java applications.
- Extending with new serverless-specific workloads or metrics can require significant effort.
- Focuses more on microservices than on FaaS/serverless paradigms.

---

## [DeathStarBench](https://github.com/delimitrou/DeathStarBench)

**Overview:** 

DeathStarBench is a suite of end-to-end cloud microservices benchmarks, 
designed to model real-world applications such as social networks, media services,
and e-commerce. It is widely used in academic research for evaluating cloud systems
and microservice architectures.

**Advantages:**  
- Offers complex, realistic application benchmarks with multiple interacting services.
- Widely adopted in research for evaluating system-level performance.
- Supports containerized deployment for reproducibility.

**Disadvantages:**  
- Not specifically focused on serverless or FaaS workloads.
- Lacks support for Java native-image benchmarking.
- Adding new Java frameworks or custom metrics is non-trivial.

---

## [SPCL Serverless Benchmarks](https://github.com/spcl/serverless-benchmarks)

**Overview:**

This project provides a set of benchmarks for evaluating serverless platforms,
focusing on function-level performance and resource usage. It includes workloads
in multiple languages and targets both open-source and commercial FaaS platforms.

**Advantages:**  
- Focuses on serverless/FaaS workloads and platforms.
- Supports multiple programming languages.
- Includes resource usage and cold start measurements.

**Disadvantages:**  
- Limited support for Java, especially for comparing JVM and native-image modes.
- Less emphasis on extensibility and integration of new Java frameworks.
- Workloads are often simple functions rather than real-world applications.

---

## [ServerlessBench (SJTU-IPADS)](https://github.com/SJTU-IPADS/ServerlessBench)

**Overview:**  
ServerlessBench is a benchmark suite for serverless computing platforms, 
providing a collection of workloads and tools for evaluating FaaS systems.
It aims to cover a range of use cases and performance metrics relevant to serverless environments.

**Advantages:**  
- Designed specifically for serverless/FaaS benchmarking.
- Provides a variety of workloads and metrics.
- Supports multiple platforms and languages.

**Disadvantages:**  
- Limited focus on Java, especially on modern frameworks and native-image.
- Extending with new, complex Java workloads is not straightforward.
- May lack some real-world, containerized Java application scenarios.

---

## How the Serverless Benchmark Suite Differs and Its Advantages

While these projects have advanced the state of serverless benchmarking, 
the Serverless Benchmark Suite introduces several key differences and advantages:

**1. Unified JVM and Native Benchmarking:**  
Most existing suites, including Barista and DeathStarBench, focus on JVM-based or
interpreted workloads and do not natively support benchmarking both JVM and native-image
(AOT-compiled) Java applications within the same harness. Our suite is designed 
from the ground up to enable direct, apples-to-apples comparisons between JVM 
and native deployments, which is critical for evaluating the impact of GraalVM
native image and similar technologies.

**2. Extensible Harness and Workload Integration:**  
While other projects provide a fixed set of benchmarks or require significant 
effort to add new workloads, this suite features a modular, extensible harness.
Users can easily integrate new Java frameworks, workloads, or custom metrics with
minimal configuration, making it suitable for both research and practical performance engineering.

**3. Comprehensive Metric Collection:**  
Compared to the above projects, which often focus on throughput and latency, our
suite automatically collects a broader set of metrics—including startup time,
time-to-first-response, CPU/memory usage, and resource consumption—out of the box.
This enables more holistic analysis of serverless performance, especially for cold start scenarios and resource-constrained environments.

**4. Real-World, Containerized Java Workloads:**  
DeathStarBench and Barista include a variety of microservices, but are not 
specifically tailored to Java serverless frameworks or the unique challenges of 
Java in FaaS environments. Our suite provides ready-to-use, containerized benchmarks for major Java frameworks (Spring, Micronaut, Quarkus), each available in both JVM and native-image forms,
 reflecting real-world serverless deployment patterns.

**5. Focus on Repeatability and Practical Usability:**  
Some prior suites require complex setup or are tightly coupled to specific cloud
providers or academic testbeds. The Serverless Benchmark Suite is designed for easy
local execution using Docker, with public images and example configs, ensuring 
that results are reproducible and accessible to a wide audience.

**Summary Table:**

| Feature                                 | Serverless Benchmark Suite | Barista | DeathStarBench | SPCL Serverless Benchmarks | ServerlessBench (SJTU) |
|------------------------------------------|:-------------------------:|:-------:|:--------------:|:-------------------------:|:----------------------:|
| JVM & Native Java Support                |           ✔️              |    ❌    |       ❌        |            ❌              |          ❌            |
| Extensible Harness & Easy Extension      |           ✔️              |   Partial|     Partial     |           Partial          |         Partial        |
| Comprehensive Metric Collection          |           ✔️              |   Partial|     Partial     |           Partial          |         Partial        |
| Real-World Java Serverless Workloads     |           ✔️              |    ❌    |       ❌        |            ❌              |          ❌            |
| Containerized, Local Execution           |           ✔️              |    ✔️    |       ✔️        |            ✔️              |          ✔️            |

In summary, the Serverless Benchmark Suite fills a critical gap by enabling rigorous, extensible, and repeatable benchmarking of both JVM and native Java serverless workloads, with a focus on real-world applicability.

# Goals
The goals of the Serverless Benchmark Suite are shaped by the limitations and gaps identified in prior projects (such as Barista, DeathStarBench, and others) and are focused on enabling rigorous, extensible, and repeatable benchmarking for both JVM and native Java serverless workloads. Specifically, our objectives are:

- **Unified Benchmarking for JVM and Native Java:**  
  Provide a single harness capable of executing and measuring both JVM-based and native-image (AOT-compiled) Java workloads, enabling direct, apples-to-apples comparisons.

- **Extensibility and Ease of Integration:**  
  Design a modular harness that allows users to easily add new Java frameworks, workloads, and custom metrics with minimal configuration, supporting both research and practical engineering needs.

- **Comprehensive Metric Collection:**  
  Automatically collect a broad set of performance metrics—including startup time, time-to-first-response, throughput, CPU/memory usage, and resource consumption—to enable holistic analysis, especially for cold start and resource-constrained scenarios.

- **Real-World, Containerized Java Workloads:**  
  Include ready-to-use, containerized benchmarks for major Java frameworks (Spring, Micronaut, Quarkus), each available in both JVM and native-image forms, reflecting real-world serverless deployment patterns.

- **Repeatability and Practical Usability:**  
  Ensure that the suite is easy to run locally using Docker, with public images and example configs, so that results are reproducible and accessible to a wide audience without requiring complex setup or cloud-specific infrastructure.

# Benchmarks and metrics

The benchmarking suite is designed to deliver raw data from fundamental
metrics. Among the fundamentals metrics are:

- Throughput
- Time for (first) request
- Resource usage
 
Throughput metric provides insights about the maximum load that software can
handle. The time-to-first-request metric is essential for assessing scalability, as it
indicates the duration required to initiate a new serverless instance and prepare it for
processing incoming requests. Together with throughput this is a traditional metric for
client-server workloads. Collecting other resource usage metrics is crucial for
detailed analysis of the whole stack of used software: application workload of the
benchmark, used framework and the host platform (JVM or native application).
