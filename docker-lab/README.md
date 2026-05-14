# docker-lab

这是单机 Docker 学习环境，不是生产级高可用环境。

你现在只有一台机器，所以要学的是：

1. 组件的集群形态和配置方式
2. 服务发现、复制、分片、选主、初始化流程
3. 你的 demo 服务如何接入这些中间件

不要骗自己说这是“真高可用”。单机挂了，所有容器一起死。

## 目录说明

- `compose.yaml`：统一入口，按 profile 拆分组件
- `.env.example`：环境变量模板
- `.env`：本地默认环境变量
- `nginx/`：Nginx 配置
- `scripts/`：Kafka/Redis/Mongo/MinIO/ClickHouse 初始化脚本
- `clickhouse/`：Keeper 和 ClickHouse 配置

## 组件拓扑

- `nginx`：单实例，后续给 `gateway` 做反向代理
- `kafka-1/2/3`：3 节点 KRaft 集群，单机 combined mode
- `redis-1..6`：6 节点 Redis Cluster，3 主 3 从
- `mongo-1..3`：3 节点 Mongo Replica Set
- `minio`：单节点多盘，4 个卷
- `keeper-1..3`：3 节点 ClickHouse Keeper
- `clickhouse-1/2`：1 分片 2 副本 ClickHouse

## 先决条件

1. 默认直接使用现成的 `.env`。只有你想自定义时，才参考 `.env.example`。

2. 如果你重建过 `.env`，必须重新生成 Kafka Cluster ID，写回 `.env`：

```bash
docker run --rm apache/kafka:4.2.0 /opt/kafka/bin/kafka-storage.sh random-uuid
```

3. 如果你要把 demo 服务也并进 Compose：

- 最简单：把 `go-gateway-demo`、`go-worker-demo`、`client-demo`、`event-consumer-demo` 放到 `docker-lab` 的同级目录
- 或者直接在 `.env` 里把 `APP_SOURCE_ROOT` 改成源码所在绝对路径，例如：

```text
APP_SOURCE_ROOT=/mnt/d/jianli/zhishichubei/tool
```

4. 如果端口冲突，就改 `.env`，不要硬顶。

## 启动顺序

### 1. Kafka

```bash
docker compose --profile kafka up -d
docker compose logs -f kafka-init
```

检查：

```bash
docker compose exec kafka-1 /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka-1:9092 --list
```

### 2. Redis Cluster

```bash
docker compose --profile redis up -d
docker compose logs -f redis-init
```

检查：

```bash
docker compose exec redis-1 redis-cli -p 7000 cluster info
docker compose exec redis-1 redis-cli -c -p 7000 cluster nodes
```

### 3. Mongo Replica Set

```bash
docker compose --profile mongo up -d
docker compose logs -f mongo-init
```

检查：

```bash
docker compose exec mongo-1 mongosh --quiet --eval "rs.status().ok"
```

连接串：

```text
mongodb://mongo-1:27017,mongo-2:27017,mongo-3:27017/?replicaSet=rs0
```

### 4. MinIO

```bash
docker compose --profile minio up -d
docker compose logs -f minio-init
```

访问：

- API: `http://127.0.0.1:${MINIO_API_HOST_PORT}`
- Console: `http://127.0.0.1:${MINIO_CONSOLE_HOST_PORT}`

### 5. ClickHouse

```bash
docker compose --profile clickhouse up -d
docker compose logs -f clickhouse-init
```

检查：

```bash
docker compose exec clickhouse-1 clickhouse-client --query "SELECT hostName()"
docker compose exec clickhouse-1 clickhouse-client --query "SELECT * FROM system.clusters WHERE cluster='lab_cluster'"
```

访问：

- HTTP: `http://127.0.0.1:${CLICKHOUSE_HTTP_HOST_PORT}`
- Native: `127.0.0.1:${CLICKHOUSE_TCP_HOST_PORT}`

### 6. Nginx

```bash
docker compose --profile nginx up -d
curl http://127.0.0.1:${NGINX_HTTP_HOST_PORT}/healthz
```

当前这个 `nginx` 还是轻量占位版，用来做最小烟测。

## 接入 demo 服务

确保 `APP_SOURCE_ROOT` 指到包含以下目录的源码根：

- `go-gateway-demo`
- `go-worker-demo`
- `client-demo`
- `event-consumer-demo`

然后执行：

```bash
docker compose --profile demo build
docker compose --profile demo up -d
```

这组 demo 的流量链路是：

`client-demo -> nginx-gateway -> gateway-1/2 -> worker-1/2 -> Kafka / MinIO -> event-consumer-demo -> ClickHouse`

这里我已经把 `gateway` 和 `worker` 改成支持静态 peer 地址，不再强依赖 Consul。

## 和现有 demo 的关系

你有两种用法：

1. 推荐：直接用 `compose --profile demo` 启动这 4 个 demo
2. 临时办法：把现有外部容器手工接入 `docker-lab-net`

```bash
docker network connect docker-lab-net <container-name>
```

临时办法很脆，重建容器就丢，不建议长期用。

## 推荐接入地址

- Kafka：`kafka-1:9092,kafka-2:9092,kafka-3:9092`
- Redis Cluster：`redis-1:7000` 作为入口节点
- Mongo Replica Set：`mongodb://mongo-1:27017,mongo-2:27017,mongo-3:27017/?replicaSet=rs0`
- MinIO：`minio:9000`
- ClickHouse：`http://clickhouse-1:8123`
- ClickHouse 用户：`${CLICKHOUSE_USER}`
- ClickHouse 密码：`${CLICKHOUSE_PASSWORD}`

## 常用排错

看全部容器：

```bash
docker compose ps
```

看某个组件日志：

```bash
docker compose logs -f kafka-1
docker compose logs -f redis-init
docker compose logs -f mongo-init
docker compose logs -f clickhouse-init
```

完全清空实验环境：

```bash
docker compose down -v
```

这会删除所有实验数据卷，不要手滑。

## 资料来源

- Docker Compose profiles: https://docs.docker.com/compose/how-tos/profiles/
- Apache Kafka Docker image / KRaft: https://kafka.apache.org/42/getting-started/docker/
- Redis Cluster: https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/
- MongoDB Replica Set: https://www.mongodb.com/docs/manual/tutorial/deploy-replica-set/
- MinIO single-node multi-drive: https://min.io/docs/minio/container/operations/install-deploy-manage/deploy-minio-single-node-multi-drive.html
- ClickHouse Keeper: https://clickhouse.com/clickhouse/keeper
- NGINX reverse proxy: https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/
