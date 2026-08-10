// 代码执行沙箱服务独立入口
// 通过 stdio 提供基于 Docker 容器隔离的代码执行 MCP 服务
// 支持 Python、Node.js、Shell 三种语言
package main

import (
	"log"

	codeexec "mcp-hub/services/code-exec"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("=================================")
	log.Println("  代码执行沙箱服务启动中...")
	log.Println("=================================")

	// 检测 Docker 是否可用
	if err := codeexec.CheckDocker(); err != nil {
		log.Printf("⚠️  Docker 检测失败: %v", err)
		log.Printf("⚠️  代码执行功能将不可用。请确保:")
		log.Printf("     1. Docker 已安装并运行")
		log.Printf("     2. Docker socket 可访问（容器内需挂载 /var/run/docker.sock）")
		log.Printf("     3. 沙箱镜像已构建（运行 make build-sandboxes）")
	} else {
		log.Println("✅ Docker 检测通过")
	}

	// 打印沙箱配置
	log.Println("沙箱配置:")
	codeexec.PrintConfig(codeexec.LoadConfig())

	log.Println("=================================")

	s := codeexec.NewService()
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务运行失败: %v", err)
	}
}
