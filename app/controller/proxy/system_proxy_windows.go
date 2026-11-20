//go:build windows

package proxy

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// SystemProxyConfig 系统代理配置
type SystemProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Type     string `json:"type"`               // "http", "https", "socks5"
	Override string `json:"override,omitempty"` // 例外列表，用分号分隔
}

// SetSystemProxy 设置系统代理（Windows）
func SetSystemProxy(config SystemProxyConfig) error {
	if !config.Enabled {
		return disableSystemProxy()
	}

	// 验证配置
	if config.Host == "" || config.Port == "" {
		return fmt.Errorf("代理服务器地址和端口不能为空")
	}

	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 构建代理服务器地址
	var proxyServer string
	switch strings.ToLower(config.Type) {
	case "http", "https":
		proxyServer = fmt.Sprintf("%s=%s:%s", strings.ToLower(config.Type), config.Host, config.Port)
	case "socks5", "socks":
		proxyServer = fmt.Sprintf("socks=%s:%s", config.Host, config.Port)
	default:
		// 默认使用 HTTP
		proxyServer = fmt.Sprintf("http=%s:%s", config.Host, config.Port)
	}

	// 设置代理服务器
	if err := key.SetStringValue("ProxyServer", proxyServer); err != nil {
		return fmt.Errorf("设置代理服务器失败: %v", err)
	}

	// 启用代理
	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("启用代理失败: %v", err)
	}

	// 设置例外列表（如果有）
	if config.Override != "" {
		if err := key.SetStringValue("ProxyOverride", config.Override); err != nil {
			return fmt.Errorf("设置例外列表失败: %v", err)
		}
	} else {
		// 默认例外：本地地址
		defaultOverride := "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;<local>"
		if err := key.SetStringValue("ProxyOverride", defaultOverride); err != nil {
			return fmt.Errorf("设置默认例外列表失败: %v", err)
		}
	}

	// 通知系统代理设置已更改
	notifyProxyChange()

	return nil
}

// GetSystemProxy 获取当前系统代理设置（Windows）
func GetSystemProxy() (SystemProxyConfig, error) {
	var config SystemProxyConfig

	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.READ)
	if err != nil {
		return config, fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 读取代理启用状态
	enabled, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		// 如果键不存在，默认未启用
		config.Enabled = false
		return config, nil
	}
	config.Enabled = enabled != 0

	if !config.Enabled {
		return config, nil
	}

	// 读取代理服务器地址
	proxyServer, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return config, fmt.Errorf("读取代理服务器地址失败: %v", err)
	}

	// 解析代理服务器地址
	// 格式可能是: http=host:port;https=host:port 或 socks=host:port 或 host:port
	if strings.Contains(proxyServer, "=") {
		// 格式: http=host:port;https=host:port
		parts := strings.Split(proxyServer, ";")
		for _, part := range parts {
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					proxyType := strings.ToLower(kv[0])
					address := kv[1]
					if strings.Contains(address, ":") {
						addrParts := strings.Split(address, ":")
						if len(addrParts) == 2 {
							config.Host = addrParts[0]
							config.Port = addrParts[1]
							config.Type = proxyType
							break // 优先使用第一个找到的
						}
					}
				}
			}
		}
	} else if strings.Contains(proxyServer, ":") {
		// 格式: host:port
		parts := strings.Split(proxyServer, ":")
		if len(parts) == 2 {
			config.Host = parts[0]
			config.Port = parts[1]
			config.Type = "http" // 默认类型
		}
	}

	// 读取例外列表
	override, _, err := key.GetStringValue("ProxyOverride")
	if err == nil {
		config.Override = override
	}

	return config, nil
}

// disableSystemProxy 禁用系统代理（Windows）
func disableSystemProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 禁用代理
	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("禁用代理失败: %v", err)
	}

	// 通知系统代理设置已更改
	notifyProxyChange()

	return nil
}

// notifyProxyChange 通知系统代理设置已更改
func notifyProxyChange() {
	// 使用 PowerShell 刷新 .NET 应用程序的代理设置（静默执行，不会弹出窗口）
	// 这个方法只会刷新 .NET 应用程序的代理缓存，不会影响系统代理设置本身
	psCmd := exec.Command("powershell", "-Command", "[System.Net.WebRequest]::DefaultWebProxy = $null; [System.Net.WebRequest]::GetSystemWebProxy() | Out-Null")
	psCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = psCmd.Run()

	// 注意：Windows 系统代理设置通过注册表修改后，大多数应用程序会在下次网络请求时
	// 自动读取新的代理设置，不需要额外的通知。这里只是刷新 .NET 应用程序的缓存。
	// 如果需要立即生效，用户可能需要重启相关应用程序或重新连接网络。
}

// 使用 WinINet API 通知系统（可选，需要 CGO）
func notifyProxyChangeWinINet() {
	// 这里可以使用 golang.org/x/sys/windows 包调用 WinINet API
	// 但为了简化，我们使用命令行的方式
	// 如果需要更可靠的通知，可以使用 CGO 调用 WinINet API
}

// IsSystemProxySupported 检查系统代理是否支持
func IsSystemProxySupported() bool {
	return true
}
