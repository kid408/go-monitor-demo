#!/bin/sh
set -eu

BOOTSTRAP_SERVER="${KAFKA_BOOTSTRAP_SERVER:-kafka-1:9092}"
TOPIC="${KAFKA_INIT_TOPIC:-user-session-events}"

echo "[kafka-init] waiting for broker ${BOOTSTRAP_SERVER}"
until /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server "${BOOTSTRAP_SERVER}" >/dev/null 2>&1; do
    sleep 2
done

echo "[kafka-init] creating topic ${TOPIC}"
/opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "${BOOTSTRAP_SERVER}" \
    --create \
    --if-not-exists \
    --topic "${TOPIC}" \
    --partitions 3 \
    --replication-factor 3

echo "[kafka-init] done"
