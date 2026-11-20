package config

import (
	"EasyTools/app/controller/connect/dbmanager/app/utils"
	"EasyTools/app/controller/system"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

type AppConfig struct {
	AppName   string `json:"app_name"`
	DbType    string `json:"db_type"`
	DbDsn     string `json:"db_dsn"`
	JwtSecret string `json:"jwt_secret"`
	AesSecret string `json:"aes_secret"`
	Address   string `json:"address"`
	Port      string `json:"port"`
}

var (
	appBaseDir = system.GetAppBaseDir()
	configFile = filepath.Join(appBaseDir, "config", "dbmanager_config.json")
)

var DefaultConfig = AppConfig{
	AppName:   "DBManager",
	DbType:    "sqlite3",
	DbDsn:     filepath.Join(appBaseDir, "config.db"),
	JwtSecret: utils.RandString(64),
	AesSecret: utils.RandString(32),
	Address:   "127.0.0.1",
	Port:      "52869",
}

func init() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("init error", err)
			os.Exit(255)
		}
	}()
	loadConfig()
}

// loadConfig 从文件加载配置，如果文件不存在则创建新配置并保存
func loadConfig() {
	// 尝试读取配置文件
	if data, err := os.ReadFile(configFile); err == nil {
		var savedConfig AppConfig
		if err := json.Unmarshal(data, &savedConfig); err == nil {
			// 使用已保存的配置
			if savedConfig.AesSecret != "" && len(savedConfig.AesSecret) == 32 {
				DefaultConfig.AesSecret = savedConfig.AesSecret
				slog.Info("已加载AES密钥从配置文件")
			}
			if savedConfig.JwtSecret != "" && len(savedConfig.JwtSecret) == 64 {
				DefaultConfig.JwtSecret = savedConfig.JwtSecret
			}
			if savedConfig.Address != "" {
				DefaultConfig.Address = savedConfig.Address
			}
			if savedConfig.Port != "" {
				DefaultConfig.Port = savedConfig.Port
			}
			if savedConfig.DbDsn != "" {
				DefaultConfig.DbDsn = savedConfig.DbDsn
			}
		}
	}

	// 保存配置到文件（如果密钥是新生成的）
	saveConfig()
}

// saveConfig 保存配置到文件
func saveConfig() {
	data, err := json.MarshalIndent(DefaultConfig, "", "  ")
	if err != nil {
		slog.Error("保存配置失败", "err", err)
		return
	}
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		slog.Error("写入配置文件失败", "err", err)
	} else {
		slog.Info("配置已保存到文件", "file", configFile)
	}
}
