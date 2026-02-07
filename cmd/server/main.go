// MCP 服务端入口
// 启动后可以通过 HTTP 协议访问各个 MCP 服务
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"mcp-hub/internal/mcp"
	"mcp-hub/services/workday"
)

func main() {
	// 命令行参数
	addr := flag.String("addr", ":8080", "服务监听地址")
	apiKeys := flag.String("api-keys", "", "API Keys（逗号分隔），留空则不启用认证")
	disableLog := flag.Bool("no-log", false, "禁用请求日志")
	flag.Parse()

	// 解析配置选项
	var opts []mcp.RegistryOption

	// API Key 认证
	if *apiKeys != "" {
		keys := strings.Split(*apiKeys, ",")
		// 清理空格
		for i := range keys {
			keys[i] = strings.TrimSpace(keys[i])
		}
		opts = append(opts, mcp.WithAuth(keys...))
	}

	// 请求日志
	if !*disableLog {
		opts = append(opts, mcp.WithLogger(true))
	}

	// 创建服务注册器
	registry := mcp.NewRegistry(opts...)

	// 注册 WorkDay 服务
	if err := registry.Register(workday.NewService()); err != nil {
		log.Fatalf("注册服务失败: %v", err)
	}

	// 未来可以注册更多服务:
	// registry.Register(weather.NewService())
	// registry.Register(calendar.NewService())

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("收到关闭信号，服务即将停止...")
		os.Exit(0)
	}()

	// 启动服务器
	if err := registry.Start(*addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
