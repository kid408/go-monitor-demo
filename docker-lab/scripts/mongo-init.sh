#!/bin/sh
set -eu

echo "[mongo-init] waiting for mongo-1"
until mongosh --quiet --host mongo-1:27017 --eval "db.adminCommand({ ping: 1 }).ok" >/dev/null 2>&1; do
    sleep 2
done

echo "[mongo-init] waiting for mongo-2"
until mongosh --quiet --host mongo-2:27017 --eval "db.adminCommand({ ping: 1 }).ok" >/dev/null 2>&1; do
    sleep 2
done

echo "[mongo-init] waiting for mongo-3"
until mongosh --quiet --host mongo-3:27017 --eval "db.adminCommand({ ping: 1 }).ok" >/dev/null 2>&1; do
    sleep 2
done

echo "[mongo-init] initiating replica set"
exec mongosh --quiet --host mongo-1:27017 /scripts/mongo-init.js
