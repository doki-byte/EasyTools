//go:build !windows

package proxy

import "fmt"

// SystemProxyConfig 系统代理配置
type SystemProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Type     string `json:"type"`               // "http", "https", "socks5"
	Override string `json:"override,omitempty"` // 例外列表，用分号分隔
}

// SetSystemProxy 设置系统代理（非 Windows 平台不支持）
func SetSystemProxy(config SystemProxyConfig) error {
	return fmt.Errorf("系统代理功能仅在 Windows 平台支持")
}

// GetSystemProxy 获取当前系统代理设置（非 Windows 平台不支持）
func GetSystemProxy() (SystemProxyConfig, error) {
	return SystemProxyConfig{}, fmt.Errorf("系统代理功能仅在 Windows 平台支持")
}

// IsSystemProxySupported 检查系统代理是否支持
func IsSystemProxySupported() bool {
	return false
}
