// Package codeexec 提供安全的代码执行沙箱服务。
//
// 本文件定义注入到用户 Python 代码前的图表捕获 preamble。
//
// 安全说明：Docker 容器是主要安全边界（cap-drop all, no-new-privileges,
// network none, 非 root 用户）。此处不再做 Python 层的模块限制，
// 而是专注于自动捕获 matplotlib / plotly 等图表库生成的图片。
//
// 工作原理：劫持 plt.show() 和 fig.show()，将图表保存为 PNG 文件到 /sandbox
// 目录。执行结束后，宿主机的 extractImages() 会扫描该目录提取图片。
package codeexec

const pythonPreamble = `
# ─── MCP Hub 图表自动捕获 ─────────────────────────────────────
# 劫持 matplotlib 和 plotly 的 show() 方法，将图表保存为文件
# 执行结束后由宿主机扫描 /sandbox 目录提取图片

import os as _os
import sys as _sys

# 图表输出目录（容器内的挂载点，与宿主机临时目录映射）
_OUTPUT_DIR = "/sandbox"

def _capture_matplotlib():
    """劫持 matplotlib.pyplot.show()，将所有 figure 保存为 PNG"""
    try:
        import matplotlib
        matplotlib.use("Agg")
        import matplotlib.pyplot as _plt

        _plot_counter = [0]

        def _save_show(*args, **kwargs):
            for _fig_num in _plt.get_fignums():
                _fig = _plt.figure(_fig_num)
                _plot_counter[0] += 1
                _path = _os.path.join(_OUTPUT_DIR, f"_plot_{_plot_counter[0]}.png")
                _fig.savefig(_path, format='png', dpi=100, bbox_inches='tight')
                print(f"[图表已保存: {_path}]")
                _plt.close(_fig)

        _plt.show = _save_show
    except ImportError:
        pass

def _capture_plotly():
    """劫持 plotly Figure.show()，优先导出 PNG，失败时降级为 HTML 文件"""
    try:
        import plotly.graph_objects as _go

        _plotly_counter = [0]

        def _plotly_show(self, *args, **kwargs):
            _plotly_counter[0] += 1
            # 尝试导出 PNG（需要 kaleido + Chromium，默认镜像未安装）
            _png_path = _os.path.join(_OUTPUT_DIR, f"_plotly_{_plotly_counter[0]}.png")
            try:
                self.write_image(_png_path, width=1200, height=700, scale=2)
                print(f"[图表已保存: {_png_path}]")
                return
            except Exception:
                pass

            # 降级：导出 HTML 文件（浏览器可打开，但 MCP 无法直接显示）
            _html_path = _os.path.join(_OUTPUT_DIR, f"_plotly_{_plotly_counter[0]}.html")
            try:
                self.write_html(_html_path, include_plotlyjs=True, full_html=True)
                print(f"[plotly 图表: 当前环境缺少 kaleido+Chromium，已保存为 HTML: {_html_path}]")
                print("[提示] 如需返回图片，请改用 matplotlib/seaborn（plt.show() 会自动捕获）")
            except Exception as _e2:
                print(f"[plotly 图表导出失败: {_e2}]")

        _go.Figure.show = _plotly_show
    except ImportError:
        pass

def _hint_unavailable_packages():
    """提示未预装的常用包（打印到 stderr，不污染用户 stdout）"""
    _missing = []
    for _pkg, _display in (('plotly', 'plotly'), ('sklearn', 'scikit-learn'),
                           ('pyarrow', 'pyarrow'), ('sympy', 'sympy'),
                           ('pydantic', 'pydantic')):
        try:
            __import__(_pkg)
        except ImportError:
            _missing.append(_display)
    if _missing:
        _msg = ("[提示] 以下包未预装: " + ", ".join(_missing) +
                "。如需使用，修改 services/code-exec/sandboxes/python/Dockerfile "
                "后执行 make build-sandbox/python 重建镜像")
        _sys.stderr.write(_msg + "\n")

def _capture_seaborn():
    """seaborn 基于 matplotlib，无需单独劫持，matplotlib 劫持已覆盖"""
    pass

# 初始化捕获
_capture_matplotlib()
_capture_plotly()
_capture_seaborn()
_hint_unavailable_packages()

# ─── 用户代码开始 ─────────────────────────────────────────────
`
