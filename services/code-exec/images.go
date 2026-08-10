// Package codeexec 提供安全的代码执行沙箱服务。
//
// 本文件实现图片文件提取：从执行后的临时目录扫描图片文件，
// 读取并 base64 编码，作为 MCP 图片内容返回。
package codeexec

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 支持的图片扩展名（小写）
var imageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".svg":  true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
}

// MIME 类型映射
var mimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
}

// 单张图片大小限制：10MB（base64 编码前）
const maxImageSize = 10 * 1024 * 1024

// 最多提取的图片数量
const maxImageCount = 20

// ImageResult 图片提取结果
type ImageResult struct {
	// Data base64 编码的图片数据
	Data string
	// MIME MIME 类型
	MIME string
	// Filename 文件名（用于调试）
	Filename string
}

// extractImages 扫描临时目录，提取所有图片文件
//
// 按文件名排序，保证返回顺序稳定。
// 跳过过大的文件（> maxImageSize）和以 _ 开头的隐藏文件（系统文件）。
func extractImages(dir string) ([]ImageResult, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !imageExtensions[ext] {
			return nil
		}

		// 跳过过大文件
		if info.Size() > maxImageSize {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("扫描图片目录失败: %w", err)
	}

	// 按文件名排序，保证顺序稳定
	sort.Strings(files)

	// 限制数量
	if len(files) > maxImageCount {
		files = files[:maxImageCount]
	}

	var images []ImageResult
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // 跳过无法读取的文件
		}

		ext := strings.ToLower(filepath.Ext(path))
		mime := mimeTypes[ext]
		if mime == "" {
			mime = "application/octet-stream"
		}

		images = append(images, ImageResult{
			Data:     base64.StdEncoding.EncodeToString(data),
			MIME:     mime,
			Filename: filepath.Base(path),
		})
	}

	return images, nil
}
