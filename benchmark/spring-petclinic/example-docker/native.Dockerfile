# Artem Bakhtin
FROM registry.access.redhat.com/ubi9/openjdk-21:1.21
WORKDIR /app
COPY spring-petclinic-sources/target/spring-petclinic-3.0.0-SNAPSHOT.output/default/spring-petclinic .
RUN mkdir -p /app/logs
ENTRYPOINT ["sh", "-c", "./spring-petclinic $HARNESS_JAVA_OPTS"]
