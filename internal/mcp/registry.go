// Package mcp 提供 MCP 服务的核心接口和注册机制。
package mcp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"mcp-hub/internal/middleware"
	"mcp-hub/web"

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

	// 添加页面路由
	r.mux.HandleFunc("/health", r.handleHealth)
	r.mux.HandleFunc("/", r.handleIndex)
	r.mux.HandleFunc("/login", r.handleLogin)
	r.mux.HandleFunc("/logout", r.handleLogout)
	r.mux.HandleFunc("/playground", r.handlePlayground)

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

// handleHealth 处理健康检查请求
func (r *Registry) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"services": len(r.services),
	})
}

// handleIndex 处理根路径请求，显示服务列表
func (r *Registry) handleIndex(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

	type ServiceInfo struct {
		Name        string
		Description string
		Path        string
	}

	var services []ServiceInfo
	for _, svc := range r.services {
		services = append(services, ServiceInfo{
			Name:        svc.Name(),
			Description: svc.Description(),
			Path:        svc.Path(),
		})
	}

	data := struct {
		AuthEnabled bool
		Services    []ServiceInfo
	}{
		AuthEnabled: r.authConfig.Enabled,
		Services:    services,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := web.Templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "模板渲染错误: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleLogin 处理登录页面
func (r *Registry) handleLogin(w http.ResponseWriter, req *http.Request) {
	redirect := req.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	// POST 请求处理登录
	if req.Method == http.MethodPost {
		apiKey := req.FormValue("api_key")
		if apiKey == "" {
			r.renderLoginPage(w, redirect, "请输入 API Key")
			return
		}

		// 验证 API Key
		valid := false
		for _, key := range r.authConfig.APIKeys {
			if key == apiKey {
				valid = true
				break
			}
		}

		if !valid {
			r.renderLoginPage(w, redirect, "无效的 API Key")
			return
		}

		// 设置 Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "api_key",
			Value:    apiKey,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   86400 * 7, // 7天
			SameSite: http.SameSiteLaxMode,
		})

		// 重定向到目标页面
		http.Redirect(w, req, redirect, http.StatusFound)
		return
	}

	// GET 请求显示登录页面
	r.renderLoginPage(w, redirect, "")
}

// renderLoginPage 渲染登录页面
func (r *Registry) renderLoginPage(w http.ResponseWriter, redirect, errorMsg string) {
	data := struct {
		Redirect string
		Error    string
	}{
		Redirect: redirect,
		Error:    errorMsg,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	web.Templates.ExecuteTemplate(w, "login.html", data)
}

// handleLogout 处理登出
func (r *Registry) handleLogout(w http.ResponseWriter, req *http.Request) {
	// 清除 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "api_key",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, req, "/login", http.StatusFound)
}

// handlePlayground 处理 Playground 页面
func (r *Registry) handlePlayground(w http.ResponseWriter, req *http.Request) {
	type ServiceInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Path        string `json:"path"`
	}

	var services []ServiceInfo
	for _, svc := range r.services {
		services = append(services, ServiceInfo{
			Name:        svc.Name(),
			Description: svc.Description(),
			Path:        svc.Path(),
		})
	}

	servicesJSON, _ := json.Marshal(services)

	data := struct {
		ServicesJSON template.JS
	}{
		ServicesJSON: template.JS(servicesJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	web.Templates.ExecuteTemplate(w, "playground.html", data)
}
