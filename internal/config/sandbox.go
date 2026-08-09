// Package config 提供配置文件解析功能。
package config

import "time"

// SandboxConfig 定义 MCP 服务的沙箱权限配置。
// 不配置（nil）表示无限制，继承宿主全部权限。
type SandboxConfig struct {
	// Network 网络访问控制
	Network *NetworkConfig `yaml:"network,omitempty"`

	// FS 文件系统访问控制
	FS *FSConfig `yaml:"fs,omitempty"`

	// PrivateTmp 是否使用独立临时目录
	PrivateTmp bool `yaml:"private_tmp,omitempty"`

	// Memory 内存上限，如 "256MB"、"1GB"
	Memory string `yaml:"memory,omitempty"`

	// CPU CPU 核数限制，如 0.5 表示半核
	CPU float64 `yaml:"cpu,omitempty"`

	// Timeout 每次请求超时时间，如 "30s"
	Timeout string `yaml:"timeout,omitempty"`

	// Env 环境变量控制
	Env *EnvConfig `yaml:"env,omitempty"`
}

// NetworkConfig 网络访问控制
type NetworkConfig struct {
	// Enabled 是否允许联网（默认 false 禁止联网）
	Enabled bool `yaml:"enabled"`
	// Egress 白名单出站地址，如 ["api.github.com:443"]
	Egress []string `yaml:"egress,omitempty"`
}

// FSConfig 文件系统访问控制
type FSConfig struct {
	// Mode 访问模式: "read-only" | "whitelist" | "full"
	//   read-only: 只读
	//   whitelist: 只允许读写指定路径
	//   full: 完全访问（默认）
	Mode string `yaml:"mode"`
	// ReadWrite 读写白名单路径列表
	ReadWrite []string `yaml:"read_write,omitempty"`
	// ReadOnly 只读白名单路径列表
	ReadOnly []string `yaml:"read_only,omitempty"`
}

// EnvConfig 环境变量控制
type EnvConfig struct {
	// Inherit 是否继承宿主环境变量（默认 false）
	Inherit bool `yaml:"inherit"`
	// Allow 白名单环境变量列表，Inherit=true 时此为额外保留列表
	Allow []string `yaml:"allow,omitempty"`
}

// ParseTimeout 解析超时配置，返回 time.Duration
func (s *SandboxConfig) ParseTimeout() (time.Duration, error) {
	if s == nil || s.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(s.Timeout)
}

// DefaultFSMode 返回默认文件系统模式
const DefaultFSMode = "full"

// IsReadOnly 判断是否只读模式
func (f *FSConfig) IsReadOnly() bool {
	return f != nil && f.Mode == "read-only"
}

// IsWhitelist 判断是否白名单模式
func (f *FSConfig) IsWhitelist() bool {
	return f != nil && f.Mode == "whitelist"
}