// Package mcp 提供 MCP 服务的核心接口和注册机制。
package mcp

import "github.com/mark3labs/mcp-go/server"

// MCPService 定义 MCP 服务接口。
// 每个 MCP 服务需要实现此接口以注册到服务注册器。
type MCPService interface {
	// Name 返回服务名称（用于日志和标识）
	Name() string

	// Path 返回服务的 HTTP 路径（如 "/mcp/workday"）
	Path() string

	// Description 返回服务描述
	Description() string

	// MCPServer 返回配置好的 MCP 服务器实例
	MCPServer() *server.MCPServer
}
