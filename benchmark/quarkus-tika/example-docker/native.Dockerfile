# Author: Artem Bakhtin
FROM registry.access.redhat.com/ubi9/openjdk-21:1.21

######################### Set up environment for POI ##########################
USER root
WORKDIR /work/
RUN chown 1001 /work \
    && chmod "g+rwX" /work \
    && chown 1001:root /work
# Shared objects to be dynamically loaded at runtime as needed,
COPY --chown=1001:root target/*.properties target/*.so /work/
COPY --chown=1001:root target/*-runner /work/application
RUN mkdir -p /app/logs

EXPOSE 8004
ENTRYPOINT ["sh", "-c", "./application $HARNESS_JAVA_OPTS -Dquarkus.http.host=0.0.0.0"]
