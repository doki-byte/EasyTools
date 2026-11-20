package infoCollection

import (
	"EasyTools/app/controller/system"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 配置结构
type Config struct {
	APIKeys map[string]string      `json:"apiKeys"`
	Enabled map[string]bool        `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}

var configPath string

func init() {
	// 获取配置目录
	baseDir := system.GetAppBaseDir()
	configPath = filepath.Join(baseDir, "config", "info_collection.json")
}

// LoadConfigForProxy 提供给便携代理模块使用的只读访问接口（导出函数）
// 避免在 proxy 模块中直接依赖未导出的 loadConfig。
func LoadConfigForProxy() (*Config, error) {
	return loadConfig()
}

// loadConfig 加载配置
func loadConfig() (*Config, error) {
	config := &Config{
		APIKeys: make(map[string]string),
		Enabled: make(map[string]bool),
		Settings: map[string]interface{}{
			"maxPages":  5,
			"pageSize":  100,
			"delay":     1.0,
			"timeout":   30,
			"enableProxy": false,
		},
	}

	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// 解析JSON
	if err := json.Unmarshal(data, config); err != nil {
		return config, err
	}

	// 确保Settings有默认值
	if config.Settings == nil {
		config.Settings = map[string]interface{}{
			"maxPages":  5,
			"delay":     1.0,
			"timeout":   30,
			"enableProxy": false,
		}
	}

	return config, nil
}

// saveConfig 保存配置
func saveConfig(req ConfigRequest) error {
	// 先加载现有配置，以便在部分字段为空时保留现有值
	existingConfig, err := loadConfig()
	if err != nil {
		// 如果加载失败，使用默认配置
		existingConfig = &Config{
			APIKeys: make(map[string]string),
			Enabled: make(map[string]bool),
			Settings: map[string]interface{}{
				"maxPages":    5,
				"pageSize":    100,
				"delay":       1.0,
				"timeout":     30,
				"enableProxy": false,
			},
		}
	}

	// 合并配置：如果请求中的字段为空，则保留现有值
	config := &Config{
		APIKeys:  make(map[string]string),
		Enabled:  make(map[string]bool),
		Settings: make(map[string]interface{}),
	}

	// 合并APIKeys：如果请求中的apiKeys为空，则保留现有的
	if len(req.APIKeys) > 0 {
		config.APIKeys = req.APIKeys
	} else {
		config.APIKeys = existingConfig.APIKeys
	}

	// 合并Enabled：如果请求中的enabled为空，则保留现有的
	if len(req.Enabled) > 0 {
		config.Enabled = req.Enabled
	} else {
		config.Enabled = existingConfig.Enabled
	}

	// 合并Settings：如果请求中的settings为空，则保留现有的
	if len(req.Settings) > 0 {
		config.Settings = req.Settings
	} else {
		config.Settings = existingConfig.Settings
	}

	// 确保Settings有默认值
	if config.Settings == nil {
		config.Settings = map[string]interface{}{
			"maxPages":    5,
			"pageSize":    100,
			"delay":       1.0,
			"timeout":     30,
			"enableProxy": false,
		}
	}

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(configPath, data, 0644)
}
