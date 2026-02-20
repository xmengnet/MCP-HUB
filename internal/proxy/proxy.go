// Package proxy 提供 stdio MCP 代理服务，用于接入外部 MCP Server。
package proxy

import (
	"context"
	"fmt"
	"log"
	"time"

	"mcp-hub/internal/config"
	mcpinternal "mcp-hub/internal/mcp"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ProxyService 代理外部 stdio MCP Server 的服务
type ProxyService struct {
	name        string
	description string
	path        string
	mcpServer   *server.MCPServer
	mcpClient   *client.Client
}

// NewProxyService 根据配置创建代理服务
// 1. 启动子进程并建立 stdio 连接
// 2. 初始化 MCP 握手
// 3. 获取远程工具列表
// 4. 将远程工具注册到本地 MCPServer（handler 透传调用）
func NewProxyService(cfg config.ServiceConfig) (*ProxyService, error) {
	// 构建环境变量列表
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 启动子进程
	mcpClient, err := client.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("启动子进程失败 (%s): %w", cfg.Command, err)
	}

	// 初始化握手
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-hub-proxy",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("MCP 握手失败 (%s): %w", cfg.Name, err)
	}

	// 创建本地 MCPServer
	mcpServer := server.NewMCPServer(
		cfg.Name,
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	svc := &ProxyService{
		name:        cfg.Name,
		description: cfg.Description,
		path:        cfg.Path,
		mcpServer:   mcpServer,
		mcpClient:   mcpClient,
	}

	// 获取远程工具并注册到本地
	if err := svc.syncTools(); err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("同步工具失败 (%s): %w", cfg.Name, err)
	}

	return svc, nil
}

// syncTools 从远程服务获取工具列表并注册到本地 MCPServer
func (s *ProxyService) syncTools() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := s.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("获取工具列表失败: %w", err)
	}

	for _, tool := range result.Tools {
		// 复制到局部变量，避免闭包问题
		t := tool
		s.mcpServer.AddTool(t, s.makeToolHandler(t.Name))
		log.Printf("  ↳ 注册工具: %s", t.Name)
	}

	log.Printf("代理服务 %q: 共注册 %d 个工具", s.name, len(result.Tools))
	return nil
}

// makeToolHandler 创建一个将请求透传到远程服务的 handler
func (s *ProxyService) makeToolHandler(toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := s.mcpClient.CallTool(ctx, req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("代理调用失败: %v", err)), nil
		}
		return result, nil
	}
}

// --- MCPService 接口实现 ---

func (s *ProxyService) Name() string                 { return s.name }
func (s *ProxyService) Path() string                 { return s.path }
func (s *ProxyService) Description() string          { return s.description }
func (s *ProxyService) MCPServer() *server.MCPServer { return s.mcpServer }

// Close 关闭与远程服务的连接
func (s *ProxyService) Close() error {
	if s.mcpClient != nil {
		return s.mcpClient.Close()
	}
	return nil
}

// LoadAll 从配置加载所有代理服务并注册到 Registry
func LoadAll(cfg *config.Config, registry *mcpinternal.Registry) ([]*ProxyService, error) {
	var proxies []*ProxyService

	for _, svcCfg := range cfg.Services {
		log.Printf("正在启动代理服务: %s (%s %v)", svcCfg.Name, svcCfg.Command, svcCfg.Args)

		proxySvc, err := NewProxyService(svcCfg)
		if err != nil {
			log.Printf("⚠️  代理服务启动失败: %s: %v", svcCfg.Name, err)
			continue // 跳过失败的服务，不影响其他服务
		}

		if err := registry.Register(proxySvc); err != nil {
			log.Printf("⚠️  代理服务注册失败: %s: %v", svcCfg.Name, err)
			proxySvc.Close()
			continue
		}

		proxies = append(proxies, proxySvc)
	}

	return proxies, nil
}
