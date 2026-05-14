#!/bin/sh
set -eu

echo "[minio-init] waiting for minio"
until mc alias set lab http://minio:9000 "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}" >/dev/null 2>&1; do
    sleep 2
done

echo "[minio-init] creating bucket ${MINIO_BUCKET}"
mc mb -p "lab/${MINIO_BUCKET}" >/dev/null 2>&1 || true
mc anonymous set private "lab/${MINIO_BUCKET}" >/dev/null 2>&1 || true

echo "[minio-init] done"
