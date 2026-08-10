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

	"mcp-hub/internal/config"
	"mcp-hub/internal/mcp"
	"mcp-hub/internal/middleware"
	"mcp-hub/internal/proxy"
)

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func main() {
	// 命令行参数（优先级：命令行 > 配置文件 > 环境变量 > 默认值）
	addr := flag.String("addr", "", "服务监听地址 (环境变量: ADDR)")
	apiKeys := flag.String("api-keys", "", "API Keys（逗号分隔） (环境变量: API_KEYS)")
	disableLog := flag.Bool("no-log", false, "禁用请求日志 (环境变量: NO_LOG)")
	configFile := flag.String("config", getEnv("CONFIG_FILE", ""), "配置文件路径 (环境变量: CONFIG_FILE)")
	flag.Parse()

	// 加载配置文件
	var cfg *config.Config
	if *configFile != "" {
		var err error
		cfg, err = config.Load(*configFile)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
	}

	// 解析最终值（命令行 > 配置文件 > 环境变量 > 默认值）
	finalAddr := resolveString(*addr, cfgAddr(cfg), getEnv("ADDR", ""), ":8080")
	finalAPIKeys := resolveString(*apiKeys, cfgAPIKeys(cfg), getEnv("API_KEYS", ""), "")
	finalNoLog := resolveBool(*disableLog, cfgNoLog(cfg), getEnvBool("NO_LOG", false))

	// 构建 Registry 选项
	var opts []mcp.RegistryOption

	if finalAPIKeys != "" {
		keys := strings.Split(finalAPIKeys, ",")
		for i := range keys {
			keys[i] = strings.TrimSpace(keys[i])
		}
		opts = append(opts, mcp.WithAuth(keys...))
	}

	if !finalNoLog {
		opts = append(opts, mcp.WithLogger(true))
	}

	// 创建 Prometheus 指标实例（如果配置启用）
	var metrics *middleware.Metrics
	if cfg != nil && cfg.Prometheus != nil && cfg.Prometheus.Enabled {
		metrics = middleware.NewMetrics()
		opts = append(opts, mcp.WithPrometheus(metrics, cfg.Prometheus))
	}

	// 创建服务注册器并设置为默认（触发内置服务自动注册）
	registry := mcp.NewRegistry(opts...)
	mcp.SetDefaultRegistry(registry)

	// 启动服务器（先启动，再后台加载服务）
	go func() {
		if err := registry.Start(finalAddr); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 后台加载外部代理服务
	var proxies []*proxy.ProxyService
	if cfg != nil && len(cfg.Services) > 0 {
		proxy.LoadAllAsync(cfg, registry, metrics, func(svc *proxy.ProxyService, err error) {
			if err != nil {
				log.Printf("⚠️  代理服务加载失败: %v", err)
				return
			}
			proxies = append(proxies, svc)
		})
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("收到关闭信号，正在关闭代理连接...")
		for _, p := range proxies {
			p.Close()
		}
		log.Println("服务即将停止...")
		os.Exit(0)
	}()

// 阻塞主 goroutine
		select {}
	}

	// --- 配置解析辅助函数 ---

// resolveString 按优先级取第一个非空值
func resolveString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// resolveBool 取第一个为 true 的值
func resolveBool(values ...bool) bool {
	for _, v := range values {
		if v {
			return true
		}
	}
	return false
}

// cfgAddr 安全获取配置文件中的 addr
func cfgAddr(cfg *config.Config) string {
	if cfg != nil {
		return cfg.Addr
	}
	return ""
}

// cfgAPIKeys 安全获取配置文件中的 api_keys（合并为逗号分隔字符串）
func cfgAPIKeys(cfg *config.Config) string {
	if cfg != nil && len(cfg.APIKeys) > 0 {
		return strings.Join(cfg.APIKeys, ",")
	}
	return ""
}

// cfgNoLog 安全获取配置文件中的 no_log
func cfgNoLog(cfg *config.Config) bool {
	if cfg != nil {
		return cfg.NoLog
	}
	return false
}
