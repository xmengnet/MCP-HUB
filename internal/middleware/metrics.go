// Package middleware 提供 HTTP 中间件，包括认证、日志和 Prometheus 指标。
package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 持有所有 Prometheus 指标收集器
type Metrics struct {
	registry        *prometheus.Registry
	requestTotal    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec
	responseSize    *prometheus.HistogramVec
	toolCallsTotal  *prometheus.CounterVec
	toolCallDuration *prometheus.HistogramVec
	serviceUp       *prometheus.GaugeVec
	serviceNameFunc func(path string) string
}

// NewMetrics 创建并注册所有 MCP 指标收集器
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	// 注册 Go 运行时和进程指标
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Metrics{registry: reg}

	// ─── HTTP 网关层指标 ──────────────────────────────────────────

m.requestTotal = promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "mcp_http_requests_total",
				Help: "HTTP 请求总数（注意：remote_addr 和 user_agent 为高基数标签，聚合查询时建议先按 service 或其他低基数标签汇总）",
			},
			[]string{"service", "method", "status_code", "remote_addr", "user_agent"},
		)

	m.requestDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_http_request_duration_seconds",
			Help:    "HTTP 请求耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	m.requestsInFlight = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mcp_http_requests_in_flight",
			Help: "当前正在处理的 HTTP 请求数",
		},
		[]string{"service"},
	)

	m.responseSize = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_http_response_size_bytes",
			Help:    "HTTP 响应体大小（字节）",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"service", "method"},
	)

	// ─── MCP 工具调用层指标 ───────────────────────────────────────

	m.toolCallsTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "MCP 工具调用总数",
		},
		[]string{"service", "tool", "status"},
	)

	m.toolCallDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_call_duration_seconds",
			Help:    "MCP 工具调用耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "tool"},
	)

	// ─── 服务生命周期指标 ─────────────────────────────────────────

	m.serviceUp = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mcp_service_up",
			Help: "服务在线状态（1=在线，0=离线）",
		},
		[]string{"service"},
	)

	return m
}

// SetServiceNameFunc 设置 URL 路径到服务名称的映射函数
func (m *Metrics) SetServiceNameFunc(fn func(path string) string) {
	m.serviceNameFunc = fn
}

// ExtractServiceName 从 URL 路径中提取服务名称
// 例如 "/mcp/workday" -> "workday", "/health" -> "health"
func ExtractServiceName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "mcp" {
		return parts[1]
	}
	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}
	return "root"
}

// metricsResponseWriter 包装 http.ResponseWriter 以捕获状态码和响应体大小
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// Middleware 返回一个记录 HTTP 层指标的中间件
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			service := m.resolveServiceName(r.URL.Path)

			// 提取客户端真实 IP（支持反向代理）
			remoteAddr := clientIP(r)
			// 提取 User-Agent
			userAgent := r.Header.Get("User-Agent")
			if userAgent == "" {
				userAgent = "unknown"
			}

			// 跟踪并发请求数
			m.requestsInFlight.WithLabelValues(service).Inc()
			defer m.requestsInFlight.WithLabelValues(service).Dec()

			start := time.Now()
			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			statusCode := strconv.Itoa(wrapped.statusCode)

			m.requestTotal.WithLabelValues(service, r.Method, statusCode, remoteAddr, userAgent).Inc()
			m.requestDuration.WithLabelValues(service, r.Method).Observe(duration.Seconds())
			m.responseSize.WithLabelValues(service, r.Method).Observe(float64(wrapped.bytesWritten))
		})
	}
}

// stripPort 从 RemoteAddr 中去除端口号，仅保留 IP
// 例如 "127.0.0.1:12345" -> "127.0.0.1"，"[::1]:12345" -> "::1"
func stripPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// clientIP 从请求中提取客户端真实 IP
// 优先顺序：X-Forwarded-For > X-Real-IP > RemoteAddr
func clientIP(r *http.Request) string {
	// 1. X-Forwarded-For（取第一个 IP，支持代理链）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if ip := strings.TrimSpace(ips[0]); ip != "" {
			return ip
		}
	}

	// 2. X-Real-IP（Nginx 常用）
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 3. 直连 IP
	return stripPort(r.RemoteAddr)
}

func (m *Metrics) resolveServiceName(path string) string {
	if m.serviceNameFunc != nil {
		return m.serviceNameFunc(path)
	}
	return ExtractServiceName(path)
}

// RecordToolCall 记录工具调用指标（由 proxy 层调用）
func (m *Metrics) RecordToolCall(service, tool, status string, duration time.Duration) {
	m.toolCallsTotal.WithLabelValues(service, tool, status).Inc()
	m.toolCallDuration.WithLabelValues(service, tool).Observe(duration.Seconds())
}

// SetServiceUp 设置服务在线状态
func (m *Metrics) SetServiceUp(service string, up bool) {
	val := 0.0
	if up {
		val = 1.0
	}
	m.serviceUp.WithLabelValues(service).Set(val)
}

// Handler 返回 /metrics 抓取端点的 HTTP 处理器
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}