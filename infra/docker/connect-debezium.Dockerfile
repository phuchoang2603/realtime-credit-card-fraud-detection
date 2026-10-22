FROM quay.io/strimzi/kafka:1.2.0-kafka-4.3.1

ARG DEBEZIUM_VERSION=3.7.0.Final
ARG DEBEZIUM_POSTGRES_SHA512=dc5334e087c7405f5b423e77373c60a89a3d7538ad38016f3959c431ad7078361ab95300a7e926c50621e7aa8375b7f4f3c5ecec4fe04ac484804d325009b6c0

USER root
RUN mkdir -p /opt/kafka/plugins/debezium-connector-postgres && \
  curl -fsSL -o /tmp/debezium-postgres.tar.gz \
    "https://repo1.maven.org/maven2/io/debezium/debezium-connector-postgres/${DEBEZIUM_VERSION}/debezium-connector-postgres-${DEBEZIUM_VERSION}-plugin.tar.gz" && \
  echo "${DEBEZIUM_POSTGRES_SHA512}  /tmp/debezium-postgres.tar.gz" | sha512sum -c - && \
  tar -xzf /tmp/debezium-postgres.tar.gz -C /opt/kafka/plugins/debezium-connector-postgres --strip-components=1 && \
  rm /tmp/debezium-postgres.tar.gz && \
  chown -R 1001:0 /opt/kafka/plugins/debezium-connector-postgres

USER 1001
