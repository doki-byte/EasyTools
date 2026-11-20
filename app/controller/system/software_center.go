package system

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	softwareSettingsFile = "software_settings.json"
	defaultCatalogURL    = "https://raw.githubusercontent.com/doki-byte/EasyTools/master/software.json"
	defaultAboutURL      = "https://raw.githubusercontent.com/doki-byte/EasyTools/master/about.json"
)

// SoftwareItem 软件条目
type SoftwareItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	LatestVersion string `json:"latestVersion"`
	ToolType      string `json:"toolType"`
	ToolLanguage  string `json:"toolLanguage"`
	LastUpdated   string `json:"lastUpdated"`
	Author        string `json:"author"`
	Homepage      string `json:"homepage"`
	IsNew         bool   `json:"isNew"`
}

// SoftwareCatalog 软件目录
type SoftwareCatalog struct {
	CatalogVersion string          `json:"catalogVersion"`
	UpdatedAt      string          `json:"updatedAt"`
	Source         string          `json:"source,omitempty"`
	Items          []SoftwareItem  `json:"items"`
	Raw            json.RawMessage `json:"-"`
}

// SoftwareSettings 持久化配置
type SoftwareSettings struct {
	CatalogURL string `json:"catalogURL,omitempty"` // 软件清单地址
}

// UpdateItem 更新说明条目
type UpdateItem struct {
	Date    string `json:"date"`
	Content string `json:"content"`
	IsNew   bool   `json:"isNew"`
}

// PersonItem 人员条目
type PersonItem struct {
	Name string `json:"name"`
	Src  string `json:"src"`
	URL  string `json:"url,omitempty"`
}

// AboutInfo 关于页面信息
type AboutInfo struct {
	Updates            []UpdateItem `json:"updates"`
	SpecialThanks      []PersonItem `json:"specialThanks"`
	FriendlySponsorship []PersonItem `json:"friendlySponsorship"`
	LatestVersion      string       `json:"latestVersion,omitempty"`
}

// SoftwareCenter 软件目录与配置中心
type SoftwareCenter struct {
	Base
}

// NewSoftwareCenter 初始化 SoftwareCenter
func NewSoftwareCenter() *SoftwareCenter {
	return &SoftwareCenter{}
}

// Startup 注入 ctx
func (s *SoftwareCenter) Startup(ctx context.Context) {
	s.Ctx = ctx
}

// ---------------------- 配置与目录 ----------------------

func softwareConfigDir() string {
	return util.NewUtil().PathExist(filepath.Join(util.NewUtil().GetAppPath(), "config"))
}

func settingsPath() string {
	return filepath.Join(softwareConfigDir(), softwareSettingsFile)
}

func defaultSettings() SoftwareSettings {
	return SoftwareSettings{
		CatalogURL: defaultCatalogURL,
	}
}

func loadSettings() SoftwareSettings {
	settings := defaultSettings()
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		_ = saveSettings(settings)
		return settings
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		_ = saveSettings(defaultSettings())
		return defaultSettings()
	}

	if settings.CatalogURL == "" {
		settings.CatalogURL = defaultCatalogURL
	}

	_ = saveSettings(settings)
	return settings
}

func saveSettings(settings SoftwareSettings) error {
	if err := os.MkdirAll(softwareConfigDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0644)
}

// ---------------------- catalog 读写 ----------------------

func fetchCatalogFromURL(url string) (*SoftwareCatalog, error) {
	if url == "" {
		return nil, errors.New("catalog url 为空")
	}

	client := proxy.GlobalProxyManager.GetHTTPClient()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "EasyTools-SoftwareCenter")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 catalog 失败: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var catalog SoftwareCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	catalog.Raw = data

	if catalog.UpdatedAt == "" {
		catalog.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	catalog.Source = url
	return &catalog, nil
}

// ---------------------- 对外暴露的方法 ----------------------

// RefreshSoftwareCatalog 强制从远端刷新软件目录
// 如果配置的URL获取失败，自动回退到默认仓库
func (s *SoftwareCenter) RefreshSoftwareCatalog() (*SoftwareCatalog, error) {
	settings := loadSettings()
	catalogURL := settings.CatalogURL
	if catalogURL == "" {
		catalogURL = defaultCatalogURL
	}

	// 尝试从配置的URL获取
	catalog, err := fetchCatalogFromURL(catalogURL)
	if err != nil {
		// 如果配置的URL不是默认URL，且获取失败，则尝试回退到默认URL
		if catalogURL != defaultCatalogURL {
			catalog, fallbackErr := fetchCatalogFromURL(defaultCatalogURL)
			if fallbackErr != nil {
				// 两个URL都失败，返回包含原始错误和回退错误的信息
				return nil, fmt.Errorf("从配置地址(%s)获取失败: %w; 回退到默认地址也失败: %w", catalogURL, err, fallbackErr)
			}
			// 回退成功，返回默认URL的数据
			return catalog, nil
		}
		// 如果已经是默认URL，直接返回错误
		return nil, err
	}

	return catalog, nil
}

// GetSoftwareSettings 读取软件中心配置
func (s *SoftwareCenter) GetSoftwareSettings() (*SoftwareSettings, error) {
	settings := loadSettings()
	return &settings, nil
}

// SaveSoftwareSettings 持久化软件中心配置
func (s *SoftwareCenter) SaveSoftwareSettings(settings SoftwareSettings) error {
	return saveSettings(settings)
}

// CheckSoftwareUpdates 直接返回远端 software.json 中的 items
// 如果配置的URL获取失败，自动回退到默认仓库
func (s *SoftwareCenter) CheckSoftwareUpdates() ([]SoftwareItem, error) {
	settings := loadSettings()

	// 使用配置的清单地址，如果未配置则使用默认地址
	catalogURL := settings.CatalogURL
	if catalogURL == "" {
		catalogURL = defaultCatalogURL
	}

	// 尝试从配置的URL获取
	catalog, err := fetchCatalogFromURL(catalogURL)
	if err != nil {
		// 如果配置的URL不是默认URL，且获取失败，则尝试回退到默认URL
		if catalogURL != defaultCatalogURL {
			catalog, fallbackErr := fetchCatalogFromURL(defaultCatalogURL)
			if fallbackErr != nil {
				// 两个URL都失败，返回包含原始错误和回退错误的信息
				return nil, fmt.Errorf("从配置地址(%s)获取失败: %w; 回退到默认地址也失败: %w", catalogURL, err, fallbackErr)
			}
			// 回退成功，返回默认URL的数据
			return catalog.Items, nil
		}
		// 如果已经是默认URL，直接返回错误
		return nil, fmt.Errorf("从远程仓库获取软件列表失败: %w", err)
	}

	return catalog.Items, nil
}

// GetAboutInfo 从远程仓库获取 about.json 信息
// 如果获取失败，返回错误（前端会使用本地内置数据）
func (s *SoftwareCenter) GetAboutInfo() (*AboutInfo, error) {
	client := proxy.GlobalProxyManager.GetHTTPClient()

	req, err := http.NewRequest(http.MethodGet, defaultAboutURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "EasyTools-AboutPage")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 about.json 失败: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var aboutInfo AboutInfo
	if err := json.Unmarshal(data, &aboutInfo); err != nil {
		return nil, err
	}

	// 如果没有设置最新版本，从 updates 中获取第一个
	if aboutInfo.LatestVersion == "" && len(aboutInfo.Updates) > 0 {
		aboutInfo.LatestVersion = aboutInfo.Updates[0].Date
	}

	return &aboutInfo, nil
}
