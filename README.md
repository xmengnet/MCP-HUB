# MCP Hub

🚀 可扩展的 MCP (Model Context Protocol) 服务框架，轻松构建和托管多个 MCP 服务。

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ 特性

- 🔌 **即插即用** - 服务独立为二进制，通过 stdio 代理接入，零代码耦合
- 🔒 **沙箱安全隔离** - 每个服务可独立配置环境变量白名单、网络控制、文件系统权限
- 💻 **代码执行沙箱** - 内置安全的 Python 代码执行环境，支持 matplotlib 图表输出
- 🌐 **代理模式** - 通过配置文件接入外部 MCP Server，零编码
- 🛤️ **路径路由** - 每个服务独立路径，互不干扰
- 🔐 **API Key 认证** - 开箱即用的安全认证
- 📝 **请求日志** - 内置请求日志中间件
- 🐳 **Docker 支持** - 一键容器化部署，多架构镜像 (amd64 + arm64)
- 🔄 **CI/CD 自动构建** - 每次提交自动构建测试，打 tag 自动发布到 GHCR

## 内置服务

| 服务 | 路径 | 描述 |
|------|------|------|
| 工作日服务 | `/mcp/workday` | 中国节假日和工作日计算 |
| Arch Linux 服务 | `/mcp/archlinux` | 官方仓库和 AUR 软件包搜索 |
| 代码执行沙箱 | `/mcp/code-exec` | 安全的 Python 代码执行环境 |

## 🚀 快速开始

### 本地运行

```bash
# 1. 编译所有二进制
make build

# 2. 复制配置
cp config.example.yaml config.yaml
# 编辑 config.yaml 配置 API Key 和服务

# 3. 启动
./bin/mcp-server -config config.yaml
```

### Docker Compose

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml 配置 API Key 和服务
docker compose up -d
```

### Docker 单次运行

```bash
docker build -t mcp-hub .
docker run -d --name mcp-hub -p 8080:8080 \
  -v ./config.yaml:/app/config.yaml:ro \
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

| 工具 | 参数 | 描述 |
|------|------|------|
| `execute_python` | `code`, `timeout?` | 在安全沙箱中执行 Python 代码 |

**安全限制:**
- 禁止 `subprocess`, `ctypes`, `shutil`, `socket` 等危险模块
- 禁止 `os.system`, `os.popen`, `os.fork` 等系统调用
- 文件写入限制在临时目录内
- 支持 `matplotlib` 图表输出（base64 PNG 嵌入响应）
- 可配置超时时间（默认 30 秒）

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
make build              # 构建全部
make build/server       # 仅构建主服务
make build/workday-svc  # 仅构建工作日服务
make build/code-exec-svc
make clean              # 清理构建产物
make list               # 列出所有可构建的二进制
```

Makefile 自动发现 `services/*/cmd/*/main.go`，新增服务无需修改构建配置。

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
│   │   └── middleware.go     # 认证/日志中间件
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
│   └── code-exec/            # 代码执行沙箱
│       ├── cmd/code-exec-svc/
│       ├── service.go
│       ├── executor.go
│       └── restrictions.go
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