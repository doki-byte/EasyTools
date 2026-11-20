package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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
	GlobalEnabled   bool `json:"globalEnabled"`
	SystemEnabled   bool `json:"systemEnabled"`   // 系统代理是否启用
	EnabledModules  int  `json:"enabledModules"`  // 已启用代理的模块数
	DisabledModules int  `json:"disabledModules"` // 已禁用代理的模块数
	TotalModules    int  `json:"totalModules"`    // 总模块数
}

var GlobalProxyManager *ProxyManager

// SetGlobalProxyManager 设置全局代理管理器
func SetGlobalProxyManager(pm *ProxyManager) {
	GlobalProxyManager = pm
}

// GetGlobalHTTPClient 安全地获取全局HTTP客户端
func GetGlobalHTTPClient() *http.Client {
	if GlobalProxyManager == nil {
		// 如果 GlobalProxyManager 还未初始化，返回默认客户端
		return &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	return GlobalProxyManager.GetHTTPClient()
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

	return pm
}

// 添加显式初始化方法
func (pm *ProxyManager) Initialize() {
	pm.loadFromDB()
}

func (pm *ProxyManager) SetCtx(ctx context.Context) {
	pm.ctx = ctx
}

func (pm *ProxyManager) loadFromDB() {
	db := Db()
	if db == nil {
		fmt.Println("ProxyManager: 数据库连接未初始化")
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查表是否存在
	tableName := model.TableNameProxyConfig
	hasTable := db.Migrator().HasTable(&model.ProxyConfig{})
	fmt.Printf("ProxyManager: 检查表 %s 是否存在: %v\n", tableName, hasTable)

	if !hasTable {
		fmt.Printf("ProxyManager: 表 %s 不存在，跳过加载\n", tableName)

		// 尝试自动创建表
		if err := db.AutoMigrate(&model.ProxyConfig{}); err != nil {
			fmt.Printf("ProxyManager: 自动创建表失败: %v\n", err)
			return
		}
		fmt.Printf("ProxyManager: 表 %s 创建成功\n", tableName)
	}

	// 加载代理配置
	var dbConfig model.ProxyConfig
	result := db.Where("id = ?", 1).First(&dbConfig)
	if result.Error != nil {
		fmt.Printf("ProxyManager: 加载配置失败: %v\n", result.Error)
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

	// 通知httpx管理器刷新配置
	if globalHTTPXManager != nil {
		globalHTTPXManager.Refresh()
	}

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

	// 保存到数据库（优先执行，避免超时）
	if err := pm.saveToDB(); err != nil {
		fmt.Printf("SetGlobalProxy: 保存到数据库失败: %v\n", err)
		return false, fmt.Errorf("保存配置失败: %v", err)
	}

	// 异步刷新httpx管理器配置，避免阻塞主流程
	go func() {
		if globalHTTPXManager != nil {
			globalHTTPXManager.Refresh()
		}
	}()

	// 触发事件
	if pm.ctx != nil {
		runtime.EventsEmit(pm.ctx, "global-proxy-updated")
	}
	return true, nil
}

// SetSystemProxy 设置系统代理
func (pm *ProxyManager) SetSystemProxy(config ProxyConfig, enabled bool) (bool, error) {
	// 验证配置
	if enabled {
		if config.Host == "" || config.Port == "" {
			return false, fmt.Errorf("代理服务器地址和端口不能为空")
		}
	}

	// 转换为系统代理配置
	systemConfig := SystemProxyConfig{
		Enabled: enabled,
		Host:    config.Host,
		Port:    config.Port,
		Type:    config.Type,
	}

	// 设置系统代理
	if err := SetSystemProxy(systemConfig); err != nil {
		return false, fmt.Errorf("设置系统代理失败: %v", err)
	}

	// 触发事件，通知前端更新状态
	if pm.ctx != nil {
		runtime.EventsEmit(pm.ctx, "global-proxy-updated")
	}

	return true, nil
}

// GetSystemProxy 获取当前系统代理设置
func (pm *ProxyManager) GetSystemProxy() (SystemProxyConfig, error) {
	return GetSystemProxy()
}

// IsSystemProxySupported 检查系统代理是否支持
func (pm *ProxyManager) IsSystemProxySupported() bool {
	return IsSystemProxySupported()
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
	result := db.Model(&model.ProxyConfig{}).Where("id = ?", 1).
		Select("global_enabled", "proxy_type", "proxy_host", "proxy_port", "proxy_timeout", "proxy_username", "proxy_password").
		Updates(dbConfig)

	// 如果更新失败或没有影响任何行（记录不存在），尝试创建
	if result.Error != nil || result.RowsAffected == 0 {
		// 如果更新失败，尝试创建
		if err := db.Create(&dbConfig).Error; err != nil {
			return fmt.Errorf("保存代理配置失败: %v", err)
		}
		fmt.Printf("saveToDB: 创建新记录成功 - GlobalEnabled: %v\n", dbConfig.GlobalEnabled)
	} else {
		fmt.Printf("saveToDB: 更新记录成功 - GlobalEnabled: %v\n", dbConfig.GlobalEnabled)
	}

	return nil
}

// 更新连接
func (pm *ProxyManager) updateClient() {
	if pm.globalEnabled && pm.globalConfig != nil {
		pm.clientCache = pm.createProxyClient(*pm.globalConfig, true)
	} else {
		// 直连客户端，优化连接池配置以提高性能
		pm.clientCache = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				MaxIdleConns:        500, // 增加最大空闲连接数
				MaxIdleConnsPerHost: 100, // 增加每个主机的连接数
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false, // 启用连接复用
			},
		}
	}
}

// 修改 GetProxyStatus 方法，返回结构体而不是 MapCheck
func (pm *ProxyManager) GetProxyStatus() ProxyStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 获取系统代理状态
	systemEnabled := false
	if systemConfig, err := GetSystemProxy(); err == nil {
		systemEnabled = systemConfig.Enabled
	}

	return ProxyStatus{
		GlobalEnabled:   pm.globalEnabled,
		SystemEnabled:   systemEnabled,
		EnabledModules:  0, // 这些字段暂时保留，后续可以扩展
		DisabledModules: 0,
		TotalModules:    0,
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
	// 对于 SOCKS5 代理，先验证配置是否有效
	if config.Type == "socks5" {
		socksAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)
		timeout := time.Duration(config.Timeout) * time.Second
		if timeout == 0 {
			timeout = 10 * time.Second
		}
		baseDialer := &net.Dialer{Timeout: timeout}
		var dialer proxy.Dialer
		var err error

		if config.Auth != nil && config.Auth.Username != "" {
			auth := &proxy.Auth{
				User:     config.Auth.Username,
				Password: config.Auth.Password,
			}
			dialer, err = proxy.SOCKS5("tcp", socksAddr, auth, baseDialer)
		} else {
			dialer, err = proxy.SOCKS5("tcp", socksAddr, nil, baseDialer)
		}

		if err != nil {
			return fmt.Errorf("创建SOCKS5代理失败: %v", err)
		}

		// 测试 SOCKS5 连接，使用超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var testConn net.Conn
		if ctxDialer, ok := dialer.(proxy.ContextDialer); ok {
			testConn, err = ctxDialer.DialContext(ctx, "tcp", "www.baidu.com:80")
		} else {
			testConn, err = dialer.Dial("tcp", "www.baidu.com:80")
		}
		if err != nil {
			return fmt.Errorf("SOCKS5代理连接测试失败: %v", err)
		}
		testConn.Close()
		return nil
	}

	// HTTP/HTTPS 代理测试
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
	if pm == nil {
		// 如果 ProxyManager 为 nil，返回默认客户端
		return &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.clientCache == nil {
		// 如果 clientCache 为 nil，初始化一个默认客户端
		pm.clientCache = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

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
		MaxIdleConns:        500, // 优化：增加连接池大小
		MaxIdleConnsPerHost: 100, // 优化：增加每个主机的连接数
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false, // 启用连接复用
	}

	// 设置代理
	proxyURL := pm.buildProxyURL(config)
	if config.Type == "socks5" {
		// SOCKS5 特殊处理，buildProxyURL 返回空字符串
		if err := pm.setSOCKS5Proxy(transport, config); err != nil {
			// 如果设置失败，返回一个会失败的客户端（transport 没有设置 DialContext）
			// 注意：这会导致连接失败，但测试函数会先验证 SOCKS5 连接
			return &http.Client{
				Timeout:   timeout,
				Transport: transport,
			}
		}
	} else if proxyURL != "" {
		pm.setHTTPProxy(transport, proxyURL)
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

func (pm *ProxyManager) setSOCKS5Proxy(transport *http.Transport, config ProxyConfig) error {
	dialer, err := pm.createSOCKS5Dialer(config)
	if err != nil {
		return err
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}
	return nil
}

// createSOCKS5Dialer 创建SOCKS5 dialer（用于TCP连接）
func (pm *ProxyManager) createSOCKS5Dialer(config ProxyConfig) (proxy.Dialer, error) {
	socksAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	baseDialer := &net.Dialer{Timeout: timeout}

	var dialer proxy.Dialer
	var err error

	if config.Auth != nil && config.Auth.Username != "" {
		auth := &proxy.Auth{
			User:     config.Auth.Username,
			Password: config.Auth.Password,
		}
		dialer, err = proxy.SOCKS5("tcp", socksAddr, auth, baseDialer)
	} else {
		dialer, err = proxy.SOCKS5("tcp", socksAddr, nil, baseDialer)
	}

	if err != nil {
		return nil, fmt.Errorf("创建SOCKS5代理失败: %v", err)
	}

	return dialer, nil
}

// GetProxyDialer 获取代理dialer（支持SOCKS5、HTTP、HTTPS）
// 返回nil表示不使用代理
func (pm *ProxyManager) GetProxyDialer() (proxy.Dialer, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.globalEnabled || pm.globalConfig == nil {
		return nil, nil // 代理未启用
	}

	switch pm.globalConfig.Type {
	case "socks5":
		return pm.createSOCKS5Dialer(*pm.globalConfig)
	case "http", "https":
		return pm.createHTTPProxyDialer(*pm.globalConfig)
	default:
		return nil, nil // 不支持的代理类型
	}
}

// GetSOCKS5Dialer 获取SOCKS5 dialer（如果代理启用且为SOCKS5类型）
// 返回nil表示不使用代理或不是SOCKS5代理
// 保留此方法以保持向后兼容
func (pm *ProxyManager) GetSOCKS5Dialer() (proxy.Dialer, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.globalEnabled || pm.globalConfig == nil {
		return nil, nil // 代理未启用
	}

	if pm.globalConfig.Type != "socks5" {
		return nil, nil // 不是SOCKS5代理
	}

	return pm.createSOCKS5Dialer(*pm.globalConfig)
}

// createHTTPProxyDialer 创建HTTP/HTTPS代理dialer（使用CONNECT方法建立TCP隧道）
func (pm *ProxyManager) createHTTPProxyDialer(config ProxyConfig) (proxy.Dialer, error) {
	// 构建代理URL
	proxyURL := pm.buildProxyURL(config)
	if proxyURL == "" {
		return nil, fmt.Errorf("代理URL为空")
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("解析代理URL失败: %v", err)
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// 创建HTTP代理dialer
	// HTTP代理使用CONNECT方法建立TCP隧道
	dialer := &httpProxyDialer{
		proxyURL: parsedURL,
		timeout:  timeout,
	}

	return dialer, nil
}

// httpProxyDialer HTTP/HTTPS代理dialer实现
type httpProxyDialer struct {
	proxyURL *url.URL
	timeout  time.Duration
}

// Dial 通过HTTP代理建立TCP连接（使用CONNECT方法）
func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	// 先连接到代理服务器
	proxyAddr := d.proxyURL.Host
	if d.proxyURL.Port() == "" {
		if d.proxyURL.Scheme == "https" {
			proxyAddr += ":443"
		} else {
			proxyAddr += ":80"
		}
	}

	conn, err := net.DialTimeout("tcp", proxyAddr, d.timeout)
	if err != nil {
		return nil, fmt.Errorf("连接代理服务器失败: %v", err)
	}

	// 发送CONNECT请求
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n", addr)
	connectReq += fmt.Sprintf("Host: %s\r\n", addr)

	// 如果有认证信息，添加Proxy-Authorization头
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}

	connectReq += "\r\n"

	// 如果是HTTPS代理，需要先建立TLS连接
	if d.proxyURL.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         d.proxyURL.Hostname(),
			InsecureSkipVerify: true,
		})
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("TLS握手失败: %v", err)
		}
		conn = tlsConn
	}

	// 发送CONNECT请求
	if _, err := conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("发送CONNECT请求失败: %v", err)
	}

	// 读取响应
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取代理响应失败: %v", err)
	}

	// 检查响应状态码（应该是200 Connection established）
	if !strings.HasPrefix(response, "HTTP/1.1 200") && !strings.HasPrefix(response, "HTTP/1.0 200") {
		// 读取完整的响应头
		for {
			line, err := reader.ReadString('\n')
			if err != nil || line == "\r\n" || line == "\n" {
				break
			}
		}
		conn.Close()
		return nil, fmt.Errorf("代理CONNECT请求失败: %s", strings.TrimSpace(response))
	}

	// 读取剩余的响应头（直到空行）
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" || line == "\n" {
			break
		}
	}

	return conn, nil
}
