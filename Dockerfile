# 构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译（静态链接，适合 alpine）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mcp-server ./cmd/server

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装证书、时区和 Node.js（用于运行社区 MCP Server）
RUN apk --no-cache add ca-certificates tzdata nodejs npm

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制
COPY --from=builder /app/mcp-server .

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/ || exit 1

# 启动命令
ENTRYPOINT ["./mcp-server"]
CMD ["-addr", ":8080"]
