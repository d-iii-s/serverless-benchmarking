# Spring Petclinic Benchmark

A simple e-commerce shopping cart service implemented with the Micronaut framework. It exposes endpoints for creating and populating shopping carts, simulating typical CRUD operations found in serverless microservices.

## Prebuilt images on DockerHub

- JVM: `aape2k/petclinic-jvm`
- Native: `aape2k/petclinic-native`

The application image was built on `Debian 6.1.140-1 (2025-05-22) x86_64 GNU/Linux` using 
`Oracle GraalVM 21.0.7` for Java. The build environment featured an Intel(R) Xeon(R) CPU @ 2.20GHz (8 cores) and 32GB RAM.

## Example Workload Configurations

- [petclinic-native](../../example-configs/petclinic-native.json)
- [petclinic-jvm](../../example-configs/petclinic-jvm.json)

## How to build JVM and NATIVE images?

Run script to build `native binary` and `JVM` files
```bash
./build.sh
```

> **Note**: The user should be logged in to Docker.

> **Note**: `target` directory should be writable

### JVM

To build image execute following command:
```bash
docker build -f example-docker/jvm.Dockerfile -t aape2k/petclinic-jvm .
```

### Native

Before building image the executable should be build using this command:
```bash
native-image -H:Name=petclinic --bundle-apply=spring-petclinic-sources/target/spring-petclinic-3.0.0-SNAPSHOT.nib
```

To build image execute following command:
```bash
docker build -f example-docker/native.Dockerfile -t aape2k/petclinic-native .
```
> **Note**: The image tag (`label`) can be changed

## Workloads

This benchmark provides the following example workloads, as defined in [`api/config.json`](./api/config.json):

### 1. `create-read-pipeline`

- **exampleName**: `exampleNewOwner`
- **wrkScripts**:
  - `create-owner.lua`
  - `create-pet.lua`
  - `read-pet-ids.lua`
  - `create-visit.lua`
  - `read-owner.lua`

This workload simulates a pipeline where a new owner is created, a pet is added, 
pet IDs are read, a visit is created, and finally the owner is read. It is useful
 for end-to-end testing of the main CRUD operations in sequence.

### 2. `mixed-requests`

- **exampleName**: `exampleNewOwner`
- **wrkScripts**:
  - `mixed-requests.lua`

This workload issues a mix of different request types to the service, simulating a more realistic and varied usage pattern.

You can select the desired workload by setting the `benchmarkConfigName` in your harness configuration (see the example configs linked above).



