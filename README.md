# go-monitor-demo

这是一个最小可观测 Go 服务示例，用来验证下面这条链路：

`Jenkins -> Docker Buildx -> Go 服务 -> Prometheus -> Fluent Bit -> Loki -> Grafana`

## 功能

- `GET /`
- `GET /healthz`
- `GET /work?delay_ms=800&status=500`
- `GET /metrics`
- 周期性写 JSON 日志到 `/app/logs/go-monitor-demo.log`
- 提供 Prometheus 指标

## 端口说明

- 业务接口：`18080`
- metrics 接口：`12112`

之前文档把 `8080/2112`、容器内端口、宿主机端口混在一起写了，这本身就容易把人带沟里。

- 容器内统一：`18080/12112`
- 本地直跑统一：`18080/12112`
- Docker / Compose / Jenkins 对外映射统一：`28080/22112`

## 本地直接运行

### PowerShell

```powershell
go mod tidy
New-Item -ItemType Directory -Force runtime-logs | Out-Null
$env:APP_PORT="18080"
$env:METRICS_PORT="12112"
$env:APP_LOG_PATH="./runtime-logs/go-monitor-demo.log"
go run .
```

你之前如果直接照着旧文档写下面这种命令，在 PowerShell 里就是错的：

```bash
APP_PORT=18080 METRICS_PORT=12112 APP_LOG_PATH=./runtime-logs/go-monitor-demo.log go run .
```

这是 Bash 写法，不是 PowerShell 写法。

### Bash

```bash
go mod tidy
mkdir -p ./runtime-logs
APP_PORT=18080 METRICS_PORT=12112 APP_LOG_PATH=./runtime-logs/go-monitor-demo.log go run .
```

默认日志文件：

```text
/app/logs/go-monitor-demo.log
```

如果你本机运行不想写到容器目录，就把日志写到当前目录下的 `runtime-logs/go-monitor-demo.log`。

## Docker 运行

### PowerShell

```powershell
docker build -t go-monitor-demo:latest .
New-Item -ItemType Directory -Force runtime-logs | Out-Null
docker run -d `
  --name go-monitor-demo `
  -e APP_PORT=18080 `
  -e METRICS_PORT=12112 `
  -p 28080:18080 `
  -p 22112:12112 `
  -v "${PWD}/runtime-logs:/app/logs" `
  go-monitor-demo:latest
```

### Bash

```bash
docker build -t go-monitor-demo:latest .
docker run -d \
  --name go-monitor-demo \
  -e APP_PORT=18080 \
  -e METRICS_PORT=12112 \
  -p 28080:18080 \
  -p 22112:12112 \
  -v $(pwd)/runtime-logs:/app/logs \
  go-monitor-demo:latest
```

## Docker Compose 联调

```bash
docker compose up -d --build
```

访问：

- `http://127.0.0.1:28080/healthz`
- `http://127.0.0.1:28080/work?delay_ms=500`
- `http://127.0.0.1:22112/metrics`
- `http://127.0.0.1:9090`
- `http://127.0.0.1:3000`

Grafana 默认账号密码一般是：

```text
admin / admin
```

## 关键指标

- `go_demo_http_requests_total`
- `go_demo_heartbeat_total`
- `go_demo_process_up`

## 日志查询

Grafana Explore 中查询：

```text
{job="go-monitor-demo"}
```
