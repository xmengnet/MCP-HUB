// Package fetchsvc 提供 Web 内容抓取 MCP 服务。
package fetchsvc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewService 创建并返回配置好的 Web 抓取 MCP Server
func NewService() *server.MCPServer {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}

	s := server.NewMCPServer(
		"fetch-service",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(
		mcp.NewTool("fetch",
			mcp.WithDescription("获取网页内容，返回纯文本格式"),
			mcp.WithString("url",
				mcp.Required(),
				mcp.Description("要抓取的网页 URL"),
			),
			mcp.WithNumber("max_length",
				mcp.Description("返回内容最大长度（默认 10000）"),
			),
			mcp.WithNumber("timeout",
				mcp.Description("超时秒数（默认 15）"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleFetch(ctx, req, client)
		},
	)

	return s
}

func handleFetch(ctx context.Context, req mcp.CallToolRequest, client *http.Client) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	maxLength := 10000
	if v, err := req.RequireFloat("max_length"); err == nil && v > 0 {
		maxLength = int(v)
		if maxLength > 100000 {
			maxLength = 100000
		}
	}

	timeout := 15
	if v, err := req.RequireFloat("timeout"); err == nil && v > 0 {
		timeout = int(v)
		if timeout > 60 {
			timeout = 60
		}
	}

	// 验证 URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 创建带超时的请求
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("创建请求失败: %v", err)), nil
	}
	httpReq.Header.Set("User-Agent", "MCP-Hub-Fetch/1.0")
	httpReq.Header.Set("Accept", "text/html,text/plain,*/*")

	resp, err := client.Do(httpReq)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return mcp.NewToolResultError(fmt.Sprintf("请求超时（%d 秒）", timeout)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("请求失败: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)), nil
	}

	// 读取内容
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxLength)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("读取内容失败: %v", err)), nil
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// 构建结果
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📄 %s\n", url))
	sb.WriteString(fmt.Sprintf("📋 类型: %s\n", contentType))
	sb.WriteString(fmt.Sprintf("📏 大小: %d bytes\n", len(content)))
	sb.WriteString(fmt.Sprintf("✅ 状态: %d %s\n\n", resp.StatusCode, resp.Status))
	sb.WriteString(content)

	if len(content) >= maxLength {
		sb.WriteString(fmt.Sprintf("\n\n... (内容被截断，最大 %d bytes)", maxLength))
	}

	return mcp.NewToolResultText(sb.String()), nil
}