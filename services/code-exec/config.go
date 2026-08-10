// Package codeexec 提供安全的代码执行沙箱服务。
//
// 本文件定义沙箱配置：镜像名、资源限制、网络模式、入口文件等。
// 支持通过环境变量覆盖默认值。
package codeexec

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SandboxConfig 单个语言沙箱的配置
type SandboxConfig struct {
	// Image Docker 镜像名（含 tag）
	Image string
	// Command 容器内执行命令（不含镜像名，如 ["python", "main.py"]）
	Command []string
	// Entrypoint 入口文件名（写入临时目录的文件名，如 "main.py"）
	Entrypoint string
	// MemoryMB 内存限制（MB）
	MemoryMB int
	// TimeoutSec 默认超时秒数
	TimeoutSec int
	// Network Docker 网络模式："none"（默认，无网络）或 "bridge"
	Network string
	// CPUCores CPU 核数限制（如 1.0 表示 1 核）
	CPUCores float64
	// PIDLimit 进程数限制
	PIDLimit int
	// FileLimit 文件描述符限制
	FileLimit int
	// ReadOnly 是否只读根文件系统（工作目录仍可写）
	ReadOnly bool
	// WrapPreamble 是否注入图表捕获 preamble（仅 Python 启用）
	WrapPreamble bool
}

// Config 代码执行服务的全局配置
type Config struct {
	// Sandboxes 按语言名索引的沙箱配置
	Sandboxes map[string]*SandboxConfig
}

// 默认资源限制常量
const (
	defaultMemoryPython = 512 // Python 数据科学包需要更多内存
	defaultMemoryNodejs = 256
	defaultMemoryShell  = 128

	defaultTimeout = 60   // 默认超时 60 秒
	maxTimeout     = 300  // 最大超时 300 秒
	defaultCPU     = 1.0  // 默认 1 核
	defaultPID     = 128  // 默认 128 进程
	defaultFiles   = 128  // 默认 128 文件描述符
)

// DefaultConfig 返回内置默认配置
func DefaultConfig() *Config {
	return &Config{
		Sandboxes: map[string]*SandboxConfig{
			"python": {
				Image:        "mcp-hub/python-sandbox:latest",
				Command:      []string{"python", "-u", "main.py"},
				Entrypoint:   "main.py",
				MemoryMB:     defaultMemoryPython,
				TimeoutSec:   defaultTimeout,
				Network:      "none",
				CPUCores:     defaultCPU,
				PIDLimit:     defaultPID,
				FileLimit:    defaultFiles,
				ReadOnly:     false,
				WrapPreamble: true,
			},
			"nodejs": {
				Image:        "mcp-hub/nodejs-sandbox:latest",
				Command:      []string{"node", "index.js"},
				Entrypoint:   "index.js",
				MemoryMB:     defaultMemoryNodejs,
				TimeoutSec:   defaultTimeout,
				Network:      "none",
				CPUCores:     defaultCPU,
				PIDLimit:     defaultPID,
				FileLimit:    defaultFiles,
				ReadOnly:     false,
				WrapPreamble: false,
			},
			"shell": {
				Image:        "mcp-hub/shell-sandbox:latest",
				Command:      []string{"/bin/bash", "script.sh"},
				Entrypoint:   "script.sh",
				MemoryMB:     defaultMemoryShell,
				TimeoutSec:   defaultTimeout,
				Network:      "none",
				CPUCores:     defaultCPU,
				PIDLimit:     defaultPID,
				FileLimit:    defaultFiles,
				ReadOnly:     false,
				WrapPreamble: false,
			},
		},
	}
}

// LoadConfig 加载配置：默认值 + 环境变量覆盖
//
// 支持的环境变量（以 Python 为例，其他语言替换 PYTHON 为 NODEJS/SHELL）:
//
//	CODE_EXEC_PYTHON_IMAGE      覆盖镜像名
//	CODE_EXEC_PYTHON_MEMORY     覆盖内存限制 (MB)
//	CODE_EXEC_PYTHON_TIMEOUT    覆盖默认超时 (秒)
//	CODE_EXEC_PYTHON_NETWORK    覆盖网络模式 ("none" 或 "bridge")
//	CODE_EXEC_PYTHON_CPU        覆盖 CPU 核数
func LoadConfig() *Config {
	cfg := DefaultConfig()

	for lang, sb := range cfg.Sandboxes {
		prefix := "CODE_EXEC_" + strings.ToUpper(lang)

		if v := os.Getenv(prefix + "_IMAGE"); v != "" {
			sb.Image = v
		}
		if v := os.Getenv(prefix + "_MEMORY"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				sb.MemoryMB = n
			}
		}
		if v := os.Getenv(prefix + "_TIMEOUT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				sb.TimeoutSec = clampTimeout(n)
			}
		}
		if v := os.Getenv(prefix + "_NETWORK"); v == "none" || v == "bridge" {
			sb.Network = v
		}
		if v := os.Getenv(prefix + "_CPU"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				sb.CPUCores = f
			}
		}
	}

	return cfg
}

// clampTimeout 将超时限制在合理范围内
func clampTimeout(sec int) int {
	if sec <= 0 {
		return defaultTimeout
	}
	if sec > maxTimeout {
		return maxTimeout
	}
	return sec
}

// CheckDocker 检测 Docker 是否可用
// 返回错误信息（如果不可用）或 nil（如果可用）
func CheckDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("未找到 docker 命令，请安装 Docker CLI 或将其加入 PATH")
	}
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法连接 Docker daemon（%v），请确认 Docker 已运行且 socket 可访问", err)
	}
	return nil
}

// PrintConfig 打印已加载的沙箱配置（用于启动日志）
func PrintConfig(cfg *Config) {
	for lang, sb := range cfg.Sandboxes {
		networkInfo := "无网络"
		if sb.Network == "bridge" {
			networkInfo = "桥接网络"
		}
		fmt.Printf("  📦 %s: 镜像=%s 内存=%dMB CPU=%.1f核 超时=%ds 网络=%s\n",
			lang, sb.Image, sb.MemoryMB, sb.CPUCores, sb.TimeoutSec, networkInfo)
	}
}
