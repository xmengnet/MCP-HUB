// Package codeexec 提供安全的代码执行沙箱服务。
//
// 本文件实现基于 Docker 容器的代码执行器。
// 每次执行启动独立容器，提供 OS 级隔离：cap-drop all, no-new-privileges,
// 非 root 用户、网络隔离、资源限制。执行结束后自动清理容器和临时目录。
package codeexec

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ExecutionResult 代码执行结果
type ExecutionResult struct {
	// Stdout 标准输出
	Stdout string
	// Stderr 标准错误
	Stderr string
	// Images 提取的图片（base64 编码 + MIME）
	Images []ImageResult
	// ExitCode 退出码
	ExitCode int
	// TimedOut 是否超时
	TimedOut bool
	// Duration 执行耗时
	Duration time.Duration
}

// Executor Docker 容器代码执行器
type Executor struct {
	// config 沙箱配置
	config *Config
}

// NewExecutor 创建执行器
func NewExecutor(cfg *Config) *Executor {
	return &Executor{config: cfg}
}

// Execute 执行代码
//
// lang: 语言名（"python", "nodejs", "shell"）
// code: 代码内容
// timeoutSec: 超时秒数（0 表示使用沙箱默认值）
func (e *Executor) Execute(ctx context.Context, lang, code string, timeoutSec int) (*ExecutionResult, error) {
	sb, ok := e.config.Sandboxes[lang]
	if !ok {
		return nil, fmt.Errorf("不支持的语言: %s（支持: python, nodejs, shell）", lang)
	}

	if timeoutSec <= 0 {
		timeoutSec = sb.TimeoutSec
	}
	timeoutSec = clampTimeout(timeoutSec)

	// 1. 创建临时目录（宿主机）
	tmpDir, err := os.MkdirTemp("", "mcp-exec-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	// 确保临时目录可被容器内非 root 用户读写
	if err := os.Chmod(tmpDir, 0777); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("设置临时目录权限失败: %w", err)
	}

	// 执行结束清理临时目录
	defer os.RemoveAll(tmpDir)

	// 2. 写入代码文件
	codeContent := code
	if sb.WrapPreamble {
		codeContent = pythonPreamble + "\n" + code
	}
	scriptPath := filepath.Join(tmpDir, sb.Entrypoint)
	if err := os.WriteFile(scriptPath, []byte(codeContent), 0644); err != nil {
		return nil, fmt.Errorf("写入代码文件失败: %w", err)
	}
	// 确保文件可被容器内非 root 用户读取
	if err := os.Chmod(scriptPath, 0644); err != nil {
		return nil, fmt.Errorf("设置代码文件权限失败: %w", err)
	}

	// 3. 构建容器名（使用随机后缀避免冲突）
	containerName := fmt.Sprintf("mcp-exec-%s-%d", lang, rand.Intn(1000000))

	// 确保 Docker 可用，检查镜像是否存在（不存在则自动拉取或构建）
	if err := e.ensureImageAvailable(ctx, lang, sb); err != nil {
		return nil, fmt.Errorf("镜像准备失败: %w", err)
	}

	// 4. 创建容器
	// 创建带超时的 context
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	startTime := time.Now()
	result, err := e.runContainer(execCtx, sb, tmpDir, containerName)
	elapsed := time.Since(startTime)
	result.Duration = elapsed

	if err != nil {
		// 超时
		if execCtx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Stderr = fmt.Sprintf("执行超时（%d 秒）", timeoutSec)
			result.ExitCode = -1
		} else {
			result.Stderr = fmt.Sprintf("%s\n%v", result.Stderr, err)
			result.ExitCode = -1
		}
	}

	// 5. 提取图片（Python 沙箱可能生成图表文件）
	if sb.WrapPreamble && result.ExitCode == 0 {
		images, _ := extractImages(tmpDir)
		result.Images = images
	}

	return result, nil
}

// runContainer 创建并运行容器，返回执行结果
func (e *Executor) runContainer(ctx context.Context, sb *SandboxConfig, tmpDir, containerName string) (*ExecutionResult, error) {
	// 构建完整的执行命令（在容器内执行）
	// 等价于:
	// docker run --rm \
	//   --name mcp-exec-python-123456 \
	//   --network=none \
	//   --cap-drop=ALL \
	//   --security-opt=no-new-privileges:true \
	//   --memory=256m \
	//   --cpus=1.0 \
	//   --pids-limit=128 \
	//   --ulimit nofile=128:128 \
	//   -v /tmp/mcp-exec-xxx:/sandbox \
	//   -w /sandbox \
	//   mcp-hub/python-sandbox:latest \
	//   python -u main.py
	args := buildDockerRunArgs(sb, tmpDir, containerName)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecutionResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// docker run 退出码非 0 通常是代码本身的错误，stderr 已包含错误信息
			// 不返回 error，让上层正常处理
			return result, nil
		}
		// 其他错误（如 Docker 命令本身失败）
		return result, err
	}

	return result, nil
}

// buildDockerRunArgs 构建 docker run 命令参数
func buildDockerRunArgs(sb *SandboxConfig, tmpDir, containerName string) []string {
	args := []string{
		"run",
		"--rm", // 容器退出后自动删除
		"--name", containerName,
		// 安全限制
		"--network", sb.Network,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true",
		// 资源限制
		"--memory", fmt.Sprintf("%dm", sb.MemoryMB),
		"--cpus", strconv.FormatFloat(sb.CPUCores, 'f', -1, 64),
		"--pids-limit", strconv.Itoa(sb.PIDLimit),
		"--ulimit", fmt.Sprintf("nofile=%d:%d", sb.FileLimit, sb.FileLimit),
		// 工作目录挂载（宿主机临时目录 → 容器 /sandbox）
		"-v", fmt.Sprintf("%s:/sandbox", tmpDir),
		"-w", "/sandbox",
	}

	// 只读根文件系统（可选）
	if sb.ReadOnly {
		args = append(args, "--read-only")
		// 即使只读，/sandbox 和 /tmp 仍可写
		args = append(args, "--tmpfs", "/tmp:rw,noexec,nosuid,size=64m")
	}

	// 镜像名
	args = append(args, sb.Image)

	// 容器内执行命令
	args = append(args, sb.Command...)

	return args
}

// ensureImageAvailable 确保镜像可用：
//  1. 镜像已存在 → 直接使用
//  2. 不存在 → 尝试 docker pull（远程镜像）
//  3. pull 失败 → 尝试从本地 Dockerfile 自动构建（docker build）
//  4. 构建失败 → 返回清晰错误
func (e *Executor) ensureImageAvailable(ctx context.Context, lang string, sb *SandboxConfig) error {
	image := sb.Image

	// 1. 检查镜像是否已存在
	check := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if check.Run() == nil {
		return nil // 镜像已存在
	}

	// 2. 镜像不存在，尝试拉取（可能来自 registry）
	log.Printf("镜像 %s 不存在，尝试拉取...", image)
	pullCtx, pullCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer pullCancel()

	var pullErr bytes.Buffer
	pull := exec.CommandContext(pullCtx, "docker", "pull", image)
	pull.Stderr = &pullErr
	if pull.Run() == nil {
		log.Printf("镜像 %s 拉取完成", image)
		return nil
	}
	log.Printf("拉取镜像 %s 失败（%s），尝试本地构建...", image, strings.TrimSpace(pullErr.String()))

	// 3. 尝试从本地 Dockerfile 自动构建
	return e.buildSandboxImage(ctx, lang, sb)
}

// buildSandboxImage 从本地 Dockerfile 构建沙箱镜像
func (e *Executor) buildSandboxImage(ctx context.Context, lang string, sb *SandboxConfig) error {
	buildDir := filepath.Join(e.config.SandboxDir, lang)
	dockerfile := filepath.Join(buildDir, "Dockerfile")

	if _, err := os.Stat(dockerfile); err != nil {
		return fmt.Errorf("镜像 %s 拉取失败且本地 Dockerfile 不存在（%s）。"+
			"请先运行 make build-sandboxes 构建沙箱镜像，或设置 CODE_EXEC_%s_IMAGE 指定自定义镜像",
			sb.Image, dockerfile, strings.ToUpper(lang))
	}

	// 构建可能较慢（下载基础镜像 + 安装包），给 10 分钟超时
	log.Printf("正在自动构建沙箱镜像 %s（从 %s）...", sb.Image, dockerfile)
	buildCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var buildOut, buildErr bytes.Buffer
	build := exec.CommandContext(buildCtx, "docker", "build", "-t", sb.Image, buildDir)
	build.Stdout = &buildOut
	build.Stderr = &buildErr
	if err := build.Run(); err != nil {
		// 截断输出，避免错误信息过长
		errMsg := buildErr.String()
		if len(errMsg) > 500 {
			errMsg = errMsg[len(errMsg)-500:]
		}
		return fmt.Errorf("自动构建沙箱镜像失败: %v（%s）。"+
			"可手动运行 make build-sandbox/%s 查看完整构建日志", err, errMsg, lang)
	}
	log.Printf("沙箱镜像 %s 自动构建完成", sb.Image)
	return nil
}
