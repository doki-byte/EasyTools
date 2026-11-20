package connect

import (
	"EasyTools/app/controller/system"
	extractorpkg "EasyTools/app/embed"
	"EasyTools/app/embed/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	configManagerOnce sync.Once
	configManager     *ConfigManager
)

// ConfigManager 配置管理器
type ConfigManager struct {
	baseDir string
	mu      sync.RWMutex
}

// GetConfigManager 获取配置管理器单例
func GetConfigManager() *ConfigManager {
	configManagerOnce.Do(func() {
		configManager = &ConfigManager{
			baseDir: system.GetAppBaseDir(),
		}
	})
	return configManager
}

// ensureConfigFile 确保配置文件存在
func (cm *ConfigManager) ensureConfigFile(filename string) error {
	configPath := filepath.Join(cm.baseDir, "config", filename)

	// 检查文件是否存在且非空
	if info, err := os.Stat(configPath); err == nil && info.Size() > 0 {
		return nil
	}

	// 优先使用解压器释放整个 config 目录
	extractorpkg.InitExtractor()
	if err := extractorpkg.GetExtractor().ExtractOne("config"); err != nil {
		// 如果解压失败，尝试直接读取嵌入文件
		if data, err := config.Assets.ReadFile(filename); err == nil {
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				return fmt.Errorf("创建配置目录失败: %v", err)
			}
			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
			return nil
		}
		return fmt.Errorf("解压配置失败: %v", err)
	}

	// 再次检查文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		// 从嵌入文件读取
		data, err := config.Assets.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("读取嵌入配置文件失败: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("创建配置目录失败: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("写入配置文件失败: %v", err)
		}
	}

	return nil
}

// LoadNoSQLCommands 加载NoSQL命令配置
func (cm *ConfigManager) LoadNoSQLCommands() (map[string][]NoSQLCommand, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if err := cm.ensureConfigFile("nosql_commands.json"); err != nil {
		return nil, err
	}

	configPath := filepath.Join(cm.baseDir, "config", "nosql_commands.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var commands map[string][]NoSQLCommand
	if err := json.Unmarshal(data, &commands); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return commands, nil
}

// SaveNoSQLCommands 保存NoSQL命令配置
func (cm *ConfigManager) SaveNoSQLCommands(commands map[string][]NoSQLCommand) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := cm.ensureConfigFile("nosql_commands.json"); err != nil {
		return err
	}

	configPath := filepath.Join(cm.baseDir, "config", "nosql_commands.json")
	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// LoadSQLTemplates 加载SQL模板配置
func (cm *ConfigManager) LoadSQLTemplates() ([]SQLTemplate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if err := cm.ensureConfigFile("sql_templates.json"); err != nil {
		return nil, err
	}

	configPath := filepath.Join(cm.baseDir, "config", "sql_templates.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var result struct {
		Templates []SQLTemplate `json:"templates"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return result.Templates, nil
}

// SaveSQLTemplates 保存SQL模板配置
func (cm *ConfigManager) SaveSQLTemplates(templates []SQLTemplate) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := cm.ensureConfigFile("sql_templates.json"); err != nil {
		return err
	}

	configPath := filepath.Join(cm.baseDir, "config", "sql_templates.json")
	data, err := json.MarshalIndent(struct {
		Templates []SQLTemplate `json:"templates"`
	}{Templates: templates}, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// NoSQLCommand NoSQL命令结构
type NoSQLCommand struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

// SQLTemplate SQL模板结构
type SQLTemplate struct {
	ID       string `json:"id"`
	DBType   string `json:"dbType"`
	Command  string `json:"command"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Template string `json:"template"`
}
