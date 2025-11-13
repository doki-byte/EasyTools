package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"EasyTools/app/model"
	"golang.org/x/net/proxy"
)

type ProxyManager struct {
	ctx context.Context
	mu  sync.RWMutex

	// 全局代理配置
	globalEnabled bool
	globalConfig  *ProxyConfig

	// 客户端缓存
	clientCache *http.Client
}

type ProxyConfig struct {
	Type    string     `json:"type"` // "http", "https", "socks5"
	Host    string     `json:"host"`
	Port    string     `json:"port"`
	Timeout int        `json:"timeout"`
	Auth    *ProxyAuth `json:"auth,omitempty"`
}

type ProxyAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 添加返回结构体
type GlobalProxyResponse struct {
	Config  *ProxyConfig `json:"config"`
	Enabled bool         `json:"enabled"`
}

type ProxyStatus struct {
	GlobalEnabled bool `json:"globalEnabled"`
}

var GlobalProxyManager *ProxyManager

// SetGlobalProxyManager 设置全局代理管理器
func SetGlobalProxyManager(pm *ProxyManager) {
	GlobalProxyManager = pm
}

func NewProxyManager() *ProxyManager {
	pm := &ProxyManager{
		globalEnabled: false,
		globalConfig: &ProxyConfig{
			Type: "http",
			Host: "127.0.0.1",
			Port: "8080",
		},
		clientCache: &http.Client{},
	}

	// 初始化时从数据库加载配置
	pm.loadFromDB()

	return pm
}

func (pm *ProxyManager) SetCtx(ctx context.Context) {
	pm.ctx = ctx
}

// loadFromDB 从数据库加载配置
func (pm *ProxyManager) loadFromDB() {
	db := Db()
	if db == nil {
		fmt.Println("ProxyManager: 数据库连接未初始化")
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 加载代理配置
	var dbConfig model.ProxyConfig
	if err := db.Where("id = ?", 1).First(&dbConfig).Error; err != nil {
		fmt.Printf("ProxyManager: 加载代理配置失败: %v, 使用默认配置\n", err)
		return
	}

	// 设置全局启用状态
	pm.globalEnabled = dbConfig.GlobalEnabled

	// 设置代理配置
	pm.globalConfig = &ProxyConfig{
		Type:    dbConfig.ProxyType,
		Host:    dbConfig.ProxyHost,
		Port:    dbConfig.ProxyPort,
		Timeout: dbConfig.ProxyTimeout,
	}

	// 设置认证信息
	if dbConfig.ProxyUsername != "" {
		pm.globalConfig.Auth = &ProxyAuth{
			Username: dbConfig.ProxyUsername,
			Password: dbConfig.ProxyPassword,
		}
	}

	// 更新客户端
	pm.updateClient()

	fmt.Printf("ProxyManager: 配置加载完成 - 全局启用: %v, 超时: %d秒\n", pm.globalEnabled, pm.globalConfig.Timeout)
}

func (pm *ProxyManager) SetGlobalProxy(config ProxyConfig, enabled bool) (bool, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 验证配置
	if enabled {
		if config.Host == "" || config.Port == "" {
			return false, fmt.Errorf("代理服务器地址和端口不能为空")
		}
	}

	// 更新配置
	pm.globalEnabled = enabled
	pm.globalConfig = &config

	// 更新客户端
	pm.updateClient()

	// 保存到数据库
	if err := pm.saveToDB(); err != nil {
		fmt.Printf("SetGlobalProxy: 保存到数据库失败: %v\n", err)
		return false, fmt.Errorf("保存配置失败: %v", err)
	}

	// 触发事件
	if pm.ctx != nil {
		runtime.EventsEmit(pm.ctx, "global-proxy-updated")
	}
	return true, nil
}

// 修复0值无法保存的问题
func (pm *ProxyManager) saveToDB() error {
	db := Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 构建数据库配置
	dbConfig := model.ProxyConfig{
		Id:             1,
		GlobalEnabled:  pm.globalEnabled,
		ProxyType:      pm.globalConfig.Type,
		ProxyHost:      pm.globalConfig.Host,
		ProxyPort:      pm.globalConfig.Port,
		ProxyTimeout:   pm.globalConfig.Timeout,
		ModuleSettings: "{}",
	}

	if pm.globalConfig.Auth != nil {
		dbConfig.ProxyUsername = pm.globalConfig.Auth.Username
		dbConfig.ProxyPassword = pm.globalConfig.Auth.Password
	} else {
		dbConfig.ProxyUsername = ""
		dbConfig.ProxyPassword = ""
	}

	// 使用 Select 明确指定要更新的字段，包括零值字段
	if err := db.Model(&model.ProxyConfig{}).Where("id = ?", 1).
		Select("global_enabled", "proxy_type", "proxy_host", "proxy_port", "proxy_timeout", "proxy_username", "proxy_password").
		Updates(dbConfig).Error; err != nil {

		// 如果更新失败，尝试创建
		if err := db.Create(&dbConfig).Error; err != nil {
			return fmt.Errorf("保存代理配置失败: %v", err)
		}
	}

	fmt.Printf("saveToDB: 保存成功 - GlobalEnabled: %v\n", dbConfig.GlobalEnabled)
	return nil
}

// 更新连接
func (pm *ProxyManager) updateClient() {
	if pm.globalEnabled && pm.globalConfig != nil {
		pm.clientCache = pm.createProxyClient(*pm.globalConfig, true)
	} else {
		// 直连客户端，使用默认超时10秒
		pm.clientCache = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
}

// 修改 GetProxyStatus 方法，返回结构体而不是 map
func (pm *ProxyManager) GetProxyStatus() ProxyStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return ProxyStatus{
		GlobalEnabled: pm.globalEnabled,
	}
}

// 修改 GetGlobalProxy 方法
func (pm *ProxyManager) GetGlobalProxy() GlobalProxyResponse {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.globalConfig == nil {
		return GlobalProxyResponse{
			Config:  &ProxyConfig{},
			Enabled: pm.globalEnabled,
		}
	}
	return GlobalProxyResponse{
		Config:  pm.globalConfig,
		Enabled: pm.globalEnabled,
	}
}

// TestProxyConnection 测试代理连接
func (pm *ProxyManager) TestProxyConnection(config ProxyConfig) (bool, error) {
	err := pm.testProxyConnection(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (pm *ProxyManager) testProxyConnection(config ProxyConfig) error {
	client := pm.createProxyClient(config, true)

	// 测试URL
	testURL := "http://www.baidu.com"
	if config.Type == "https" {
		testURL = "https://www.baidu.com"
	}

	resp, err := client.Get(testURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("代理测试失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// GetHTTPClient 获取全局HTTP客户端
func (pm *ProxyManager) GetHTTPClient() *http.Client {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.clientCache
}

func (pm *ProxyManager) createProxyClient(config ProxyConfig, enabled bool) *http.Client {
	// 设置超时时间，默认为10秒
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // 默认超时10秒
	}

	if !enabled {
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 设置代理
	proxyURL := pm.buildProxyURL(config)
	if proxyURL != "" {
		switch config.Type {
		case "socks5":
			pm.setSOCKS5Proxy(transport, config)
		default:
			pm.setHTTPProxy(transport, proxyURL)
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func (pm *ProxyManager) buildProxyURL(config ProxyConfig) string {
	var scheme string
	switch config.Type {
	case "http", "https":
		scheme = config.Type
	case "socks5":
		return "" // SOCKS5 特殊处理
	default:
		scheme = "http"
	}

	proxyURL := fmt.Sprintf("%s://%s:%s", scheme, config.Host, config.Port)

	// 添加认证信息
	if config.Auth != nil && config.Auth.Username != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", scheme,
			url.QueryEscape(config.Auth.Username),
			url.QueryEscape(config.Auth.Password),
			config.Host, config.Port)
	}

	return proxyURL
}

func (pm *ProxyManager) setHTTPProxy(transport *http.Transport, proxyURL string) {
	parsedURL, err := url.Parse(proxyURL)
	if err == nil {
		transport.Proxy = http.ProxyURL(parsedURL)
	}
}

func (pm *ProxyManager) setSOCKS5Proxy(transport *http.Transport, config ProxyConfig) {
	socksAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)

	var dialer proxy.Dialer
	var err error

	if config.Auth != nil && config.Auth.Username != "" {
		auth := &proxy.Auth{
			User:     config.Auth.Username,
			Password: config.Auth.Password,
		}
		dialer, err = proxy.SOCKS5("tcp", socksAddr, auth, proxy.Direct)
	} else {
		dialer, err = proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	}

	if err == nil {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}
}
