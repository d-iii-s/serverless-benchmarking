## About the Workloads

This suite includes several representative Java serverless workloads, each containerized and ready for benchmarking in both JVM and native-image modes. The benchmarks are chosen to cover a range of real-world use cases, from CRUD microservices to document processing and simple REST APIs. Each workload is available as both a JVM and a native image, enabling direct performance comparisons.

---

### Micronaut Shopcart

A simple e-commerce shopping cart service implemented with the Micronaut framework. It exposes endpoints for creating and populating shopping carts, simulating typical CRUD operations found in serverless microservices.

[View more...](./micronaut-shopcart/README.md)

---

### Spring Petclinic
 
A classic sample application for demonstrating Spring framework features, adapted for serverless benchmarking. It simulates a veterinary clinic system with CRUD operations for owners, pets, and visits.

[View more...](./spring-petclinic/README.md)

---

### Quarkus Tika

A document parsing microservice built with Quarkus and Apache Tika. It exposes endpoints for uploading and parsing documents (e.g., ODT, PDF), simulating serverless document processing workloads.

[View more...](./quarkus-tika/README.md)

> **Note**: The application source code for these benchmarks is based on the [Barista project](https://github.com/barista-benchmarks/barista). However, all load testing scripts in the `api/` directory, the Docker build scripts in `example-docker/`, and the compilation and run instructions in each workload's `README.md` were created specifically for this benchmarking suite.
