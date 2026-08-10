# MCP Hub 多阶段构建
# 使用 Makefile 自动发现并构建所有服务二进制
# 新增服务只需创建 services/<name>/cmd/<name>/main.go，无需修改此文件
#
# 使用方式:
#   docker build -t mcp-hub .                    # 构建全部
#   docker build --target all-builder -t mcp-hub . # 构建所有二进制

# ─── 构建阶段 ──────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

# 安装 make
RUN apk --no-cache add make

# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码和 Makefile
COPY . .

# ─── 全部构建 ──────────────────────────────────────────────────
FROM builder AS all-builder
RUN CGO_ENABLED=0 GOOS=linux make build

# ─── 运行阶段 ──────────────────────────────────────────────────
FROM alpine:3.24 AS runtime

WORKDIR /app

# 安装运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    nodejs \
    npm \
    python3 \
    py3-pip \
    py3-numpy \
    py3-matplotlib \
    py3-uv \
    coreutils \
    docker-cli \
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
