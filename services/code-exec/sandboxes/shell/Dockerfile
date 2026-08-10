# Shell 代码执行沙箱镜像
# 基于 Alpine，预装常用命令行工具
FROM alpine:3.24

# 安装常用工具
# - bash: 主 shell
# - curl/wget: HTTP 请求
# - jq: JSON 处理
# - coreutils: 完整版 GNU coreutils（替代 Alpine 的 busybox 版本）
# - findutils/grep/sed: 文本处理（awk 由 busybox 提供）
# - git: 版本控制
# - python3: 轻量脚本辅助
# - sqlite: 数据库操作
RUN apk add --no-cache \
    bash \
    curl \
    wget \
    jq \
    coreutils \
    findutils \
    grep \
    sed \
    git \
    python3 \
    sqlite

# 创建非 root 用户
RUN adduser -D -s /bin/bash sandbox

WORKDIR /sandbox

USER sandbox

# 默认使用 bash
CMD ["/bin/bash"]
