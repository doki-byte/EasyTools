package config

import (
	"EasyTools/app/controller/system"
	extractor "EasyTools/app/embed"
	embedconfig "EasyTools/app/embed/config"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Email     string
	FofaKey   string
	HunterKey string
	QuakeKey  string
	Maxpage   string

	// 搜索语句配置
	FofaSocks5Query   string
	FofaXuiQuery      string
	HunterSocks5Query string
	HunterXuiQuery    string
	QuakeSocks5Query  string
	QuakeXuiQuery     string

	CoroutineCount int
	LiveProxies    int
	AllProxies     int
	LiveProxyLists []string
	Timeout        string
	SocksAddress   string
	FilePath       string

	Status int

	Code  int
	Error string
	Xui   string
}

// 获取配置文件路径
func getConfigPath() string {
	configName := "proxy_config.json"
	baseDir := system.GetAppBaseDir()
	return filepath.Join(baseDir, "config", configName)
}

// 初始化配置
func initConfig() (*Config, error) {
	// 读取现有配置
	return loadConfig()
}

// 加载配置
func loadConfig() (*Config, error) {
	configPath := getConfigPath()

	// 确保默认配置文件存在（第一次运行或文件被删除时）
	if err := ensureDefaultProxyConfig(configPath); err != nil {
		return nil, fmt.Errorf("初始化默认配置失败: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		// 如果解析失败，尝试恢复为内置默认配置后再解析一次
		log.Printf("解析代理配置失败，尝试恢复默认配置: %v", err)
		if err := resetProxyConfig(configPath); err != nil {
			return nil, fmt.Errorf("解析配置文件失败且恢复默认配置失败: %v", err)
		}

		data, err = os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("重新读取默认配置失败: %v", err)
		}

		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析默认配置失败: %v", err)
		}
	}

	return &config, nil
}

// ensureDefaultProxyConfig 确保本地存在默认的 proxy_config.json
func ensureDefaultProxyConfig(configPath string) error {
	info, err := os.Stat(configPath)
	if err == nil && info.Size() > 0 {
		// 文件已存在且非空
		return nil
	}

	// 优先使用解压器释放整个 config 目录（包含 cms_finger.db 等所有文件）
	extractor.InitExtractor()
	if err := extractor.GetExtractor().ExtractOne("config"); err != nil {
		log.Printf("解压 config 资源失败，将尝试直接写入默认 proxy_config.json: %v", err)
	}

	// 如果解压后文件已经存在且非空，直接返回
	info, err = os.Stat(configPath)
	if err == nil && info.Size() > 0 {
		return nil
	}

	// 兜底：单独写入内置默认 proxy_config.json，避免第一次运行失败
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	data, err := embedconfig.Assets.ReadFile("proxy_config.json")
	if err != nil {
		return fmt.Errorf("读取内置默认配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入默认配置失败: %v", err)
	}

	return nil
}

// resetProxyConfig 使用内置默认配置覆盖本地配置（用于文件损坏时的恢复）
func resetProxyConfig(configPath string) error {
	// 从内置资源中读取默认配置
	data, err := embedconfig.Assets.ReadFile("proxy_config.json")
	if err != nil {
		return fmt.Errorf("读取内置默认配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入默认配置失败: %v", err)
	}

	return nil
}

func GetConfig() *Config {
	config, err := initConfig()
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}
	return config
}

func (p *Config) SaveConfig() error {

	configPath := getConfigPath()
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

func (p *Config) GetProfile() Config {
	return *GetConfig()
}

func (p *Config) GetCoroutineCount() int {
	return p.CoroutineCount
}

func (p *Config) GetLiveProxies() int {
	return p.LiveProxies
}

func (p *Config) SetAllProxies(datasets []string) {
	p.AllProxies = len(datasets)
	p.LiveProxyLists = datasets
}

func (p *Config) SetLiveProxies(datasets []string) {
	p.LiveProxyLists = datasets
	p.LiveProxies = len(datasets)
}

func (p *Config) GetTimeout() string { return p.Timeout }

func (p *Config) GetSocksAddress() string {
	return p.SocksAddress
}

func (p *Config) GetStatus() int {
	return p.Status
}

func (p *Config) SetStatus(i int) {
	p.Status = i
}
