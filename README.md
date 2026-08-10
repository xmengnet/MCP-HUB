# MCP Hub

🚀 可扩展的 MCP (Model Context Protocol) 服务框架，轻松构建和托管多个 MCP 服务。

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ 特性

- 🔌 **即插即用** - 服务独立为二进制，通过 stdio 代理接入，零代码耦合
- 🔒 **沙箱安全隔离** - 每个服务可独立配置环境变量白名单、网络控制、文件系统权限
- 💻 **代码执行沙箱** - Docker 容器隔离的多语言执行环境（Python/Node.js/Shell），自动捕获图表
- 🌐 **代理模式** - 通过配置文件接入外部 MCP Server，零编码
- 🛤️ **路径路由** - 每个服务独立路径，互不干扰
- 🔐 **API Key 认证** - 开箱即用的安全认证
- 📝 **请求日志** - 内置请求日志中间件
- 🐳 **Docker 支持** - 一键容器化部署，多架构镜像 (amd64 + arm64)
- 🔄 **CI/CD 自动构建** - 每次提交自动构建测试，打 tag 自动发布到 GHCR
- 📊 **Prometheus 观测** - 内置指标采集，支持 `/metrics` 端点
- 📈 **Grafana 面板** - 提供开箱即用的观测仪表盘模板

## 📊 观测与监控

启用 Prometheus 指标后，可在主端口 `/metrics` 路径获取指标数据。

### 快速启用

```yaml
# config.yaml
prometheus:
  enabled: true
```

### 配置认证（可选）

```yaml
prometheus:
  enabled: true
  # 方式一：Bearer Token（推荐）
  token: "your-prometheus-token"
  # 方式二：Basic Auth
  # basic_user: "prometheus"
  # basic_pass: "secret"
```

留空则无认证，适合内网使用。

### Prometheus 抓取配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'mcp-hub'
    scrape_interval: 15s
    metrics_path: /metrics
    static_configs:
      - targets: ['localhost:8080']

  # 如果配置了 Bearer Token 认证
  - job_name: 'mcp-hub-auth'
    scrape_interval: 15s
    metrics_path: /metrics
    authorization:
      credentials: 'your-prometheus-token'
    static_configs:
      - targets: ['localhost:8080']

  # 如果配置了 Basic Auth 认证
  - job_name: 'mcp-hub-basic'
    scrape_interval: 15s
    metrics_path: /metrics
    basic_auth:
      username: 'prometheus'
      password: 'secret'
    static_configs:
      - targets: ['localhost:8080']
```

### 可用指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `mcp_http_requests_total` | Counter | `service`, `method`, `status_code`, `remote_addr`, `user_agent` | HTTP 请求总数（含来源 IP 和客户端标识） |
| `mcp_http_request_duration_seconds` | Histogram | `service`, `method` | 请求延迟分布 |
| `mcp_http_requests_in_flight` | Gauge | `service` | 当前并发请求数 |
| `mcp_http_response_size_bytes` | Histogram | `service`, `method` | 响应体大小分布 |
| `mcp_tool_calls_total` | Counter | `service`, `tool`, `status` | 工具调用总数（ok/error） |
| `mcp_tool_call_duration_seconds` | Histogram | `service`, `tool` | 工具调用耗时分布 |
| `mcp_service_up` | Gauge | `service` | 服务在线状态（1=在线，0=离线） |
| `go_*` | 多种 | — | Go 运行时指标（内存、协程、GC 等） |
| `process_*` | 多种 | — | 进程资源指标 |

### Grafana 仪表盘

项目提供了开箱即用的 Grafana 仪表盘模板，位于 `grafana/dashboard.json`。

**导入方式：**

1. 打开 Grafana Web UI → **Connections** → **Data Sources** → 添加 Prometheus 数据源
2. 左侧菜单 → **Dashboards** → **New** → **Import**
3. 上传 `grafana/dashboard.json` 或粘贴文件内容
4. 选择 Prometheus 数据源 → **Import**

仪表盘包含以下面板：

| 面板 | 说明 |
|------|------|
| 服务健康状态 | 各 MCP 代理服务的在线/离线状态 |
| HTTP 请求速率 | 每秒请求数（按服务划分） |
| HTTP 请求延迟 (P50/P95/P99) | 请求延迟百分位分布 |
| HTTP 状态码分布 | 各状态码的请求速率 |
| 并发请求数 | 当前正在处理的请求数 |
| 响应体大小分布 | 响应体大小 P50/P95/P99 |
| MCP 工具调用速率 | 每秒工具调用数（按服务/工具划分） |
| MCP 工具调用错误率 | 工具调用错误比例 |
| MCP 工具调用延迟 (P95) | 工具调用 P95 延迟 |
| Go 堆内存使用 | 堆内存分配情况 |
| Go 协程数 | 当前 Goroutine 数量 |
| Go GC 暂停时间 | 垃圾回收暂停时间 |
| Top 来源 IP | 请求量最多的客户端 IP 地址 |
| Top 客户端标识 | 请求量最多的 User-Agent |

### 验证指标

```bash
# 直接访问（无认证）
curl http://localhost:8080/metrics

# Bearer Token 认证
curl -H "Authorization: Bearer your-token" http://localhost:8080/metrics

# Basic Auth 认证
curl -u prometheus:secret http://localhost:8080/metrics
```

### 配置字段

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `prometheus.enabled` | 是否启用指标采集 | `false` |
| `prometheus.token` | Bearer Token 认证凭证 | 空（无认证） |
| `prometheus.basic_user` | Basic Auth 用户名 | 空 |
| `prometheus.basic_pass` | Basic Auth 密码 | 空 |

---

## 🚢 部署注意事项

### Docker 部署对指标的影响

在 Docker 中运行时，`mcp_http_requests_total` 的 `remote_addr` 标签会显示 Docker 网关 IP（如 `172.17.0.1`），而非真实客户端 IP。这是 Docker 网络 NAT 的正常行为，不影响其他指标（延迟、状态码、工具调用等）的准确性。

**如果需要获取真实客户端 IP**，建议在 Docker 前面加一层反向代理（如 Nginx），通过 `X-Forwarded-For` 头传递真实 IP。

### Nginx 反向代理配置

由于 MCP Hub 使用 SSE（Server-Sent Events）进行流式通信，Nginx 反向代理需要特殊配置：

```nginx
upstream mcp-hub {
    server 127.0.0.1:8080;
    keepalive 64;
}

server {
    listen 80;
    server_name mcp.example.com;

    # MCP Hub 主服务
    location / {
        proxy_pass http://mcp-hub;

        # SSE 必需：禁用缓冲，保证流式传输
        proxy_buffering off;
        proxy_cache off;

        # 传递真实客户端 IP
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Host $host;

        # SSE 长连接超时设置
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;

        # HTTP 版本
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        # 请求体大小限制（代码执行可能上传较大内容）
        client_max_body_size 10m;
    }

    # 指标路径（如需独立控制）
    location /metrics {
        proxy_pass http://mcp-hub;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $host;
    }
}
```

**关键配置说明：**

| 配置项 | 说明 |
|--------|------|
| `proxy_buffering off` | **必须**。SSE 依赖流式响应，Nginx 默认会缓冲，导致延迟增加或连接卡死 |
| `proxy_read_timeout 86400s` | SSE 连接可能长期保持，默认 60s 超时会断开连接 |
| `proxy_http_version 1.1` | 禁用 HTTP/1.0 以支持长连接和分块传输 |
| `proxy_set_header X-Real-IP` | 将真实客户端 IP 传递给后端，供日志和指标使用 |
| `client_max_body_size 10m` | 代码执行沙箱可能上传较大 Python 脚本 |

**Nginx 下 `remote_addr` 说明：**

配置了 `proxy_set_header X-Real-IP $remote_addr` 后，MCP Hub 会自动识别 `X-Real-IP` 或 `X-Forwarded-For` 请求头，并在 `mcp_http_requests_total` 的 `remote_addr` 标签中使用真实客户端 IP。优先级顺序：`X-Forwarded-For`（取第一个 IP）> `X-Real-IP` > 直连 IP。

---

## 内置服务

| 服务 | 路径 | 描述 |
|------|------|------|
| 工作日服务 | `/mcp/workday` | 中国节假日和工作日计算 |
| Arch Linux 服务 | `/mcp/archlinux` | 官方仓库和 AUR 软件包搜索 |
| 代码执行沙箱 | `/mcp/code-exec` | Docker 隔离的 Python/Node.js/Shell 执行环境 |

## 🚀 快速开始

### 本地运行

```bash
# 1. 编译所有二进制
make build

# 2.（可选）构建代码执行沙箱镜像（需要 Docker）
make build-sandboxes

# 3. 复制配置
cp config.example.yaml config.yaml
# 编辑 config.yaml 配置 API Key 和服务

# 4. 启动
./bin/mcp-server -config config.yaml
```

### Docker Compose

```bash
# 1. 构建代码执行沙箱镜像（宿主机上执行一次即可）
make build-sandboxes

# 2. 启动 MCP Hub（自动挂载 Docker socket 供代码沙箱使用）
cp config.example.yaml config.yaml
# 编辑 config.yaml 配置 API Key 和服务
docker compose up -d
```

> ⚠️ **代码执行沙箱需要 Docker**：docker-compose 已挂载 `/var/run/docker.sock`，
> MCP Hub 容器内的 code-exec 服务会创建独立容器执行代码。

### Docker 单次运行

```bash
docker build -t mcp-hub .
docker run -d --name mcp-hub -p 8080:8080 \
  -v ./config.yaml:/app/config.yaml:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  mcp-hub:latest
```

### 从 GHCR 拉取

```bash
docker pull ghcr.io/liyp/mcp-hub:latest
```

## 📖 MCP 调用指南

### 请求格式 (JSON-RPC 2.0)

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "方法名",
    "params": { ... }
}
```

### 基本调用流程

```bash
# 1. 列出工具
curl -X POST http://localhost:8080/mcp/workday \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# 2. 调用工具
curl -X POST http://localhost:8080/mcp/workday \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "get_month_workdays",
      "arguments": {"year": 2026, "month": 2}
    }
  }'
```

### 响应格式

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{"type": "text", "text": "2026年2月工作日: 16天"}]
  }
}
```

## 🔧 工作日服务工具

| 工具 | 描述 |
|------|------|
| `get_date_info` | 查询日期类型 |
| `get_date_detail` | 获取日期详细信息（类型、调休、薪资等） |
| `get_month_workdays` | 计算月份工作日数 |
| `calculate_man_days` | 计算人日 |
| `get_holiday_list` | 获取节假日列表 |
| `get_next_workday` | 查找下一个工作日 |
| `get_next_holiday` | 查找下一个节假日 |
| `batch_check_dates` | 批量查询日期 |
| `get_period_workdays` | 计算时间段工作日 |

## 🐧 Arch Linux 服务工具

| 工具 | 参数 | 描述 |
|------|------|------|
| `search_package` | `keyword`, `repo?`, `source?` | 搜索官方仓库和 AUR 软件包 |
| `get_package_info` | `name`, `source?` | 获取包详情（依赖、许可证、维护者等） |
| `get_maintainer_packages` | `maintainer` | 获取 AUR 维护者的所有包 |

**参数说明:**
- `source`: `official`（官方仓库）、`aur`、`all`（默认）
- `repo`: 指定仓库，如 `core`, `extra`, `multilib`

## 💻 代码执行沙箱工具

基于 **Docker 容器隔离** 的代码执行环境，支持三种语言：

| 工具 | 参数 | 描述 |
|------|------|------|
| `execute_python` | `code`, `timeout?` | 在 Docker 容器中执行 Python 3（预装 numpy/pandas/matplotlib/seaborn） |
| `execute_nodejs` | `code`, `timeout?` | 在 Docker 容器中执行 Node.js 22（支持 TypeScript） |
| `execute_shell` | `code`, `timeout?` | 在 Docker 容器中执行 Shell（Alpine + bash + curl/jq/git） |

**安全隔离（每次执行独立容器）:**
- 丢弃所有 Linux 能力（`--cap-drop=ALL`）+ `no-new-privileges`
- 非 root 用户执行（`sandbox`）
- 网络隔离：默认无网络，可通过 config.yaml 或环境变量开启
- 资源限制：内存 / CPU / 进程数 / 文件描述符
- 容器退出后自动清理（`--rm`）

**沙箱容器生命周期:**
- 沙箱容器**不是常驻进程**，每次调用工具时创建，执行完自动删除（`--rm`）
- `code-exec-svc` 进程由主服务启动时自动拉起（stdio 子进程），无需手动操作

**网络配置（重要）:**
- `config.yaml` 中 code-exec 服务的 `sandbox.network.enabled` 控制**沙箱容器内代码**是否联网
- `true` → 沙箱容器使用 bridge 网络（可访问外网）；`false`/未配置 → 无网络（默认）
- 该配置由主服务通过 `MCP_SANDBOX_NETWORK` 环境变量自动传递给 code-exec-svc，无需手动设置
- 如需更细粒度控制，可用 `CODE_EXEC_<LANG>_NETWORK` 环境变量单独覆盖（优先级更高）

**图表输出:**
- `matplotlib` / `seaborn` 图表自动捕获（调用 `plt.show()` 即可，返回 base64 PNG）
- 工作目录 `/sandbox` 中生成的图片文件也会自动提取
- `plotly` 未预装（可自定义镜像），使用时会输出提示到 stderr

**前置条件:**
1. Docker 已安装并运行（部署在容器内需挂载 Docker socket）
2. 沙箱镜像**自动管理**：镜像不存在时首次执行会自动构建（无需手动操作）；
   如需预构建可运行 `make build-sandboxes`
3. 镜像大小：python 371MB / nodejs 316MB / shell 71MB

**镜像自动构建逻辑（按需触发）:**
- 镜像已存在 → 直接使用
- 不存在 → 尝试 `docker pull`（远程镜像）
- 拉取失败 → 自动从本地 `services/code-exec/sandboxes/<lang>/Dockerfile` 构建
- 全部失败 → 返回清晰错误提示

**配置（环境变量，可选）:**
```bash
CODE_EXEC_PYTHON_IMAGE=mcp-hub/python-sandbox:latest  # 自定义镜像
CODE_EXEC_PYTHON_MEMORY=512                           # 内存限制 MB
CODE_EXEC_PYTHON_NETWORK=none                         # none 或 bridge
CODE_EXEC_PYTHON_TIMEOUT=60                           # 默认超时秒数
CODE_EXEC_SANDBOX_DIR=./services/code-exec/sandboxes  # 沙箱 Dockerfile 目录
CODE_EXEC_NODEJS_IMAGE=... / CODE_EXEC_SHELL_IMAGE=...  # 其他语言同理
```

## 🔐 认证

```bash
# 方式1: X-API-Key
-H "X-API-Key: your-key"

# 方式2: Bearer Token
-H "Authorization: Bearer your-key"
```

## ⚙️ 配置

所有配置统一在 `config.yaml` 中管理，也可通过命令行参数或环境变量覆盖。

**优先级**: 命令行参数 > 配置文件 > 环境变量 > 默认值

### 完整配置示例

```yaml
# config.yaml
addr: ":8080"

api_keys:
  - "your-secret-key"

no_log: false

services:
  - name: "工作日服务"
    command: "./bin/workday-svc"
    path: "/mcp/workday"
    sandbox:
      network:
        enabled: true
      env:
        inherit: false
        allow: ["PATH", "HOME"]

  - name: "代码执行沙箱"
    command: "./bin/code-exec-svc"
    path: "/mcp/code-exec"
    sandbox:
      network:
        enabled: false
      timeout: "60s"
      env:
        inherit: false
        allow: ["PATH"]

  - name: "文件系统"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    path: "/mcp/filesystem"
    sandbox:
      network:
        enabled: false
      fs:
        mode: "read-only"
      env:
        inherit: false
        allow: ["PATH", "HOME"]
```

### 配置字段说明

| 字段 | 命令行 | 环境变量 | 说明 | 默认值 |
|------|--------|---------|------|--------|
| `addr` | `-addr` | `ADDR` | 监听地址 | `:8080` |
| `api_keys` | `-api-keys` | `API_KEYS` | API Key 列表 | 空 |
| `no_log` | `-no-log` | `NO_LOG` | 禁用请求日志 | `false` |
| `services` | — | — | 外部 MCP 服务列表 | 空 |

### 代理服务配置

每个 `services` 条目定义一个通过 stdio 接入的外部 MCP Server：

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | ✅ | 服务名称 |
| `description` | ❌ | 服务描述 |
| `command` | ✅ | 启动命令（如 `npx`、`./bin/workday-svc`） |
| `args` | ❌ | 命令参数列表 |
| `path` | ✅ | HTTP 路由路径，建议 `/mcp/<name>` 格式 |
| `env` | ❌ | 传递给子进程的环境变量 |
| `sandbox` | ❌ | 沙箱权限配置（见下方） |

### 沙箱配置（Sandbox）

通过 `sandbox` 字段控制每个代理服务的权限，不配置则默认无限制。

```yaml
services:
  - name: "文件系统"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    path: "/mcp/filesystem"
    sandbox:
      network:
        enabled: false              # 禁止联网
      fs:
        mode: "read-only"           # 只读文件系统
      env:
        inherit: false              # 不继承宿主环境变量
        allow: ["PATH", "HOME"]     # 只传递白名单内的变量
      timeout: "30s"                # 每次请求超时
```

#### 沙箱字段说明

| 字段 | 子字段 | 说明 | 默认值 |
|------|--------|------|--------|
| `network` | `enabled` | 是否允许联网 | `false`（禁止联网） |
| | `egress` | 出站白名单地址列表 | 空 |
| `fs` | `mode` | 文件系统模式：`read-only` / `whitelist` / `full` | `full` |
| | `read_write` | 读写白名单路径（mode=whitelist 时有效） | 空 |
| | `read_only` | 只读白名单路径（mode=whitelist 时有效） | 空 |
| `env` | `inherit` | 是否继承宿主环境变量 | `false` |
| | `allow` | 环境变量白名单 | 空 |
| `timeout` | — | 每次请求超时时间，如 `"30s"` | 无限制 |
| `memory` | — | 内存上限，如 `"256MB"`（需系统支持） | 无限制 |
| `private_tmp` | — | 是否使用独立临时目录 | `false` |

> **安全建议**: 对于来源不明的第三方服务，建议设置 `sandbox.env.inherit: false` 并只允许 `PATH` 和 `HOME`，防止 SSH 密钥、API Token 等敏感环境变量泄露。

---

## 构建系统

使用 Makefile 一键构建所有二进制：

```bash
make build              # 构建全部二进制
make build/server       # 仅构建主服务
make build/workday-svc  # 仅构建工作日服务
make build/code-exec-svc
make build-sandboxes    # 构建代码执行沙箱 Docker 镜像（python/nodejs/shell）
make build-sandbox/python  # 构建单个沙箱镜像
make build-all          # 构建全部（二进制 + 沙箱镜像）
make clean              # 清理构建产物
make list               # 列出所有可构建的二进制
```

Makefile 自动发现 `services/*/cmd/*/main.go` 和 `services/code-exec/sandboxes/*/Dockerfile`，新增服务无需修改构建配置。

## 🔄 CI/CD

项目使用 GitHub Actions 自动构建和发布：

- **每次提交** → 自动构建 + 测试 + lint
- **打 `v*` tag** → 构建多架构 Docker 镜像 (amd64 + arm64) 并推送到 GHCR

```bash
# 打 tag 发布
git tag v1.0.0
git push origin v1.0.0

# 拉取镜像
docker pull ghcr.io/liyp/mcp-hub:latest
```

## 🐳 Docker 多目标构建

Dockerfile 支持多目标构建，可以选择构建特定服务镜像：

```bash
# 构建全部（默认）
docker build -t mcp-hub .

# 构建特定目标
docker build --target server -t mcp-hub-server .
docker build --target code-exec-svc-builder -t code-exec-svc .
```

---

## 📁 项目结构

```
mcp-hub/
├── cmd/server/main.go        # 服务入口（不硬编码任何服务）
├── internal/
│   ├── config/config.go      # 配置文件解析
│   ├── config/sandbox.go     # 沙箱配置类型
│   ├── mcp/
│   │   ├── service.go        # MCPService 接口定义
│   │   └── registry.go       # 服务注册器 + HTTP 路由 + Web 页面
│   ├── middleware/
	│   │   ├── middleware.go     # 认证/日志中间件
	│   │   └── metrics.go       # Prometheus 指标采集中间件
	│   └── proxy/proxy.go        # stdio 代理服务（含沙箱隔离）
├── pkg/                      # 公共库
│   ├── holiday/              # 中国节假日计算
│   └── archlinux/            # Arch Linux 包搜索客户端
├── services/                 # 独立 MCP 服务二进制
│   ├── workday/              # 工作日服务
│   │   ├── cmd/workday-svc/  # 独立入口
│   │   └── service.go
│   ├── archlinux/            # Arch Linux 服务
│   │   ├── cmd/archlinux-svc/
│   │   └── service.go
│   └── code-exec/            # 代码执行沙箱（Docker 容器隔离）
│       ├── cmd/code-exec-svc/
│       ├── config.go         # 沙箱配置（环境变量可覆盖）
│       ├── service.go        # 3 个 MCP 工具注册
│       ├── executor.go       # Docker 容器执行器
│       ├── images.go         # 图表文件提取
│       ├── restrictions.go   # 图表捕获 preamble
│       └── sandboxes/        # 沙箱镜像定义（Dockerfile）
│           ├── python/       # Python 3.13 + numpy/pandas/matplotlib/seaborn
│           ├── nodejs/       # Node.js 22 + TypeScript
│           └── shell/        # Alpine + bash + curl/jq/git
├── web/                      # Web 前端模板
│   ├── templates/
│   │   ├── index.html
│   │   ├── login.html
│   │   └── playground.html
│   └── embed.go
├── Makefile                  # 一键构建系统
├── Dockerfile                # 多目标多阶段构建
├── docker-compose.yml
├── .github/workflows/ci.yml  # CI/CD 流水线
├── grafana/
│   └── dashboard.json        # Grafana 观测面板模板
├── config.example.yaml
└── go.mod
```

## 🧩 开发指南

### 添加新服务

只需 **2 步**：

#### 1. 创建服务代码

在 `services/` 下创建新目录和 `service.go`：

```go
// services/weather/service.go
package weather

import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func NewService() *server.MCPServer {
    s := server.NewMCPServer(
        "weather-service", "1.0.0",
        server.WithToolCapabilities(false),
        server.WithRecovery(),
    )
    s.AddTool(mcp.NewTool("get_weather",
        mcp.WithDescription("查询城市天气"),
        mcp.WithString("city", mcp.Required(), mcp.Description("城市名称")),
    ), handleGetWeather)
    return s
}
```

#### 2. 创建独立入口

```go
// services/weather/cmd/weather-svc/main.go
package main

import (
    "mcp-hub/services/weather"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    server.ServeStdio(weather.NewService())
}
```

#### 3. 添加到配置

```yaml
# config.yaml
services:
  - name: "天气服务"
    command: "./bin/weather-svc"
    path: "/mcp/weather"
```

**无需修改任何 Go 代码**，Makefile 会自动发现新服务。

### 开发规范

1. **服务路径**：统一使用 `/mcp/<service-name>` 格式
2. **工具命名**：使用 `snake_case`，如 `get_weather`、`calculate_distance`
3. **错误处理**：使用 `mcp.NewToolResultError()` 返回错误
4. **日志**：使用标准库 `log` 包

## 📄 License

MIT