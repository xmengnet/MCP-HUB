// Package mcp 提供 MCP 服务的核心接口和注册机制。
package mcp

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"mcp-hub/internal/middleware"

	"github.com/mark3labs/mcp-go/server"
)

var (
	// defaultRegistry 全局默认注册器，用于 init() 自动注册
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
	// pendingServices 暂存 init() 中注册的服务（此时 Registry 可能未配置）
	pendingServices []MCPService
	pendingMu       sync.Mutex
)

// DefaultRegistry 返回全局默认注册器
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// SetDefaultRegistry 设置全局默认注册器（应在 main 中配置后调用）
func SetDefaultRegistry(r *Registry) {
	defaultRegistry = r
	// 注册所有待处理的服务
	pendingMu.Lock()
	defer pendingMu.Unlock()
	for _, svc := range pendingServices {
		if err := r.Register(svc); err != nil {
			log.Printf("注册服务失败: %v", err)
		}
	}
	pendingServices = nil
}

// Register 使用全局注册器注册服务（用于 init() 自动注册）
func Register(svc MCPService) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	pendingServices = append(pendingServices, svc)
}

// MustRegister 注册服务，失败时 panic
func MustRegister(svc MCPService) {
	Register(svc)
}

// Registry 管理多个 MCP 服务的注册和路由。
// 支持动态注册服务，每个服务通过独立的 HTTP 路径访问。
type Registry struct {
	services    map[string]MCPService
	mux         *http.ServeMux
	addr        string
	authConfig  middleware.AuthConfig
	logConfig   middleware.LogConfig
	middlewares []func(http.Handler) http.Handler
}

// RegistryOption 注册器配置选项
type RegistryOption func(*Registry)

// WithAuth 配置 API Key 认证
func WithAuth(apiKeys ...string) RegistryOption {
	return func(r *Registry) {
		r.authConfig.Enabled = true
		r.authConfig.APIKeys = apiKeys
	}
}

// WithLogger 启用请求日志
func WithLogger(enabled bool) RegistryOption {
	return func(r *Registry) {
		r.logConfig.Enabled = enabled
	}
}

// WithMiddleware 添加自定义中间件
func WithMiddleware(mw func(http.Handler) http.Handler) RegistryOption {
	return func(r *Registry) {
		r.middlewares = append(r.middlewares, mw)
	}
}

// NewRegistry 创建新的服务注册器
func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{
		services:    make(map[string]MCPService),
		mux:         http.NewServeMux(),
		authConfig:  middleware.DefaultAuthConfig(),
		logConfig:   middleware.DefaultLogConfig(),
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Register 注册一个 MCP 服务
func (r *Registry) Register(svc MCPService) error {
	if _, exists := r.services[svc.Path()]; exists {
		return fmt.Errorf("服务路径 %s 已注册", svc.Path())
	}

	// 创建 Streamable HTTP 服务器，配置为无状态模式
	httpServer := server.NewStreamableHTTPServer(
		svc.MCPServer(),
		server.WithStateLess(true),
	)

	// 注册到路由
	r.mux.Handle(svc.Path(), httpServer)
	r.services[svc.Path()] = svc

	log.Printf("已注册 MCP 服务: %s -> %s (%s)", svc.Name(), svc.Path(), svc.Description())

	return nil
}

// Handler 返回带中间件的 HTTP 处理器
func (r *Registry) Handler() http.Handler {
	var handler http.Handler = r.mux

	// 应用自定义中间件
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	// 应用认证中间件
	if r.authConfig.Enabled {
		handler = middleware.Auth(r.authConfig)(handler)
	}

	// 应用日志中间件（最外层）
	if r.logConfig.Enabled {
		handler = middleware.Logger(r.logConfig)(handler)
	}

	return handler
}

// ListServices 列出所有已注册的服务
func (r *Registry) ListServices() []MCPService {
	services := make([]MCPService, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, svc)
	}
	return services
}

// Start 启动 HTTP 服务器
func (r *Registry) Start(addr string) error {
	r.addr = addr

	// 添加根路径显示服务列表
	r.mux.HandleFunc("/", r.handleIndex)

	log.Println("=================================")
	log.Printf("  MCP 服务器启动于 %s", addr)
	log.Println("=================================")

	if r.authConfig.Enabled {
		log.Printf("🔐 认证已启用，需要 API Key")
	} else {
		log.Println("⚠️  认证未启用，服务公开访问")
	}

	if r.logConfig.Enabled {
		log.Println("📝 请求日志已启用")
	}

	log.Println("已注册的服务:")
	for _, svc := range r.services {
		log.Printf("  - %s: %s", svc.Name(), svc.Path())
	}

	return http.ListenAndServe(addr, r.Handler())
}

// handleIndex 处理根路径请求，显示服务列表
func (r *Registry) handleIndex(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

	authStatus := "🔓 公开访问"
	if r.authConfig.Enabled {
		authStatus = "🔐 需要 API Key"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>MCP 服务中心</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .status { background: #e8f5e9; padding: 10px; border-radius: 8px; margin-bottom: 20px; }
        .service { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 8px; }
        .service h3 { margin: 0 0 5px 0; color: #0066cc; }
        .service p { margin: 5px 0; color: #666; }
        .service code { background: #e0e0e0; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>🚀 MCP 服务中心</h1>
    <div class="status">状态: %s</div>
    <p>以下是已注册的 MCP 服务：</p>
`, authStatus)

	// 获取请求的 Host 构建完整 URL
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	host := req.Host

	for _, svc := range r.services {
		fmt.Fprintf(w, `
    <div class="service">
        <h3>%s</h3>
        <p>%s</p>
        <p>端点: <code>%s://%s%s</code></p>
    </div>
`, svc.Name(), svc.Description(), scheme, host, svc.Path())
	}

	fmt.Fprintf(w, `
</body>
</html>
`)
}
