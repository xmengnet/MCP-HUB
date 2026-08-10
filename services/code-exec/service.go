// Package codeexec 提供安全的代码执行沙箱服务。
//
// 本文件定义 MCP Tool 并注册到 MCPServer。
// 提供 3 个工具：execute_python、execute_nodejs、execute_shell。
// 每次调用启动独立 Docker 容器，提供 OS 级隔离。
package codeexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewService 创建并返回配置好的代码执行 MCP Server
//
// executor 为 nil 时使用默认配置创建执行器
func NewService() *server.MCPServer {
	cfg := LoadConfig()
	return NewServiceWithConfig(cfg)
}

// NewServiceWithConfig 使用指定配置创建 MCP Server
func NewServiceWithConfig(cfg *Config) *server.MCPServer {
	executor := NewExecutor(cfg)

	s := server.NewMCPServer(
		"code-exec-service",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	registerPythonTool(s, executor, cfg)
	registerNodejsTool(s, executor, cfg)
	registerShellTool(s, executor, cfg)

	return s
}

// registerPythonTool 注册 Python 代码执行工具
func registerPythonTool(s *server.MCPServer, executor *Executor, cfg *Config) {
	sb := cfg.Sandboxes["python"]
	packages := "numpy, pandas, matplotlib, seaborn, requests, Pillow"

	s.AddTool(
		mcp.NewTool("execute_python",
			mcp.WithDescription(fmt.Sprintf(`在 Docker 容器隔离的沙箱中执行 Python 3 代码。

环境信息:
- 预装包: %s（如需其他包，请用 matplotlib 等已装库替代）
- 内存限制: %dMB，CPU: %.1f核，超时: %d秒
- 网络: %s
- 工作目录 /sandbox 可读写，用户代码生成的文件会自动提取
- matplotlib/seaborn 图表会自动捕获并返回（调用 plt.show() 即可）

安全说明: 代码在独立容器中执行，无 root 权限，无网络（除非配置开启）。`,
				packages, sb.MemoryMB, sb.CPUCores, sb.TimeoutSec,
				networkDescription(sb.Network))),
			mcp.WithString("code",
				mcp.Required(),
				mcp.Description("要执行的 Python 3 代码"),
			),
			mcp.WithNumber("timeout",
				mcp.Description(fmt.Sprintf("超时秒数（默认 %d，最大 %d）", sb.TimeoutSec, maxTimeout)),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleExecute(ctx, req, executor, "python")
		},
	)
}

// registerNodejsTool 注册 Node.js 代码执行工具
func registerNodejsTool(s *server.MCPServer, executor *Executor, cfg *Config) {
	sb := cfg.Sandboxes["nodejs"]

	s.AddTool(
		mcp.NewTool("execute_nodejs",
			mcp.WithDescription(fmt.Sprintf(`在 Docker 容器隔离的沙箱中执行 Node.js 22 代码（JavaScript）。

环境信息:
- Node.js 22，支持 CommonJS 语法（require / module.exports）
- 预装: tsx（TypeScript 运行时）、typescript
- 内存限制: %dMB，CPU: %.1f核，超时: %d秒
- 网络: %s
- 工作目录 /sandbox 可读写

安全说明: 代码在独立容器中执行，无 root 权限，无网络（除非配置开启）。`,
				sb.MemoryMB, sb.CPUCores, sb.TimeoutSec,
				networkDescription(sb.Network))),
			mcp.WithString("code",
				mcp.Required(),
				mcp.Description("要执行的 JavaScript 代码（CommonJS 语法）"),
			),
			mcp.WithNumber("timeout",
				mcp.Description(fmt.Sprintf("超时秒数（默认 %d，最大 %d）", sb.TimeoutSec, maxTimeout)),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleExecute(ctx, req, executor, "nodejs")
		},
	)
}

// registerShellTool 注册 Shell 代码执行工具
func registerShellTool(s *server.MCPServer, executor *Executor, cfg *Config) {
	sb := cfg.Sandboxes["shell"]

	s.AddTool(
		mcp.NewTool("execute_shell",
			mcp.WithDescription(fmt.Sprintf(`在 Docker 容器隔离的沙箱中执行 Shell 命令。

环境信息:
- Alpine Linux + bash，预装 curl, wget, jq, git, python3, sqlite
- 内存限制: %dMB，CPU: %.1f核，超时: %d秒
- 网络: %s（默认无网络，可通过配置开启）
- 工作目录 /sandbox 可读写

安全说明: 代码在独立容器中执行，无 root 权限。适合系统操作、文本处理、脚本编排。`,
				sb.MemoryMB, sb.CPUCores, sb.TimeoutSec,
				networkDescription(sb.Network))),
			mcp.WithString("code",
				mcp.Required(),
				mcp.Description("要执行的 Shell 脚本（bash 语法）"),
			),
			mcp.WithNumber("timeout",
				mcp.Description(fmt.Sprintf("超时秒数（默认 %d，最大 %d）", sb.TimeoutSec, maxTimeout)),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleExecute(ctx, req, executor, "shell")
		},
	)
}

// handleExecute 通用的执行处理器
func handleExecute(ctx context.Context, req mcp.CallToolRequest, executor *Executor, lang string) (*mcp.CallToolResult, error) {
	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	timeout := 0
	if timeoutVal, err := req.RequireFloat("timeout"); err == nil {
		timeout = int(timeoutVal)
	}

	result, err := executor.Execute(ctx, lang, code, timeout)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v", err)), nil
	}

	// 构建响应内容
	var contents []mcp.Content

	// 添加 stdout
	if result.Stdout != "" {
		contents = append(contents, mcp.NewTextContent(result.Stdout))
	}

	// 添加图片（仅 Python 沙箱会生成图片）
	for _, img := range result.Images {
		contents = append(contents, mcp.NewImageContent(img.Data, img.MIME))
	}

	// 添加执行信息
	var info strings.Builder
	info.WriteString(fmt.Sprintf("退出码: %d", result.ExitCode))
	if result.TimedOut {
		info.WriteString(" ⚠️ 超时")
	}
	info.WriteString(fmt.Sprintf(" ⏱ 耗时: %s", result.Duration.Round(time.Millisecond)))
	if result.Stderr != "" {
		info.WriteString(fmt.Sprintf("\nstderr:\n%s", result.Stderr))
	}
	contents = append(contents, mcp.NewTextContent(info.String()))

	return &mcp.CallToolResult{
		Content: contents,
	}, nil
}

// networkDescription 返回网络模式的可读描述
func networkDescription(network string) string {
	if network == "bridge" {
		return "桥接（可访问外网）"
	}
	return "无（无法访问网络）"
}
