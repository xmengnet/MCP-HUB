# Python 代码执行沙箱镜像
# 激进瘦身版：只保留 AI 代码生成场景最常用的核心包
# 目标大小约 400MB（全功能版约 1.3GB）
FROM python:3.13-slim

# 安装常用 Python 包（只留核心）
# - numpy/pandas: 数据处理
# - matplotlib/seaborn: 图表（自动捕获，无需额外依赖）
# - requests: HTTP 客户端
# - Pillow: 图像处理
#
# 有意省略的包（控制镜像大小）:
# - plotly/kaleido: 需要 Chromium(400-600MB) 才能导出图片，已降级为 HTML 输出
# - scikit-learn/pyarrow/sympy: 体积大(约 330MB)，按需可自行 docker build 扩展
RUN pip install --no-cache-dir \
    numpy \
    pandas \
    matplotlib \
    seaborn \
    requests \
    Pillow

# 创建非 root 用户
RUN useradd -m -s /bin/bash sandbox

# 设置工作目录（与宿主机临时目录的挂载点一致）
WORKDIR /sandbox

USER sandbox
