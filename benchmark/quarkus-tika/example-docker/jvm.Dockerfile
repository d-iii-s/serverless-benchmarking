# Author: Artem Baktin
FROM registry.access.redhat.com/ubi9/openjdk-21:1.21

ENV LANGUAGE='en_US:en'
USER root
RUN mkdir -p /app/logs  && chmod +w /app/logs

# We make four distinct layers so if there are application changes the library layers can be re-used
COPY target/quarkus-app/lib/ /deployments/lib/
COPY target/quarkus-app/*.jar /deployments/
COPY target/quarkus-app/app/ /deployments/app/
COPY target/quarkus-app/quarkus/ /deployments/quarkus/

EXPOSE 8004
ENV AB_JOLOKIA_OFF=""
ENTRYPOINT [ "sh", "-c", "java $HARNESS_JAVA_OPTS -Dquarkus.http.host=0.0.0.0 -Djava.util.logging.manager=org.jboss.logmanager.LogManager -jar /deployments/quarkus-run.jar" ]

