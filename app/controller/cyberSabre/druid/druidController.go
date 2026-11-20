package druid

import (
	proxySetting "EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	socksproxy "golang.org/x/net/proxy"
)

const druidPathPrefix = "/druid"

// DruidController Druid 利用控制器
type DruidController struct {
	ctx context.Context
	system.Base
}

// NewDruidController 创建 Druid 控制器
func NewDruidController() *DruidController {
	return &DruidController{}
}

// Startup 启动时设置上下文
func (d *DruidController) Startup(ctx context.Context) {
	d.ctx = ctx
}

// 统一日志事件发送
func (d *DruidController) sendLog(level, message string) {
	if d.ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "Druid利用",
	}
	runtime.EventsEmit(d.ctx, "log_event", eventData)
}

// DruidConfig 配置结构
type DruidConfig struct {
	URL      string            `json:"url"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Headers  map[string]string `json:"headers"`
	Proxy    string            `json:"proxy"`
	Timeout  int               `json:"timeout"`
}

// DruidResult Druid 信息结果
type DruidResult struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Sessions  []string  `json:"sessions"`
	SQLs      []string  `json:"sqls"`
	URIs      []string  `json:"uris"`
	JDBCInfo  JDBCInfo  `json:"jdbcInfo"`
	BasicInfo BasicInfo `json:"basicInfo"`
}

// JDBCInfo JDBC 信息
type JDBCInfo struct {
	UserName      string `json:"userName"`
	JDBCURL       string `json:"jdbcUrl"`
	JavaClassPath string `json:"javaClassPath"`
	JavaVMName    string `json:"javaVMName"`
	JavaVersion   string `json:"javaVersion"`
}

// BasicInfo 基础信息
type BasicInfo struct {
	JavaClassPath string `json:"javaClassPath"`
	JavaVMName    string `json:"javaVMName"`
	JavaVersion   string `json:"javaVersion"`
}

// ScanDruid 扫描 Druid 信息
func (d *DruidController) ScanDruid(configJSON string) (*DruidResult, error) {
	var config DruidConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	// 兼容输入站点根路径、/druid 和 /druid/index.html 这几种格式
	baseURL := normalizeDruidBaseURL(config.URL)
	result := &DruidResult{
		Success:  false,
		Sessions: []string{},
		SQLs:     []string{},
		URIs:     []string{},
	}

	// 检查是否需要登录
	var cookie string
	var err error
	if config.Username != "" && config.Password != "" {
		d.sendLog("info", "尝试登录 Druid...")
		cookie, err = d.login(baseURL, config.Username, config.Password, config.Headers, config.Proxy, config.Timeout)
		if err != nil {
			result.Message = fmt.Sprintf("登录失败: %v", err)
			return result, nil
		}
		if cookie == "" {
			result.Message = "登录失败: 未获取到 Cookie"
			return result, nil
		}
		d.sendLog("info", "登录成功")
	}

	// 检查是否可以访问
	d.sendLog("info", "检查 Druid 访问权限...")
	code, err := d.httpGet(baseURL+"/druid/index.html", cookie, config.Headers, config.Proxy, config.Timeout)
	if err != nil {
		result.Message = fmt.Sprintf("访问失败: %v", err)
		return result, nil
	}

	if code == 302 {
		result.Message = "Druid 需要登录，请提供用户名和密码"
		return result, nil
	} else if code != 200 {
		result.Message = fmt.Sprintf("访问失败，状态码: %d", code)
		return result, nil
	}

	d.sendLog("info", "开始获取 Druid 信息...")

	// 获取 Sessions
	d.sendLog("info", "获取 Sessions...")
	sessions, err := d.getDruidJSON(baseURL+"/druid/websession.json", cookie, config.Headers, config.Proxy, config.Timeout, "SESSIONID")
	if err == nil {
		result.Sessions = sessions
		d.sendLog("info", fmt.Sprintf("获取到 %d 个 Session", len(sessions)))
	}

	// 获取 SQLs
	d.sendLog("info", "获取 SQL 信息...")
	sqls, err := d.getDruidJSON(baseURL+"/druid/sql.json", cookie, config.Headers, config.Proxy, config.Timeout, "SQL")
	if err == nil {
		result.SQLs = sqls
		d.sendLog("info", fmt.Sprintf("获取到 %d 条 SQL", len(sqls)))
	}

	// 获取 URIs
	d.sendLog("info", "获取 URI 信息...")
	uris, err := d.getDruidJSON(baseURL+"/druid/weburi.json", cookie, config.Headers, config.Proxy, config.Timeout, "URI")
	if err == nil {
		result.URIs = uris
		d.sendLog("info", fmt.Sprintf("获取到 %d 个 URI", len(uris)))
	}

	// 获取 JDBC 信息
	d.sendLog("info", "获取 JDBC 信息...")
	jdbcInfo, err := d.getJDBCInfo(baseURL, cookie, config.Headers, config.Proxy, config.Timeout)
	if err == nil {
		result.JDBCInfo = jdbcInfo
	}

	// 获取基础信息
	d.sendLog("info", "获取基础信息...")
	basicInfo, err := d.getBasicInfo(baseURL, cookie, config.Headers, config.Proxy, config.Timeout)
	if err != nil {
		d.sendLog("warn", fmt.Sprintf("获取基础信息失败: %v", err))
	} else {
		result.BasicInfo = basicInfo
		if basicInfo.JavaClassPath != "" || basicInfo.JavaVMName != "" || basicInfo.JavaVersion != "" {
			d.sendLog("info", "基础信息获取成功")
		} else {
			d.sendLog("warn", "基础信息为空，可能该 Druid 版本不支持 basic.json 接口")
		}
	}

	result.Success = true
	result.Message = "获取成功"
	d.sendLog("info", "Druid 信息获取完成")

	return result, nil
}

func normalizeDruidBaseURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return strings.TrimRight(rawURL, "/")
	}

	cleanPath := strings.TrimRight(parsedURL.Path, "/")
	lowerPath := strings.ToLower(cleanPath)
	switch {
	case strings.HasSuffix(lowerPath, druidPathPrefix+"/index.html"):
		cleanPath = cleanPath[:len(cleanPath)-len(druidPathPrefix+"/index.html")]
	case strings.HasSuffix(lowerPath, druidPathPrefix):
		cleanPath = cleanPath[:len(cleanPath)-len(druidPathPrefix)]
	}

	parsedURL.Path = cleanPath
	parsedURL.RawPath = ""
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	return strings.TrimRight(parsedURL.String(), "/")
}

// login 登录 Druid
func (d *DruidController) login(baseURL, username, password string, headers map[string]string, proxyStr string, timeout int) (string, error) {
	// 创建允许跟随重定向的客户端用于登录
	client := d.createLoginHTTPClient(proxyStr, timeout)

	// 构建登录参数
	data := url.Values{}
	data.Set("loginUsername", username)
	data.Set("loginPassword", password)

	req, err := http.NewRequest("POST", baseURL+"/druid/submitLogin", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// HTTP 客户端会自动跟随重定向，从最终响应中获取所有 Cookie
	cookies := resp.Cookies()
	var cookieStr strings.Builder
	for i, c := range cookies {
		if i > 0 {
			cookieStr.WriteString("; ")
		}
		cookieStr.WriteString(c.Name + "=" + c.Value)
	}

	return cookieStr.String(), nil
}

// httpGet 发送 GET 请求
func (d *DruidController) httpGet(urlStr, cookie string, headers map[string]string, proxyStr string, timeout int) (int, error) {
	client := d.createHTTPClient(proxyStr, timeout)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return 0, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// getDruidJSON 获取 Druid JSON 数据并提取指定字段
func (d *DruidController) getDruidJSON(urlStr, cookie string, headers map[string]string, proxyStr string, timeout int, fieldName string) ([]string, error) {
	client := d.createHTTPClient(proxyStr, timeout)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, err
	}

	// 提取 Content 数组
	content, ok := jsonData["Content"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	var results []string
	for _, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if value, exists := itemMap[fieldName]; exists {
			if str, ok := value.(string); ok && str != "" {
				// 清理特殊字符
				str = strings.ReplaceAll(str, "\n", " ")
				str = strings.ReplaceAll(str, "\t", " ")
				str = strings.ReplaceAll(str, "\\n", "")
				results = append(results, str)
			}
		}
	}

	return results, nil
}

// getJDBCInfo 获取 JDBC 信息
func (d *DruidController) getJDBCInfo(baseURL, cookie string, headers map[string]string, proxyStr string, timeout int) (JDBCInfo, error) {
	var jdbcInfo JDBCInfo

	// 获取 datasource.json
	client := d.createHTTPClient(proxyStr, timeout)
	req, err := http.NewRequest("GET", baseURL+"/druid/datasource.json", nil)
	if err != nil {
		return jdbcInfo, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return jdbcInfo, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return jdbcInfo, err
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return jdbcInfo, err
	}

	content, ok := jsonData["Content"].([]interface{})
	if ok && len(content) > 0 {
		if itemMap, ok := content[0].(map[string]interface{}); ok {
			if userName, ok := itemMap["UserName"].(string); ok {
				jdbcInfo.UserName = userName
			}
			if jdbcURL, ok := itemMap["URL"].(string); ok {
				jdbcInfo.JDBCURL = jdbcURL
			}
		}
	}

	// 获取 basic.json
	basicInfo, err := d.getBasicInfo(baseURL, cookie, headers, proxyStr, timeout)
	if err == nil {
		jdbcInfo.JavaClassPath = basicInfo.JavaClassPath
		jdbcInfo.JavaVMName = basicInfo.JavaVMName
		jdbcInfo.JavaVersion = basicInfo.JavaVersion
	}

	return jdbcInfo, nil
}

// getBasicInfo 获取基础信息
func (d *DruidController) getBasicInfo(baseURL, cookie string, headers map[string]string, proxyStr string, timeout int) (BasicInfo, error) {
	var basicInfo BasicInfo

	client := d.createHTTPClient(proxyStr, timeout)
	req, err := http.NewRequest("GET", baseURL+"/druid/basic.json", nil)
	if err != nil {
		return basicInfo, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return basicInfo, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != 200 {
		return basicInfo, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return basicInfo, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否为空
	if len(body) == 0 {
		return basicInfo, fmt.Errorf("响应为空")
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return basicInfo, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	// 尝试从 Content 获取（可能是对象或数组）
	var contentObj map[string]interface{}
	if content, ok := jsonData["Content"].(map[string]interface{}); ok {
		// Content 是对象格式：{ "ResultCode": 1, "Content": { "JavaClassPath": "...", ... } }
		contentObj = content
	} else if contentArray, ok := jsonData["Content"].([]interface{}); ok && len(contentArray) > 0 {
		// Content 是数组格式：{ "Content": [{ "JavaClassPath": "...", ... }] }
		if itemMap, ok := contentArray[0].(map[string]interface{}); ok {
			contentObj = itemMap
		}
	} else {
		// 如果 Content 不存在，尝试直接从根对象获取
		contentObj = jsonData
	}

	// 从 contentObj 中提取字段
	if contentObj != nil {
		if javaClassPath, ok := contentObj["JavaClassPath"].(string); ok {
			basicInfo.JavaClassPath = javaClassPath
		}
		if javaVMName, ok := contentObj["JavaVMName"].(string); ok {
			basicInfo.JavaVMName = javaVMName
		}
		if javaVersion, ok := contentObj["JavaVersion"].(string); ok {
			basicInfo.JavaVersion = javaVersion
		}
	}

	return basicInfo, nil
}

// createHTTPClient 创建 HTTP 客户端
func (d *DruidController) createHTTPClient(proxyStr string, timeout int) *http.Client {
	// 设置超时时间
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeoutDuration == 0 {
		timeoutDuration = 10 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 如果指定了代理，使用指定的代理
	if proxyStr != "" {
		proxyURL, err := url.Parse(proxyStr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	} else {
		// 否则使用全局代理配置
		if proxySetting.GlobalProxyManager != nil {
			globalProxyConfig := proxySetting.GlobalProxyManager.GetGlobalProxy()
			if globalProxyConfig.Enabled && globalProxyConfig.Config != nil {
				config := globalProxyConfig.Config
				// 支持 SOCKS5 代理
				if config.Type == "socks5" {
					socksAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)
					var dialer socksproxy.Dialer
					var err error

					if config.Auth != nil && config.Auth.Username != "" {
						auth := &socksproxy.Auth{
							User:     config.Auth.Username,
							Password: config.Auth.Password,
						}
						dialer, err = socksproxy.SOCKS5("tcp", socksAddr, auth, socksproxy.Direct)
					} else {
						dialer, err = socksproxy.SOCKS5("tcp", socksAddr, nil, socksproxy.Direct)
					}

					if err == nil {
						transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
							return dialer.Dial(network, addr)
						}
					}
				} else {
					// HTTP/HTTPS 代理
					proxyURLStr := d.buildProxyURL(*config)
					if proxyURLStr != "" {
						proxyURL, err := url.Parse(proxyURLStr)
						if err == nil {
							transport.Proxy = http.ProxyURL(proxyURL)
						}
					}
				}
			}
		}
	}

	return &http.Client{
		Timeout:   timeoutDuration,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createLoginHTTPClient 创建用于登录的 HTTP 客户端（允许跟随重定向）
func (d *DruidController) createLoginHTTPClient(proxyStr string, timeout int) *http.Client {
	// 设置超时时间
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeoutDuration == 0 {
		timeoutDuration = 10 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 如果指定了代理，使用指定的代理
	if proxyStr != "" {
		proxyURL, err := url.Parse(proxyStr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	} else {
		// 否则使用全局代理配置
		if proxySetting.GlobalProxyManager != nil {
			globalProxyConfig := proxySetting.GlobalProxyManager.GetGlobalProxy()
			if globalProxyConfig.Enabled && globalProxyConfig.Config != nil {
				config := globalProxyConfig.Config
				// 支持 SOCKS5 代理
				if config.Type == "socks5" {
					socksAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)
					var dialer socksproxy.Dialer
					var err error

					if config.Auth != nil && config.Auth.Username != "" {
						auth := &socksproxy.Auth{
							User:     config.Auth.Username,
							Password: config.Auth.Password,
						}
						dialer, err = socksproxy.SOCKS5("tcp", socksAddr, auth, socksproxy.Direct)
					} else {
						dialer, err = socksproxy.SOCKS5("tcp", socksAddr, nil, socksproxy.Direct)
					}

					if err == nil {
						transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
							return dialer.Dial(network, addr)
						}
					}
				} else {
					// HTTP/HTTPS 代理
					proxyURLStr := d.buildProxyURL(*config)
					if proxyURLStr != "" {
						proxyURL, err := url.Parse(proxyURLStr)
						if err == nil {
							transport.Proxy = http.ProxyURL(proxyURL)
						}
					}
				}
			}
		}
	}

	// 登录客户端允许跟随重定向（最多 10 次）
	return &http.Client{
		Timeout:   timeoutDuration,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}
}

// buildProxyURL 构建代理 URL
func (d *DruidController) buildProxyURL(config proxySetting.ProxyConfig) string {
	scheme := config.Type
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}

	proxyURLStr := fmt.Sprintf("%s://%s:%s", scheme, config.Host, config.Port)
	if config.Auth != nil && config.Auth.Username != "" {
		proxyURLStr = fmt.Sprintf("%s://%s:%s@%s:%s", scheme,
			url.QueryEscape(config.Auth.Username),
			url.QueryEscape(config.Auth.Password),
			config.Host, config.Port)
	}

	return proxyURLStr
}
