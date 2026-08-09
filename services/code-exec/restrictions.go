// Package codeexec 提供安全的代码执行沙箱服务。
package codeexec

// SecurityPreamble 是注入到用户代码前的 Python 安全限制脚本。
//
// 注意：此限制仅作为防御纵深的一层，不可单独依赖。
// 主要安全由子进程隔离 + 环境过滤 + 资源限制提供。
const SecurityPreamble = `
import sys as _sys
import builtins as _builtins

# ─── 1. 禁止危险模块 ────────────────────────────────────────
_BLOCKED_MODULES = frozenset({
    'subprocess', 'ctypes', 'shutil', 'importlib',
    'inspect', 'socket', 'ssl', 'signal',
})

_ORIGINAL_IMPORT = _builtins.__import__

def _safe_import(name, *args, _orig=_ORIGINAL_IMPORT, _blocked=_BLOCKED_MODULES):
    top_name = name.split('.')[0]
    if top_name in _blocked:
        raise ImportError(f"模块 '{name}' 被禁止使用（安全沙箱限制）")
    return _orig(name, *args)

_builtins.__import__ = _safe_import

# ─── 2. 限制 os 模块 ────────────────────────────────────────
import os as _os

_BLOCKED_OS_ATTRS = frozenset({
    'system', 'popen', 'posix_spawn', 'spawnl', 'spawnle',
    'spawnlp', 'spawnlpe', 'spawnv', 'spawnve', 'spawnvp',
    'spawnvpe', 'execv', 'execl', 'execve', 'execle',
    'execlp', 'execvp', 'execvpe', 'fork', 'forkpty',
    'kill', 'killpg', 'remove', 'rmdir', 'unlink',
    'rename', 'chmod', 'chown', 'symlink', 'link',
    'makedirs', 'mkdir', 'truncate',
})

class _RestrictedOS:
    def __getattr__(self, name):
        if name in _BLOCKED_OS_ATTRS:
            raise PermissionError(f"os.{name} 被禁止使用（安全沙箱限制）")
        return getattr(_os, name)
    def __setattr__(self, name, value):
        raise PermissionError("不允许修改 os 模块属性")

_sys.modules['os'] = _RestrictedOS()

# ─── 3. 限制文件写入 ────────────────────────────────────────
import pathlib as _pathlib

_ORIGINAL_OPEN = _builtins.open
_CWD = _pathlib.Path.cwd().resolve()

def _safe_open(file, mode='r', *args, _orig=_ORIGINAL_OPEN, _cwd=str(_CWD), **kwargs):
    if 'w' in mode or 'a' in mode or 'x' in mode or '+' in mode:
        f = _pathlib.Path(file)
        if not f.is_absolute():
            f = _pathlib.Path(_cwd) / f
        f = f.resolve()
        if not str(f).startswith(_cwd):
            raise PermissionError(f"不允许在临时目录外写入文件: {file}")
    return _orig(file, mode, *args, **kwargs)

_builtins.open = _safe_open
`