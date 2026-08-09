// Package archlinux 提供 Arch Linux 软件包搜索 MCP 服务。
package archlinux

import (
	"context"
	"fmt"
	"strings"

	"mcp-hub/pkg/archlinux"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Service Arch Linux MCP 服务
type Service struct {
	client    *archlinux.Client
	mcpServer *server.MCPServer
}

// NewService 创建并返回配置好的 MCP Server（已注册所有工具）
func NewService() *server.MCPServer {
	s := &Service{
		client: archlinux.NewClient(),
	}

	// 创建 MCP 服务器
	s.mcpServer = server.NewMCPServer(
		"archlinux-service",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// 注册所有工具
	s.registerTools()

	return s.mcpServer
}

// registerTools 注册所有工具
func (s *Service) registerTools() {
	// 1. 搜索软件包
	s.mcpServer.AddTool(
		mcp.NewTool("search_package",
			mcp.WithDescription("搜索 Arch Linux 官方仓库和 AUR 的软件包"),
			mcp.WithString("keyword",
				mcp.Required(),
				mcp.Description("搜索关键字"),
			),
			mcp.WithString("repo",
				mcp.Description("指定仓库（core/extra/multilib），留空搜索所有"),
			),
			mcp.WithString("source",
				mcp.Description("搜索来源: official（官方仓库）、aur、all（默认，两者都搜）"),
			),
		),
		s.handleSearchPackage,
	)

	// 2. 获取包详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_package_info",
			mcp.WithDescription("获取软件包的详细信息，包括依赖、版本、维护者等"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("包名"),
			),
			mcp.WithString("source",
				mcp.Description("来源: official 或 aur，留空自动检测"),
			),
		),
		s.handleGetPackageInfo,
	)

	// 3. 获取维护者的包列表
	s.mcpServer.AddTool(
		mcp.NewTool("get_maintainer_packages",
			mcp.WithDescription("获取指定 AUR 维护者维护的所有软件包"),
			mcp.WithString("maintainer",
				mcp.Required(),
				mcp.Description("维护者用户名"),
			),
		),
		s.handleGetMaintainerPackages,
	)
}

// handleSearchPackage 处理搜索软件包请求
func (s *Service) handleSearchPackage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, err := req.RequireString("keyword")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	repo, _ := req.RequireString("repo")
	source, _ := req.RequireString("source")
	if source == "" {
		source = "all"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 搜索关键字: %s\n\n", keyword))

	// 搜索官方仓库
	if source == "all" || source == "official" {
		officialPkgs, err := s.client.SearchOfficialByKeyword(keyword, repo)
		if err != nil {
			sb.WriteString(fmt.Sprintf("⚠️ 官方仓库搜索失败: %s\n\n", err.Error()))
		} else if len(officialPkgs) == 0 {
			sb.WriteString("📂 官方仓库: 未找到匹配的软件包\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("📂 官方仓库 (%d 个结果):\n", len(officialPkgs)))
			// 最多显示 10 个
			limit := 10
			if len(officialPkgs) < limit {
				limit = len(officialPkgs)
			}
			for i := 0; i < limit; i++ {
				pkg := officialPkgs[i]
				sb.WriteString(fmt.Sprintf("  • %s/%s %s\n    %s\n",
					pkg.Repo, pkg.PkgName, pkg.PkgVer, pkg.PkgDesc))
			}
			if len(officialPkgs) > 10 {
				sb.WriteString(fmt.Sprintf("  ... 还有 %d 个结果\n", len(officialPkgs)-10))
			}
			sb.WriteString("\n")
		}
	}

	// 搜索 AUR
	if source == "all" || source == "aur" {
		aurPkgs, err := s.client.SearchAUR(keyword)
		if err != nil {
			sb.WriteString(fmt.Sprintf("⚠️ AUR 搜索失败: %s\n", err.Error()))
		} else if len(aurPkgs) == 0 {
			sb.WriteString("📦 AUR: 未找到匹配的软件包\n")
		} else {
			sb.WriteString(fmt.Sprintf("📦 AUR (%d 个结果):\n", len(aurPkgs)))
			// 最多显示 10 个
			limit := 10
			if len(aurPkgs) < limit {
				limit = len(aurPkgs)
			}
			for i := 0; i < limit; i++ {
				pkg := aurPkgs[i]
				outOfDate := ""
				if pkg.OutOfDate != nil {
					outOfDate = " ⚠️过期"
				}
				sb.WriteString(fmt.Sprintf("  • %s %s (⭐%d)%s\n    %s\n",
					pkg.Name, pkg.Version, pkg.NumVotes, outOfDate, pkg.Description))
			}
			if len(aurPkgs) > 10 {
				sb.WriteString(fmt.Sprintf("  ... 还有 %d 个结果\n", len(aurPkgs)-10))
			}
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// handleGetPackageInfo 处理获取包详情请求
func (s *Service) handleGetPackageInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	source, _ := req.RequireString("source")

	var sb strings.Builder

	// 如果未指定来源，先尝试官方仓库
	if source == "" || source == "official" {
		officialPkgs, err := s.client.SearchOfficial(name, "")
		if err == nil && len(officialPkgs) > 0 {
			pkg := officialPkgs[0]
			sb.WriteString("📂 来源: 官方仓库\n\n")
			sb.WriteString(fmt.Sprintf("📦 包名: %s\n", pkg.PkgName))
			sb.WriteString(fmt.Sprintf("📁 仓库: %s\n", pkg.Repo))
			sb.WriteString(fmt.Sprintf("📌 版本: %s-%s\n", pkg.PkgVer, pkg.PkgRel))
			sb.WriteString(fmt.Sprintf("📝 描述: %s\n", pkg.PkgDesc))
			sb.WriteString(fmt.Sprintf("🏗️ 架构: %s\n", pkg.Arch))
			sb.WriteString(fmt.Sprintf("👤 打包: %s\n", pkg.Packager))
			sb.WriteString(fmt.Sprintf("🔗 上游: %s\n", pkg.URL))
			sb.WriteString(fmt.Sprintf("📅 更新: %s\n", pkg.LastUpdate))
			if pkg.CompressedSize > 0 {
				sb.WriteString(fmt.Sprintf("💾 下载大小: %s\n", archlinux.FormatSize(pkg.CompressedSize)))
			}
			if pkg.InstalledSize > 0 {
				sb.WriteString(fmt.Sprintf("📀 安装大小: %s\n", archlinux.FormatSize(pkg.InstalledSize)))
			}
			sb.WriteString(fmt.Sprintf("🌐 链接: https://archlinux.org/packages/%s/%s/%s/\n",
				pkg.Repo, pkg.Arch, pkg.PkgName))

			return mcp.NewToolResultText(sb.String()), nil
		}
		if source == "official" {
			return mcp.NewToolResultError(fmt.Sprintf("在官方仓库中未找到包: %s", name)), nil
		}
	}

	// 尝试 AUR
	aurPkg, err := s.client.GetAURInfo(name)
	if err != nil {
		if source == "aur" {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("在官方仓库和 AUR 中均未找到包: %s", name)), nil
	}

	sb.WriteString("📦 来源: AUR\n\n")
	sb.WriteString(fmt.Sprintf("📦 包名: %s\n", aurPkg.Name))
	sb.WriteString(fmt.Sprintf("📌 版本: %s\n", aurPkg.Version))
	sb.WriteString(fmt.Sprintf("📝 描述: %s\n", aurPkg.Description))

	maintainer := aurPkg.Maintainer
	if maintainer == "" {
		maintainer = "孤儿包 (无维护者)"
	}
	sb.WriteString(fmt.Sprintf("👤 维护者: %s\n", maintainer))

	if len(aurPkg.CoMaintainers) > 0 {
		sb.WriteString(fmt.Sprintf("👥 共同维护: %s\n", strings.Join(aurPkg.CoMaintainers, ", ")))
	}

	sb.WriteString(fmt.Sprintf("🔗 上游: %s\n", aurPkg.URL))
	sb.WriteString(fmt.Sprintf("⭐ 投票: %d\n", aurPkg.NumVotes))
	sb.WriteString(fmt.Sprintf("📈 流行度: %.2f\n", aurPkg.Popularity))
	sb.WriteString(fmt.Sprintf("📅 提交: %s\n", archlinux.FormatTimestamp(aurPkg.FirstSubmitted)))
	sb.WriteString(fmt.Sprintf("📅 更新: %s\n", archlinux.FormatTimestamp(aurPkg.LastModified)))

	if aurPkg.OutOfDate != nil {
		sb.WriteString(fmt.Sprintf("⚠️ 过期: %s\n", archlinux.FormatTimestamp(*aurPkg.OutOfDate)))
	}

	if len(aurPkg.License) > 0 {
		sb.WriteString(fmt.Sprintf("📜 许可证: %s\n", strings.Join(aurPkg.License, ", ")))
	}

	if len(aurPkg.Depends) > 0 {
		sb.WriteString(fmt.Sprintf("📦 依赖: %s\n", strings.Join(aurPkg.Depends, ", ")))
	}

	if len(aurPkg.MakeDepends) > 0 {
		sb.WriteString(fmt.Sprintf("🔧 构建依赖: %s\n", strings.Join(aurPkg.MakeDepends, ", ")))
	}

	if len(aurPkg.OptDepends) > 0 {
		sb.WriteString(fmt.Sprintf("💡 可选依赖: %s\n", strings.Join(aurPkg.OptDepends, ", ")))
	}

	sb.WriteString(fmt.Sprintf("🌐 AUR 链接: https://aur.archlinux.org/packages/%s\n", aurPkg.Name))

	return mcp.NewToolResultText(sb.String()), nil
}

// handleGetMaintainerPackages 处理获取维护者包列表请求
func (s *Service) handleGetMaintainerPackages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	maintainer, err := req.RequireString("maintainer")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pkgs, err := s.client.GetMaintainerPackages(maintainer)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(pkgs) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("维护者 %s 没有维护任何 AUR 包", maintainer)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👤 维护者: %s\n", maintainer))
	sb.WriteString(fmt.Sprintf("📦 维护包数量: %d\n\n", len(pkgs)))

	for _, pkg := range pkgs {
		outOfDate := ""
		if pkg.OutOfDate != nil {
			outOfDate = " ⚠️过期"
		}
		sb.WriteString(fmt.Sprintf("• %s %s (⭐%d)%s\n  %s\n",
			pkg.Name, pkg.Version, pkg.NumVotes, outOfDate, pkg.Description))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
