// Package codeexec 提供安全的代码执行沙箱服务。
package codeexec

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewService 创建并返回配置好的代码执行 MCP Server
func NewService() *server.MCPServer {
	executor := NewExecutor()

	s := server.NewMCPServer(
		"code-exec-service",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(
		mcp.NewTool("execute_python",
			mcp.WithDescription("在安全沙箱中执行 Python 代码，返回文本输出和图表（支持 matplotlib）"),
			mcp.WithString("code",
				mcp.Required(),
				mcp.Description("要执行的 Python 代码"),
			),
			mcp.WithNumber("timeout",
				mcp.Description("超时秒数（默认 30，最大 120）"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleExecute(ctx, req, executor)
		},
	)

	return s
}

func handleExecute(ctx context.Context, req mcp.CallToolRequest, executor *Executor) (*mcp.CallToolResult, error) {
	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	timeout := 0
	if timeoutVal, err := req.RequireFloat("timeout"); err == nil {
		timeout = int(timeoutVal)
		if timeout > 120 {
			timeout = 120
		}
	}

	result, err := executor.Execute(ctx, code, timeout)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v", err)), nil
	}

	// 构建响应内容
	var contents []mcp.Content

	// 添加文本输出
	if result.Stdout != "" {
		contents = append(contents, mcp.NewTextContent(result.Stdout))
	}

	// 添加图片
	for _, img := range result.Images {
		contents = append(contents, mcp.NewImageContent(img, "image/png"))
	}

	// 添加执行信息
	var info strings.Builder
	info.WriteString(fmt.Sprintf("退出码: %d", result.ExitCode))
	if result.TimedOut {
		info.WriteString(" ⚠️ 超时")
	}
	if result.Stderr != "" {
		info.WriteString(fmt.Sprintf("\n%s", result.Stderr))
	}
	contents = append(contents, mcp.NewTextContent(info.String()))

	return &mcp.CallToolResult{
		Content: contents,
	}, nil
}
