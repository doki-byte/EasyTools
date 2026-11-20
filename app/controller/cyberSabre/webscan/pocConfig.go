package webscan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"EasyTools/app/controller/system"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PocConfigController POC 配置控制器
type PocConfigController struct {
	ctx context.Context
	system.Base
}

// NewPocConfigController 创建 POC 配置控制器
func NewPocConfigController() *PocConfigController {
	return &PocConfigController{}
}

// Startup 初始化上下文
func (p *PocConfigController) Startup(ctx context.Context) {
	p.ctx = ctx
}

// PocConfig POC 配置结构
type PocConfig struct {
	TemplateFolders []string `json:"template_folders"` // POC 模板文件夹路径列表
	AutoReload      bool     `json:"auto_reload"`      // 是否自动重新加载标签
	DefaultHeaders  string   `json:"default_headers"`  // 默认请求头
	// 扫描配置（用于漏洞扫描 / 指纹扫描）
	Threads   int `json:"threads"`    // 扫描线程数
	Timeout   int `json:"timeout"`    // 超时时间（秒）
	RateLimit int `json:"rate_limit"` // 速率限制（每秒请求数）
	Retries   int `json:"retries"`    // 重试次数
}

// GetPocConfig 获取 POC 配置
func (p *PocConfigController) GetPocConfig() (*PocConfig, error) {
	configPath := p.getConfigPath()

	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return p.applyConfigDefaults(p.getDefaultConfig()), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config PocConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return p.applyConfigDefaults(&config), nil
}

// SavePocConfig 保存 POC 配置
func (p *PocConfigController) SavePocConfig(configJSON string) error {
	var config PocConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}
	p.applyConfigDefaults(&config)

	// 验证模板文件夹路径
	for _, folder := range config.TemplateFolders {
		if folder != "" {
			if _, err := os.Stat(folder); os.IsNotExist(err) {
				return fmt.Errorf("模板文件夹不存在: %s", folder)
			}
		}
	}

	if err := p.saveConfig(&config); err != nil {
		return err
	}

	// 如果启用了自动重新加载，则重新加载标签
	if config.AutoReload {
		if err := ReloadPocTags(p.ctx, config.TemplateFolders); err != nil {
			runtime.EventsEmit(p.ctx, "log_event", map[string]interface{}{
				"level":   "warning",
				"message": fmt.Sprintf("重新加载 POC 标签失败: %v", err),
				"source":  "POC配置",
			})
		} else {
			runtime.EventsEmit(p.ctx, "log_event", map[string]interface{}{
				"level":   "info",
				"message": "POC 标签已重新加载",
				"source":  "POC配置",
			})
		}
	}

	return nil
}

// ReloadPocTags 重新加载 POC 标签
func (p *PocConfigController) ReloadPocTags() error {
	config, err := p.GetPocConfig()
	if err != nil {
		return err
	}

	if err := ReloadPocTags(p.ctx, config.TemplateFolders); err != nil {
		return fmt.Errorf("重新加载标签失败: %v", err)
	}

	runtime.EventsEmit(p.ctx, "log_event", map[string]interface{}{
		"level":   "info",
		"message": "POC 标签已重新加载",
		"source":  "POC配置",
	})

	return nil
}

// ValidateTemplateFolder 验证模板文件夹
func (p *PocConfigController) ValidateTemplateFolder(folderPath string) map[string]interface{} {
	result := map[string]interface{}{
		"valid":   false,
		"message": "",
		"count":   0,
	}

	if folderPath == "" {
		result["message"] = "路径不能为空"
		return result
	}

	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			result["message"] = "文件夹不存在"
		} else {
			result["message"] = fmt.Sprintf("访问文件夹失败: %v", err)
		}
		return result
	}

	if !info.IsDir() {
		result["message"] = "路径不是文件夹"
		return result
	}

	// 统计 YAML 文件数量
	count := 0
	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			count++
		}
		return nil
	})

	result["valid"] = true
	result["message"] = "验证成功"
	result["count"] = count

	return result
}

// getConfigPath 获取配置文件路径
func (p *PocConfigController) getConfigPath() string {
	appPath := p.GetAppPath()
	return filepath.Join(appPath, "config", "poc_config.json")
}

// getDefaultConfig 获取默认配置
func (p *PocConfigController) getDefaultConfig() *PocConfig {
	appPath := p.GetAppPath()
	defaultPocPath := filepath.Join(appPath, "tools", "nuclei-templates")
	folders := []string{}
	// 先把内置 nuclei 模板加入默认列表
	if _, err := os.Stat(defaultPocPath); err == nil {
		folders = append(folders, defaultPocPath)
	}

	return &PocConfig{
		TemplateFolders: folders,
		AutoReload:      true,
		DefaultHeaders:  "",
		Threads:         5,
		Timeout:         10,
		RateLimit:       150,
		Retries:         1,
	}
}

// GetPocTemplateFolders 获取 POC 模板文件夹列表（供其他模块使用）
func (p *PocConfigController) GetPocTemplateFolders() ([]string, error) {
	config, err := p.GetPocConfig()
	if err != nil {
		return nil, err
	}
	return config.TemplateFolders, nil
}

// applyConfigDefaults 补齐缺省配置
func (p *PocConfigController) applyConfigDefaults(cfg *PocConfig) *PocConfig {
	if cfg == nil {
		return p.getDefaultConfig()
	}
	if cfg.TemplateFolders == nil {
		cfg.TemplateFolders = []string{}
	}
	if cfg.Threads <= 0 {
		cfg.Threads = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	if cfg.RateLimit <= 0 {
		cfg.RateLimit = 150
	}
	if cfg.Retries < 0 {
		cfg.Retries = 1
	}
	return cfg
}

// saveConfig 序列化配置到磁盘
func (p *PocConfigController) saveConfig(cfg *PocConfig) error {
	cfg = p.applyConfigDefaults(cfg)

	configPath := p.getConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}
	return nil
}
