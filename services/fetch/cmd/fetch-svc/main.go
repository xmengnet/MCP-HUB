// Web 抓取服务独立入口
package main

import (
	"log"

	"mcp-hub/services/fetch"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("Web 抓取服务启动中...")
	s := fetchsvc.NewService()
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务运行失败: %v", err)
	}
}