// Package codeexec 提供安全的代码执行沙箱服务。
package codeexec

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ExecutionResult 代码执行结果
type ExecutionResult struct {
	Stdout   string   `json:"stdout"`
	Stderr   string   `json:"stderr"`
	Images   []string `json:"images,omitempty"`
	ExitCode int      `json:"exit_code"`
	TimedOut bool     `json:"timed_out"`
	Success  bool     `json:"success"`
}

// Executor 代码执行引擎
type Executor struct {
	PythonPath     string
	DefaultTimeout int
	MaxMemory      int
	MaxProcesses   int
}

// NewExecutor 创建默认执行器
func NewExecutor() *Executor {
	pythonPath := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		if p, err := exec.LookPath("python"); err == nil {
			pythonPath = p
		}
	}
	return &Executor{
		PythonPath:     pythonPath,
		DefaultTimeout: 30,
		MaxMemory:      256,
		MaxProcesses:   20,
	}
}

var imagePattern = regexp.MustCompile(`\[IMAGE_DATA_BEGIN\](.*?)\[IMAGE_DATA_END\]`)

// Execute 执行 Python 代码并返回结果
func (e *Executor) Execute(ctx context.Context, code string, timeoutSec int) (*ExecutionResult, error) {
	if timeoutSec <= 0 {
		timeoutSec = e.DefaultTimeout
	}

	tmpDir, err := os.MkdirTemp("", "mcp-exec-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	assembled := buildScript(code)
	scriptPath := filepath.Join(tmpDir, "_mcp_task.py")
	if err := os.WriteFile(scriptPath, []byte(assembled), 0644); err != nil {
		return nil, fmt.Errorf("写入脚本文件失败: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, e.PythonPath, scriptPath)
	cmd.Dir = tmpDir
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", tmpDir),
		fmt.Sprintf("TMPDIR=%s", tmpDir),
		"MPLBACKEND=Agg",
		"PYTHONIOENCODING=utf-8",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONHASHSEED=" + strconv.Itoa(rand.Intn(100000)),
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err = cmd.Run()
	elapsed := time.Since(startTime)

	result := &ExecutionResult{ExitCode: 0, TimedOut: false}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Stderr = fmt.Sprintf("执行超时（%d 秒）", timeoutSec)
		} else {
			result.Stderr = err.Error()
		}
	}

	output := stdout.String()
	images := extractImages(&output)
	result.Images = images
	result.Stdout = strings.TrimSpace(output)
	result.Stderr = strings.TrimSpace(stderr.String())
	result.Success = result.ExitCode == 0 && !result.TimedOut

	if result.Stderr != "" {
		result.Stderr = fmt.Sprintf("%s\n\n⏱ 耗时: %v", result.Stderr, elapsed.Round(time.Millisecond))
	} else {
		result.Stderr = fmt.Sprintf("⏱ 耗时: %v", elapsed.Round(time.Millisecond))
	}

	return result, nil
}

func buildScript(code string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString("# MCP Code Execution Sandbox\n\n")
	sb.WriteString(matplotlibPreamble)
	sb.WriteString("\n")
	sb.WriteString(SecurityPreamble)
	sb.WriteString("\n")
	sb.WriteString(code)
	sb.WriteString("\n")
	return sb.String()
}

const matplotlibPreamble = `
import matplotlib
matplotlib.use("Agg")
try:
    import matplotlib.pyplot as _plt
    _ORIGINAL_SHOW = _plt.show
    def _hijack_show(*args, **kwargs):
        import io as _io
        import base64 as _base64
        figs = _plt.get_fignums()
        for _fig_num in figs:
            _fig = _plt.figure(_fig_num)
            _buf = _io.BytesIO()
            _fig.savefig(_buf, format='png', dpi=100, bbox_inches='tight')
            _buf.seek(0)
            _img_data = _base64.b64encode(_buf.read()).decode('utf-8')
            _buf.close()
            _plt.clf()
            print(f"[IMAGE_DATA_BEGIN]{_img_data}[IMAGE_DATA_END]")
    _plt.show = _hijack_show
except ImportError:
    pass
`

func extractImages(output *string) []string {
	matches := imagePattern.FindAllStringSubmatch(*output, -1)
	if len(matches) == 0 {
		return nil
	}
	images := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			img := strings.TrimSpace(m[1])
			if _, err := base64.StdEncoding.DecodeString(img); err == nil {
				images = append(images, img)
			}
		}
	}
	*output = imagePattern.ReplaceAllString(*output, "")
	return images
}
