# MCP Hub 多阶段构建
# 构建所有服务二进制到同一个镜像中
# 使用方式:
#   docker build -t mcp-hub .                    # 构建全部
#   docker build --target server -t mcp-hub .     # 仅主服务

# ─── 构建阶段 ──────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# ─── 主服务 ────────────────────────────────────────────────────
FROM builder AS server-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/mcp-server ./cmd/server

# ─── 工作日服务 ────────────────────────────────────────────────
FROM builder AS workday-svc-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/workday-svc ./services/workday/cmd/workday-svc

# ─── Arch Linux 服务 ───────────────────────────────────────────
FROM builder AS archlinux-svc-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/archlinux-svc ./services/archlinux/cmd/archlinux-svc

# ─── 代码执行沙箱服务 ──────────────────────────────────────────
FROM builder AS code-exec-svc-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/code-exec-svc ./services/code-exec/cmd/code-exec-svc

# ─── 全部构建 ──────────────────────────────────────────────────
FROM builder AS all-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/mcp-server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/workday-svc ./services/workday/cmd/workday-svc && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/archlinux-svc ./services/archlinux/cmd/archlinux-svc && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/code-exec-svc ./services/code-exec/cmd/code-exec-svc

# ─── 运行阶段 ──────────────────────────────────────────────────
FROM alpine:3.19 AS runtime

WORKDIR /app

# 安装运行时依赖
#   nodejs/npm — 用于运行社区 MCP Server（如 npx 包）
#   python3    — 用于代码执行沙箱服务
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    nodejs \
    npm \
    python3 \
    py3-pip \
    py3-numpy \
    py3-matplotlib \
    && rm -rf /var/cache/apk/*

# 设置时区
ENV TZ=Asia/Shanghai

# 从 all-builder 阶段复制所有二进制
COPY --from=all-builder /app/bin/ ./bin/

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/health || exit 1

# 默认启动主服务
ENTRYPOINT ["./bin/mcp-server"]
CMD ["-config", "/app/config.yaml"]