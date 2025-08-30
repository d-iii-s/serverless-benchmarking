# Artem Bakhtin
FROM registry.access.redhat.com/ubi9/openjdk-21:1.21
WORKDIR /app
COPY spring-petclinic-sources/target/*.jar app.jar
RUN mkdir -p /app/logs
ENTRYPOINT ["sh", "-c", "java $HARNESS_JAVA_OPTS -jar app.jar"]
