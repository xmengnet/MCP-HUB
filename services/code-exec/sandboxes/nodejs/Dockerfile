# Node.js 代码执行沙箱镜像
# 支持 CommonJS、ESM 和 TypeScript（通过 tsx）
FROM node:22-slim

# 启用 corepack（管理 yarn/pnpm 版本）
RUN corepack enable

# 预装 TypeScript 运行时工具
# - tsx: 直接运行 .ts 文件（基于 esbuild，速度快）
# - typescript: TypeScript 编译器
RUN npm install -g tsx typescript

# 创建非 root 用户
RUN useradd -m -s /bin/bash sandbox

# 设置工作目录并授权给 sandbox 用户（代码通过 docker cp 传入，非 bind mount）
WORKDIR /sandbox
RUN chown sandbox:sandbox /sandbox

USER sandbox
