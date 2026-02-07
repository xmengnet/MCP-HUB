# MCP Hub

🚀 可扩展的 MCP (Model Context Protocol) 服务框架，轻松构建和托管多个 MCP 服务。

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ 特性

-  **即插即用** - 简单接口，快速添加新服务
- ️ **路径路由** - 每个服务独立路径，互不干扰
-  **API Key 认证** - 开箱即用的安全认证
-  **请求日志** - 内置请求日志中间件
-  **Docker 支持** - 一键容器化部署

## 📦 内置服务

| 服务 | 路径 | 描述 |
|------|------|------|
| 工作日服务 | `/mcp/workday` | 中国节假日和工作日计算 |
| *更多服务开发中...* | | |

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
| `get_month_workdays` | 计算月份工作日数 |
| `calculate_man_days` | 计算人日 |
| `get_holiday_list` | 获取节假日列表 |
| `get_next_workday` | 查找下一个工作日 |
| `get_next_holiday` | 查找下一个节假日 |
| `batch_check_dates` | 批量查询日期 |
| `get_period_workdays` | 计算时间段工作日 |

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

## 📚 添加新服务

1. 创建 `services/xxx/service.go`，实现接口：

```go
type MCPService interface {
    Name() string
    Path() string
    Description() string
    MCPServer() *server.MCPServer
}
```

2. 在 `cmd/server/main.go` 注册：

```go
registry.Register(xxx.NewService())
```

3. 重新编译即可

## 🏗️ 项目结构

```
├── cmd/server/         # 服务入口
├── internal/
│   ├── mcp/            # 服务框架核心
│   └── middleware/     # 认证/日志中间件
├── pkg/                # 公共库
├── services/           # MCP 服务实现
│   └── workday/        # 工作日服务
├── Dockerfile
└── docker-compose.yml
```

## 📄 License

MIT
