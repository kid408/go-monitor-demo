#!/bin/sh
set -eu

DB="${CLICKHOUSE_DATABASE:-app}"
TABLE="${CLICKHOUSE_TABLE:-session_events}"
USER_NAME="${CLICKHOUSE_USER:-lab}"
PASSWORD="${CLICKHOUSE_PASSWORD:-lab123}"

echo "[clickhouse-init] waiting for clickhouse-1"
until clickhouse-client --host clickhouse-1 --user "${USER_NAME}" --password "${PASSWORD}" --query "SELECT 1" >/dev/null 2>&1; do
    sleep 2
done

echo "[clickhouse-init] waiting for clickhouse-2"
until clickhouse-client --host clickhouse-2 --user "${USER_NAME}" --password "${PASSWORD}" --query "SELECT 1" >/dev/null 2>&1; do
    sleep 2
done

echo "[clickhouse-init] creating cluster schema"
clickhouse-client --host clickhouse-1 --user "${USER_NAME}" --password "${PASSWORD}" --multiquery <<SQL
CREATE DATABASE IF NOT EXISTS ${DB} ON CLUSTER lab_cluster;

CREATE TABLE IF NOT EXISTS ${DB}.${TABLE} ON CLUSTER lab_cluster
(
    event_time DateTime64(3, 'Asia/Shanghai'),
    consumed_at DateTime64(3, 'Asia/Shanghai'),
    event_id String,
    session_id String,
    client_id String,
    user_id UInt64,
    device_id String,
    action LowCardinality(String),
    payload String,
    gateway_id String,
    worker_id String,
    snapshot_object_key String,
    snapshot_exists UInt8,
    snapshot_size_bytes Int64,
    snapshot_payload String,
    kafka_partition Int32,
    kafka_offset Int64,
    kafka_timestamp DateTime64(3, 'Asia/Shanghai')
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/${DB}.${TABLE}', '{replica}')
ORDER BY (event_time, session_id, event_id);

CREATE TABLE IF NOT EXISTS ${DB}.${TABLE}_all ON CLUSTER lab_cluster
AS ${DB}.${TABLE}
ENGINE = Distributed(lab_cluster, ${DB}, ${TABLE}, cityHash64(event_id));
SQL

echo "[clickhouse-init] done"
