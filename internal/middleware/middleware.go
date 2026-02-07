// Package middleware 提供 HTTP 中间件，包括认证和日志。
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// APIKeys 有效的 API Key 列表
	APIKeys []string
	// Enabled 是否启用认证
	Enabled bool
	// HeaderName API Key 请求头名称，默认 "X-API-Key"
	HeaderName string
	// ExcludePaths 不需要认证的路径前缀
	ExcludePaths []string
}

// DefaultAuthConfig 返回默认认证配置
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		APIKeys:      []string{},
		Enabled:      false,
		HeaderName:   "X-API-Key",
		ExcludePaths: []string{"/"},
	}
}

// Auth 创建认证中间件
func Auth(config AuthConfig) func(http.Handler) http.Handler {
	// 构建 API Key 查找表
	validKeys := make(map[string]bool)
	for _, key := range config.APIKeys {
		if key != "" {
			validKeys[key] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果未启用认证，直接放行
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// 检查是否为排除路径
			for _, path := range config.ExcludePaths {
				if r.URL.Path == path || strings.HasPrefix(r.URL.Path, path+"/") {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 获取 API Key
			apiKey := r.Header.Get(config.HeaderName)
			if apiKey == "" {
				// 也支持 Bearer token 格式
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// 验证 API Key
			if apiKey == "" {
				http.Error(w, "缺少 API Key，请在请求头中添加 X-API-Key 或 Authorization: Bearer <key>", http.StatusUnauthorized)
				return
			}

			if !validKeys[apiKey] {
				http.Error(w, "无效的 API Key", http.StatusUnauthorized)
				return
			}

			// 认证通过，继续处理
			next.ServeHTTP(w, r)
		})
	}
}

// LogConfig 日志配置
type LogConfig struct {
	// Enabled 是否启用日志
	Enabled bool
	// LogBody 是否记录请求体（可能包含敏感信息）
	LogBody bool
}

// DefaultLogConfig 返回默认日志配置
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Enabled: true,
		LogBody: false,
	}
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger 创建日志中间件
func Logger(config LogConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// 包装 ResponseWriter 以捕获状态码
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 处理请求
			next.ServeHTTP(wrapped, r)

			// 计算耗时
			duration := time.Since(start)

			// 记录日志
			log.Printf("[%s] %s %s | %d | %v | %s",
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
				wrapped.statusCode,
				duration,
				r.Header.Get("User-Agent"),
			)
		})
	}
}

// Chain 串联多个中间件
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
