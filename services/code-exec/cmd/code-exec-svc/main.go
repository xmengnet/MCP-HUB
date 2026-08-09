// 代码执行沙箱服务独立入口
// 通过 stdio 提供安全的 Python 代码执行 MCP 服务
package main

import (
	"log"

	"mcp-hub/services/code-exec"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("代码执行沙箱服务启动中...")
	s := codeexec.NewService()
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务运行失败: %v", err)
	}
}