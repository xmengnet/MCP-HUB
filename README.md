# MCP Hub

🚀 可扩展的 MCP (Model Context Protocol) 服务框架，轻松构建和托管多个 MCP 服务。

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ 特性

- 🔌 **即插即用** - 简单接口，快速添加新服务
- 🛤️ **路径路由** - 每个服务独立路径，互不干扰
- 🔐 **API Key 认证** - 开箱即用的安全认证
- 📝 **请求日志** - 内置请求日志中间件
- 🐳 **Docker 支持** - 一键容器化部署
- 🧩 **Init 自动注册** - 空导入即可注册，无需修改 main 函数

## 内置服务

| 服务 | 路径 | 描述 |
|------|------|------|
| 工作日服务 | `/mcp/workday` | 中国节假日和工作日计算 |
| Arch Linux 服务 | `/mcp/archlinux` | 官方仓库和 AUR 软件包搜索 |

## 🚀 快速开始

### Docker Compose

```bash
echo "API_KEYS=your-secret-key" > .env
docker compose up -d
```

### 本地运行

```bash
go build -o mcp-hub ./cmd/server
./mcp-hub -addr :8080 -api-keys "your-key"
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

## 🔐 认证

```bash
# 方式1: X-API-Key
-H "X-API-Key: your-key"

# 方式2: Bearer Token
-H "Authorization: Bearer your-key"
```

## ⚙️ 配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-addr` | 监听地址 | `:8080` |
| `-api-keys` | API Key（逗号分隔） | 空 |
| `-no-log` | 禁用日志 | `false` |

---

## �️ 开发指南

### 项目结构

```
mcp-hub/
├── cmd/server/main.go      # 服务入口（空导入注册服务）
├── internal/
│   ├── mcp/
│   │   ├── service.go      # MCPService 接口定义
│   │   └── registry.go     # 服务注册器（含全局注册机制）
│   └── middleware/
│       └── middleware.go   # 认证/日志中间件
├── pkg/                    # 公共库（可被服务复用）
│   └── holiday/            # 节假日计算库
├── services/               # MCP 服务实现
│   └── workday/            # 工作日服务示例
│       └── service.go
├── Dockerfile
└── docker-compose.yml
```

### 添加新服务

只需 **2 步**：

#### 1. 创建服务文件

在 `services/` 下创建新目录和 `service.go`：

```go
// services/weather/service.go
package weather

import (
    mcpinternal "mcp-hub/internal/mcp"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// init 自动注册到全局注册器
func init() {
    mcpinternal.Register(NewService())
}

type Service struct {
    mcpServer *server.MCPServer
}

func NewService() mcpinternal.MCPService {
    svc := &Service{}
    svc.mcpServer = server.NewMCPServer(
        "weather-service",
        "1.0.0",
        server.WithToolCapabilities(false),
        server.WithRecovery(),
    )
    svc.registerTools()
    return svc
}

func (s *Service) Name() string        { return "天气服务" }
func (s *Service) Path() string        { return "/mcp/weather" }
func (s *Service) Description() string { return "天气查询服务" }
func (s *Service) MCPServer() *server.MCPServer { return s.mcpServer }

func (s *Service) registerTools() {
    s.mcpServer.AddTool(
        mcp.NewTool("get_weather",
            mcp.WithDescription("查询城市天气"),
            mcp.WithString("city", mcp.Required(), mcp.Description("城市名称")),
        ),
        s.handleGetWeather,
    )
}

// ... 实现工具处理函数
```

#### 2. 添加空导入

在 `cmd/server/main.go` 添加一行导入：

```go
import (
    // ...
    _ "mcp-hub/services/workday"
    _ "mcp-hub/services/weather"  // 新增这一行
)
```

#### 3. 编译运行

```bash
go build -o mcp-hub ./cmd/server
./mcp-hub
```

### MCPService 接口

```go
type MCPService interface {
    Name() string                    // 服务名称（日志显示）
    Path() string                    // HTTP 路径（如 /mcp/weather）
    Description() string             // 服务描述
    MCPServer() *server.MCPServer    // MCP 服务器实例
}
```

### 注册机制说明

项目使用 **Init 自动注册模式**：

1. 每个服务包的 `init()` 函数调用 `mcp.Register()` 注册服务
2. 服务暂存在 `pendingServices` 切片中
3. `main()` 创建 Registry 后调用 `SetDefaultRegistry()`
4. 所有待处理服务自动注册到 Registry

这种模式的优点：
- ✅ 添加服务只需空导入一行
- ✅ 无需修改 main 函数逻辑
- ✅ 跨平台兼容
- ✅ 编译时类型检查

### 开发规范

1. **服务路径**：统一使用 `/mcp/<service-name>` 格式
2. **工具命名**：使用 `snake_case`，如 `get_weather`、`calculate_distance`
3. **错误处理**：使用 `mcp.NewToolResultError()` 返回错误
4. **日志**：使用标准库 `log` 包

## 📄 License

MIT
