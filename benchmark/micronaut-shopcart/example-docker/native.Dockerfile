# Author: Artem Bakhtin
FROM registry.access.redhat.com/ubi9/openjdk-21:1.21
WORKDIR /app
COPY target/shopcart-0.3.10.output/default/shopcart .
RUN mkdir -p /app/logs
ENTRYPOINT ["sh","-c", "./shopcart $HARNESS_JAVA_OPTS"]