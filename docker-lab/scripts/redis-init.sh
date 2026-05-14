#!/bin/sh
set -eu

NODES="
redis-1:7000
redis-2:7001
redis-3:7002
redis-4:7003
redis-5:7004
redis-6:7005
"

echo "[redis-init] waiting for redis nodes"
for node in ${NODES}; do
    host="${node%:*}"
    port="${node#*:}"
    until redis-cli -h "${host}" -p "${port}" ping >/dev/null 2>&1; do
        sleep 2
    done
done

if redis-cli -h redis-1 -p 7000 cluster info 2>/dev/null | grep -q 'cluster_state:ok'; then
    echo "[redis-init] cluster already initialized"
    exit 0
fi

echo "[redis-init] creating redis cluster"
yes yes | redis-cli --cluster create \
    redis-1:7000 \
    redis-2:7001 \
    redis-3:7002 \
    redis-4:7003 \
    redis-5:7004 \
    redis-6:7005 \
    --cluster-replicas 1

echo "[redis-init] done"
