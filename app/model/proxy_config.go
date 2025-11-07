package model

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

const TableNameProxyConfig = "proxy_config"

type ProxyConfig struct {
	Id             int    `json:"id" gorm:"column:id;primaryKey;autoIncrement:true"`
	GlobalEnabled  bool   `json:"global_enabled" gorm:"column:global_enabled"`
	ProxyType      string `json:"proxy_type" gorm:"column:proxy_type"`
	ProxyHost      string `json:"proxy_host" gorm:"column:proxy_host"`
	ProxyPort      string `json:"proxy_port" gorm:"column:proxy_port"`
	ProxyTimeout   int    `json:"proxy_timeout" gorm:"column:proxy_timeout;default:10"`
	ProxyUsername  string `json:"proxy_username" gorm:"column:proxy_username"`
	ProxyPassword  string `json:"proxy_password" gorm:"column:proxy_password"`
	ModuleSettings string `json:"module_settings" gorm:"column:module_settings;type:text"` // JSON格式存储模块设置
}

func (*ProxyConfig) TableName() string {
	return TableNameProxyConfig
}

func (p *ProxyConfig) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&ProxyConfig{}).Count(&count).Error; err != nil {
		log.Printf("Error counting proxy config records: %v", err)
		return
	}

	if count == 0 {
		defaultProxyConfig := ProxyConfig{
			Id:             1,
			GlobalEnabled:  false,
			ProxyType:      "http",
			ProxyHost:      "127.0.0.1",
			ProxyPort:      "8080",
			ProxyTimeout:   10,
			ProxyUsername:  "",
			ProxyPassword:  "",
			ModuleSettings: "{}", // 空的JSON对象
		}
		if err := db.Create(&defaultProxyConfig).Error; err != nil {
			log.Printf("Error inserting default proxy config data: %v", err)
		} else {
			log.Println("Default proxy config initialized")
		}
	}
}

// GetProxyConfig 获取代理配置
func (p *ProxyConfig) GetProxyConfig(db *gorm.DB) (*ProxyConfig, error) {
	var config ProxyConfig
	if err := db.Where("id = ?", 1).First(&config).Error; err != nil {
		return nil, fmt.Errorf("查询代理配置失败: %v", err)
	}
	return &config, nil
}

// UpdateProxyConfig 更新代理配置
func (p *ProxyConfig) UpdateProxyConfig(db *gorm.DB, config *ProxyConfig) error {
	if err := db.Where("id = ?", 1).Save(config).Error; err != nil {
		return fmt.Errorf("更新代理配置失败: %v", err)
	}
	return nil
}

// IsProxyEnabled 检查代理是否启用
func (p *ProxyConfig) IsProxyEnabled(db *gorm.DB) (bool, error) {
	var config ProxyConfig
	if err := db.Select("global_enabled").Where("id = ?", 1).First(&config).Error; err != nil {
		return false, fmt.Errorf("查询代理启用状态失败: %v", err)
	}
	return config.GlobalEnabled, nil
}
