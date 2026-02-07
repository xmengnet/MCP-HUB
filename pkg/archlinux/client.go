// Package archlinux 提供 Arch Linux 官方仓库和 AUR 的 API 客户端。
package archlinux

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	officialBaseURL = "https://archlinux.org/packages/search/json/"
	aurBaseURL      = "https://aur.archlinux.org/rpc/v5"
)

// OfficialPackage 官方仓库包信息
type OfficialPackage struct {
	Repo       string `json:"repo"`
	PkgName    string `json:"pkgname"`
	PkgBase    string `json:"pkgbase"`
	PkgVer     string `json:"pkgver"`
	PkgRel     string `json:"pkgrel"`
	Epoch      int    `json:"epoch"`
	PkgDesc    string `json:"pkgdesc"`
	URL        string `json:"url"`
	Arch       string `json:"arch"`
	Packager   string `json:"packager"`
	LastUpdate string `json:"last_update"`
	FlagDate   string `json:"flag_date"`
	// 大小信息
	CompressedSize int64 `json:"compressed_size"`
	InstalledSize  int64 `json:"installed_size"`
}

// OfficialSearchResponse 官方仓库搜索响应
type OfficialSearchResponse struct {
	Version int               `json:"version"`
	Limit   int               `json:"limit"`
	Valid   bool              `json:"valid"`
	Results []OfficialPackage `json:"results"`
}

// AURPackage AUR 包信息
type AURPackage struct {
	ID             int     `json:"ID"`
	Name           string  `json:"Name"`
	PackageBaseID  int     `json:"PackageBaseID"`
	PackageBase    string  `json:"PackageBase"`
	Version        string  `json:"Version"`
	Description    string  `json:"Description"`
	URL            string  `json:"URL"`
	NumVotes       int     `json:"NumVotes"`
	Popularity     float64 `json:"Popularity"`
	OutOfDate      *int64  `json:"OutOfDate"` // Unix timestamp, null if not flagged
	Maintainer     string  `json:"Maintainer"`
	Submitter      string  `json:"Submitter"`
	FirstSubmitted int64   `json:"FirstSubmitted"`
	LastModified   int64   `json:"LastModified"`
	URLPath        string  `json:"URLPath"`
	// 详细信息（仅 info 接口返回）
	License       []string `json:"License"`
	Depends       []string `json:"Depends"`
	MakeDepends   []string `json:"MakeDepends"`
	OptDepends    []string `json:"OptDepends"`
	CheckDepends  []string `json:"CheckDepends"`
	Provides      []string `json:"Provides"`
	Conflicts     []string `json:"Conflicts"`
	Replaces      []string `json:"Replaces"`
	Groups        []string `json:"Groups"`
	Keywords      []string `json:"Keywords"`
	CoMaintainers []string `json:"CoMaintainers"`
}

// AURResponse AUR API 响应
type AURResponse struct {
	Version     int          `json:"version"`
	Type        string       `json:"type"`
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
	Error       string       `json:"error,omitempty"`
}

// Client Arch Linux API 客户端
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient 创建新的 API 客户端
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		userAgent:  "MCP-HUB/1.0 (Arch Linux Package Search)",
	}
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(reqURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return body, nil
}

// SearchOfficial 搜索官方仓库
// name: 包名（支持精确匹配）
// repo: 可选，指定仓库（core, extra, multilib 等）
func (c *Client) SearchOfficial(name string, repo string) ([]OfficialPackage, error) {
	params := url.Values{}
	params.Set("name", name)
	if repo != "" {
		params.Set("repo", repo)
	}

	reqURL := officialBaseURL + "?" + params.Encode()
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var result OfficialSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return result.Results, nil
}

// SearchOfficialByKeyword 按关键字搜索官方仓库（模糊搜索）
func (c *Client) SearchOfficialByKeyword(keyword string, repo string) ([]OfficialPackage, error) {
	params := url.Values{}
	params.Set("q", keyword)
	if repo != "" {
		params.Set("repo", repo)
	}

	reqURL := officialBaseURL + "?" + params.Encode()
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var result OfficialSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return result.Results, nil
}

// SearchAUR 搜索 AUR 包
// keyword: 搜索关键字（模糊匹配名称和描述）
func (c *Client) SearchAUR(keyword string) ([]AURPackage, error) {
	reqURL := fmt.Sprintf("%s/search/%s", aurBaseURL, url.PathEscape(keyword))
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var result AURResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("AUR API 错误: %s", result.Error)
	}

	return result.Results, nil
}

// GetAURInfo 获取 AUR 包详细信息
func (c *Client) GetAURInfo(pkgname string) (*AURPackage, error) {
	reqURL := fmt.Sprintf("%s/info/%s", aurBaseURL, url.PathEscape(pkgname))
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var result AURResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("AUR API 错误: %s", result.Error)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("未找到包: %s", pkgname)
	}

	return &result.Results[0], nil
}

// GetMaintainerPackages 获取维护者维护的所有 AUR 包
func (c *Client) GetMaintainerPackages(maintainer string) ([]AURPackage, error) {
	params := url.Values{}
	params.Set("by", "maintainer")
	reqURL := fmt.Sprintf("%s/search/%s?%s", aurBaseURL, url.PathEscape(maintainer), params.Encode())
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var result AURResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("AUR API 错误: %s", result.Error)
	}

	return result.Results, nil
}

// SuggestAUR 获取 AUR 包名建议（前缀匹配）
func (c *Client) SuggestAUR(prefix string) ([]string, error) {
	reqURL := fmt.Sprintf("%s/suggest/%s", aurBaseURL, url.PathEscape(prefix))
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var suggestions []string
	if err := json.Unmarshal(body, &suggestions); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return suggestions, nil
}

// FormatTimestamp 格式化 Unix 时间戳
func FormatTimestamp(ts int64) string {
	if ts == 0 {
		return "N/A"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// FormatSize 格式化文件大小
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
