// Package proxy 提供 stdio MCP 代理服务，用于接入外部 MCP Server。
package proxy

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"mcp-hub/internal/config"
	mcpinternal "mcp-hub/internal/mcp"
	"mcp-hub/internal/middleware"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
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
	sandbox     *config.SandboxConfig
	metrics     *middleware.Metrics
}

// NewProxyService 根据配置创建代理服务
// 1. 应用沙箱配置（环境变量过滤等）
// 2. 启动子进程并建立 stdio 连接
// 3. 初始化 MCP 握手
// 4. 获取远程工具列表
// 5. 将远程工具注册到本地 MCPServer（handler 透传调用）
func NewProxyService(cfg config.ServiceConfig, metrics *middleware.Metrics) (*ProxyService, error) {
	// 构建并过滤环境变量列表
	env := buildEnv(cfg.Env, cfg.Sandbox)

	// 使用自定义 CommandFunc 来精确控制子进程环境变量
	// 原因: mcp-go 默认的 cmd.Env = append(os.Environ(), env...) 会把宿主环境全部传进去
	// 我们的自定义函数会用 buildEnv 的结果完全替换，实现真正的环境隔离
	cmdFunc := func(ctx context.Context, command string, cmdEnv []string, args []string) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Env = cmdEnv
		return cmd, nil
	}

	// 启动子进程（使用自定义 CommandFunc）
	mcpClient, err := client.NewStdioMCPClientWithOptions(
		cfg.Command, env, cfg.Args,
		transport.WithCommandFunc(cmdFunc),
	)
	if err != nil {
		return nil, fmt.Errorf("启动子进程失败 (%s): %w", cfg.Command, err)
	}

	// 打印沙箱配置信息
	if cfg.Sandbox != nil {
		log.Printf("  🔒 沙箱: 网络=%v 文件=%s 环境变量=%d个 超时=%s",
			cfg.Sandbox.Network != nil && cfg.Sandbox.Network.Enabled,
			sandboxFSMode(cfg.Sandbox.FS),
			len(env),
			cfg.Sandbox.Timeout)
	}

// 初始化握手（npx 首次下载包可能较慢，给 120 秒超时）
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
		sandbox:     cfg.Sandbox,
		metrics:     metrics,
	}

	// 获取远程工具并注册到本地
	if err := svc.syncTools(); err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("同步工具失败 (%s): %w", cfg.Name, err)
	}

	// 记录服务在线状态
	if metrics != nil {
		metrics.SetServiceUp(cfg.Name, true)
	}

	return svc, nil
}

// buildEnv 构建环境变量列表，并根据沙箱配置过滤
// 沙箱规则:
//   - sandbox.Env.Inherit=false（默认）: 不继承宿主环境，只传递 Env 白名单 + 配置中显式设置的 env
//   - sandbox.Env.Inherit=true: 继承宿主环境，额外保留 Env.Allow 白名单
func buildEnv(cfgEnv map[string]string, sandbox *config.SandboxConfig) []string {
	// 先收集配置中显式指定的环境变量
	var explicit []string
	for k, v := range cfgEnv {
		explicit = append(explicit, fmt.Sprintf("%s=%s", k, v))
	}

	// 没有沙箱配置，直接返回显式变量（继承宿主全部环境变量）
	if sandbox == nil || sandbox.Env == nil {
		return explicit
	}

	envConfig := sandbox.Env

	// 如果不继承宿主环境
	if !envConfig.Inherit {
		// 只传递白名单 + 显式配置的变量
		var filtered []string
		filtered = append(filtered, explicit...)

		// 从宿主环境读取白名单变量
		for _, key := range envConfig.Allow {
			if val := os.Getenv(key); val != "" {
				// 避免被显式变量覆盖
				if _, exists := cfgEnv[key]; !exists {
					filtered = append(filtered, fmt.Sprintf("%s=%s", key, val))
				}
			}
		}
		return filtered
	}

	// 继承宿主环境，但只保留白名单
	hostEnv := os.Environ()
	if len(envConfig.Allow) == 0 {
		return explicit
	}

	allowSet := make(map[string]bool, len(envConfig.Allow))
	for _, k := range envConfig.Allow {
		allowSet[k] = true
	}

	var filtered []string
	filtered = append(filtered, explicit...)
	for _, entry := range hostEnv {
		key := strings.SplitN(entry, "=", 2)[0]
		if allowSet[key] {
			// 避免被显式变量覆盖
			if _, exists := cfgEnv[key]; !exists {
				filtered = append(filtered, entry)
			}
		}
	}
	return filtered
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
// 如果配置了超时，会在 context 上附加超时控制
func (s *ProxyService) makeToolHandler(toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// 应用超时控制
		execCtx := ctx
		if s.sandbox != nil && s.sandbox.Timeout != "" {
			if timeout, err := time.ParseDuration(s.sandbox.Timeout); err == nil && timeout > 0 {
				var cancel context.CancelFunc
				execCtx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		result, err := s.mcpClient.CallTool(execCtx, req)

		// 记录工具调用指标
		if s.metrics != nil {
			status := "ok"
			if err != nil {
				status = "error"
			} else if result != nil && result.IsError {
				status = "error"
			}
			s.metrics.RecordToolCall(s.name, toolName, status, time.Since(start))
		}

		if err != nil {
			if execCtx.Err() == context.DeadlineExceeded {
				return mcp.NewToolResultError(fmt.Sprintf("工具 %q 调用超时 (%s)", toolName, s.sandbox.Timeout)), nil
			}
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
	if s.metrics != nil {
		s.metrics.SetServiceUp(s.name, false)
	}
	if s.mcpClient != nil {
		return s.mcpClient.Close()
	}
	return nil
}

// sandboxFSMode 返回文件系统沙箱模式的可读字符串
func sandboxFSMode(fs *config.FSConfig) string {
	if fs == nil || fs.Mode == "" {
		return "full"
	}
	return fs.Mode
}

// LoadAll 从配置加载所有代理服务并注册到 Registry（同步）
func LoadAll(cfg *config.Config, registry *mcpinternal.Registry, metrics *middleware.Metrics) ([]*ProxyService, error) {
	var proxies []*ProxyService

	for _, svcCfg := range cfg.Services {
		log.Printf("正在启动代理服务: %s (%s %v)", svcCfg.Name, svcCfg.Command, svcCfg.Args)

		proxySvc, err := NewProxyService(svcCfg, metrics)
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

// LoadAllAsync 异步加载所有代理服务，每加载完成一个回调一次
// 服务器可先启动，服务在后台逐步注册
func LoadAllAsync(cfg *config.Config, registry *mcpinternal.Registry, metrics *middleware.Metrics, callback func(*ProxyService, error)) {
	for _, svcCfg := range cfg.Services {
		cfg := svcCfg // 捕获循环变量
		go func() {
			log.Printf("正在启动代理服务: %s (%s %v)", cfg.Name, cfg.Command, cfg.Args)

			proxySvc, err := NewProxyService(cfg, metrics)
			if err != nil {
				callback(nil, fmt.Errorf("%s: %w", cfg.Name, err))
				return
			}

			if err := registry.Register(proxySvc); err != nil {
				proxySvc.Close()
				callback(nil, fmt.Errorf("%s: %w", cfg.Name, err))
				return
			}

			callback(proxySvc, nil)
		}()
	}
}