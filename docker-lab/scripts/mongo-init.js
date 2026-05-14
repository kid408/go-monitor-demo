const replicaSet = process.env.MONGO_REPLICA_SET || "rs0";

try {
  const status = rs.status();
  if (status.ok === 1) {
    print("[mongo-init] replica set already initialized");
    quit(0);
  }
} catch (err) {
  // continue to initiate
}

rs.initiate({
  _id: replicaSet,
  members: [
    { _id: 0, host: "mongo-1:27017", priority: 2 },
    { _id: 1, host: "mongo-2:27017", priority: 1 },
    { _id: 2, host: "mongo-3:27017", priority: 1 }
  ]
});

let ready = false;
for (let i = 0; i < 30; i += 1) {
  try {
    const status = rs.status();
    if (status.ok === 1) {
      ready = true;
      break;
    }
  } catch (err) {
    // retry
  }
  sleep(2000);
}

if (!ready) {
  throw new Error("replica set did not become ready");
}

print("[mongo-init] replica set initialized");
