// Arch Linux 服务独立入口
// 通过 stdio 提供 MCP 服务，由主进程的 proxy 代理加载
package main

import (
	"log"

	"mcp-hub/services/archlinux"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("Arch Linux 服务启动中...")
	s := archlinux.NewService()
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务运行失败: %v", err)
	}
}