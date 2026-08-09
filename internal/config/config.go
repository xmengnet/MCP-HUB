// Package config 提供配置文件解析功能。
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	// Addr 服务监听地址
	Addr string `yaml:"addr"`
	// APIKeys API Key 列表
	APIKeys []string `yaml:"api_keys"`
	// NoLog 禁用请求日志
	NoLog bool `yaml:"no_log"`
	// Services 外部 MCP 服务列表
	Services []ServiceConfig `yaml:"services"`
}

// ServiceConfig 外部 MCP 服务配置
type ServiceConfig struct {
	// Name 服务名称（显示名）
	Name string `yaml:"name"`
	// Description 服务描述
	Description string `yaml:"description"`
	// Command 启动命令（如 npx, python, node 等）
	Command string `yaml:"command"`
	// Args 命令参数
	Args []string `yaml:"args"`
	// Path HTTP 路径（如 /mcp/filesystem）
	Path string `yaml:"path"`
	// Env 环境变量
	Env map[string]string `yaml:"env"`
	// Sandbox 沙箱权限配置（nil 表示无限制）
	Sandbox *SandboxConfig `yaml:"sandbox,omitempty"`
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证服务配置
	for i, svc := range cfg.Services {
		if svc.Name == "" {
			return nil, fmt.Errorf("服务 #%d: name 不能为空", i+1)
		}
		if svc.Command == "" {
			return nil, fmt.Errorf("服务 %q: command 不能为空", svc.Name)
		}
		if svc.Path == "" {
			return nil, fmt.Errorf("服务 %q: path 不能为空", svc.Name)
		}
	}

	return &cfg, nil
}
