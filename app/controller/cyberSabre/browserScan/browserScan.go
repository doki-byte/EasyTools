package browserScan

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/projectdiscovery/httpx/common/httpx"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BrowserScan 浏览器扫描器（渗透测试模块）
type BrowserScan struct {
	system.Base
	tasks      map[string]*ScanTask
	tasksMutex sync.RWMutex
	configPath string
	httpxMgr   *proxy.HTTPXManager // httpx管理器，用于优化HTTP请求
}

// ScanTask 扫描任务
type ScanTask struct {
	ID         string          `json:"id"`
	URL        string          `json:"url"`
	Status     string          `json:"status"` // running, completed, stopped, error
	StartTime  time.Time       `json:"start_time"`
	EndTime    *time.Time      `json:"end_time,omitempty"`
	Results    *ExtractResults `json:"results"`
	Logs       []LogEntry      `json:"logs"`
	Config     *ScanConfig     `json:"config"`
	PacketLogs []PacketLog     `json:"packet_logs"` // 数据包日志
	browser    *rod.Browser
	page       *rod.Page
	cancelFunc context.CancelFunc
}

// PacketLog 数据包日志
type PacketLog struct {
	URL            string              `json:"url"`
	Method         string              `json:"method"`
	StatusCode     int                 `json:"status_code"`
	Duration       string              `json:"duration"`
	RequestPacket  string              `json:"request_packet"`
	ResponsePacket string              `json:"response_packet"`
	ExtractedInfo  []ExtractedInfoItem `json:"extracted_info"`
}

// ExtractedInfoItem 提取的信息项
type ExtractedInfoItem struct {
	Rule  string `json:"rule"`
	Value string `json:"value"`
}

// ScanConfig 扫描配置
type ScanConfig struct {
	WaitTime        int      `json:"wait_time"`         // 等待JS加载时间（秒）
	ExtractRules    []string `json:"extract_rules"`     // 提取规则列表
	ApiRoutes       []string `json:"api_routes"`        // API路由列表
	GlobalApiRoutes []string `json:"global_api_routes"` // 全局API路由
	AllowList       []string `json:"allow_list"`        // 白名单域名
	AutoTestApi     bool     `json:"auto_test_api"`     // 自动对接口进行发包测试
	TestMethods     []string `json:"test_methods"`      // HTTP方法列表，默认GET
	WorkerNum       int      `json:"worker_num"`        // 并发线程数，默认10
}

// ExtractResults 提取结果
type ExtractResults struct {
	IPs             []string `json:"ips"`
	IPPorts         []string `json:"ip_ports"`
	Domains         []string `json:"domains"`
	Paths           []string `json:"paths"`
	IncompletePaths []string `json:"incomplete_paths"`
	URLs            []string `json:"urls"`
	StaticFiles     []string `json:"static_files"`
	JSFiles         []string `json:"js_files"`
	APIEndpoints    []string `json:"api_endpoints"`
	JWT             []string `json:"jwt"`
	Secrets         []string `json:"secrets"` // 保留用于兼容性
	Mobiles         []string `json:"mobiles"`
	Emails          []string `json:"emails"`
	SFZ             []string `json:"sfz"`
	Algorithms      []string `json:"algorithms"`
	// 敏感信息分类字段
	SecurityKeys []string `json:"security_keys"`
	Passwords    []string `json:"passwords"`
	Usernames    []string `json:"usernames"`
	AKSKs        []string `json:"aksk"`
	AppIDs       []string `json:"app_ids"`
	AccessKeys   []string `json:"access_keys"`
	APIKeys      []string `json:"api_keys"`
	SecretKeys   []string `json:"secret_keys"`
	IDCards      []string `json:"id_cards"`
	// VueAnalysis 页面内 Vue / Vue Router 分析（参考 VueCrack，由 Rod 注入 detector 采集）
	VueAnalysis *VueAnalysisResult `json:"vue_analysis,omitempty"`
}

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"` // info, warning, error
	Message string `json:"message"`
}

// ExtractRulesConfig 提取规则配置
type ExtractRulesConfig struct {
	Rules map[string]string `json:"rules"`
}

// NewBrowserScan 创建新的浏览器扫描器
func NewBrowserScan() *BrowserScan {
	bs := &BrowserScan{
		tasks:      make(map[string]*ScanTask),
		configPath: filepath.Join("EasyToolsFiles", "config", "browser_scan_config.json"),
		httpxMgr:   proxy.GetHTTPXManager(), // 初始化httpx管理器
	}
	return bs
}

// Startup 初始化
func (bs *BrowserScan) Startup(ctx context.Context) {
	bs.SetCtx(ctx)
	bs.initConfig()
}

// initConfig 初始化配置文件
func (bs *BrowserScan) initConfig() {
	configDir := filepath.Dir(bs.configPath)
	_ = os.MkdirAll(configDir, 0755)

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(bs.configPath); os.IsNotExist(err) {
		defaultConfig := ExtractRulesConfig{
			Rules: bs.getDefaultRules(),
		}
		_ = bs.saveConfig(defaultConfig)
	}
}

// getDefaultRules 获取默认提取规则（参考FindSomething）
func (bs *BrowserScan) getDefaultRules() map[string]string {
	return map[string]string{
		"ip":              `['"](([a-zA-Z0-9]+:)?\/\/)?\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(\/.*?)?['"]`,
		"ip_port":         `['"](([a-zA-Z0-9]+:)?\/\/)?\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\:\d{1,5}(\/.*?)?['"]`,
		"domain":          `['"](([a-zA-Z0-9]+:)?\/\/)?[a-zA-Z0-9\-\.]*?\.(xin|com|cn|net|com\.cn|vip|top|cc|shop|club|wang|xyz|luxe|site|news|pub|fun|online|win|red|loan|ren|mom|net\.cn|org|link|biz|bid|help|tech|date|mobi|so|me|tv|co|vc|pw|video|party|pics|website|store|ltd|ink|trade|live|wiki|space|gift|lol|work|band|info|click|photo|market|tel|social|press|game|kim|org\.cn|games|pro|men|love|studio|rocks|asia|group|science|design|software|engineer|lawyer|fit|beer|tw|我爱你|中国|公司|网络|在线|网址|网店|集团|中文网)(\:\d{1,5})?(\/)?['"]`,
		"path":            `['"](?:\/|\.\.\/|\.\/)[^\/\>\< \)\(\{\}\,\'\"\\]([^\>\< \)\(\{\}\,\'\"\\])*?['"]`,
		"incomplete_path": `['"][^\/\>\< \)\(\{\}\,\'\"\\][\w\/]*?\/[\w\/]*?['"]`,
		"url":             `['"](([a-zA-Z0-9]+:)?\/\/)?[a-zA-Z0-9\-\.]*?\.(xin|com|cn|net|com\.cn|vip|top|cc|shop|club|wang|xyz|luxe|site|news|pub|fun|online|win|red|loan|ren|mom|net\.cn|org|link|biz|bid|help|tech|date|mobi|so|me|tv|co|vc|pw|video|party|pics|website|store|ltd|ink|trade|live|wiki|space|gift|lol|work|band|info|click|photo|market|tel|social|press|game|kim|org\.cn|games|pro|men|love|studio|rocks|asia|group|science|design|software|engineer|lawyer|fit|beer|tw|我爱你|中国|公司|网络|在线|网址|网店|集团|中文网)(\:\d{1,5})?(\/.*?)?['"]`,
		"jwt":             `['"](ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9._-]{10,}|ey[A-Za-z0-9_\/+-]{10,}\.[A-Za-z0-9._\/+-]{10,})['"]`,
		"mobile":          `['"](1(3([0-35-9]\d|4[1-8])|4[14-9]\d|5([\d]\d|7[1-79])|66\d|7[2-35-8]\d|8\d{2}|9[89]\d)\d{7})['"]`,
		"email":           `['"][a-zA-Z0-9\._\-]*@[a-zA-Z0-9\._\-]{1,63}\.((?!js|css|jpg|jpeg|png|ico)[a-zA-Z]{2,})['"]`,
		"sfz":             `['"]((\d{8}(0\d|10|11|12)([0-2]\d|30|31)\d{3}$)|(\d{6}(18|19|20)\d{2}(0[1-9]|10|11|12)([0-2]\d|30|31)\d{3}(\d|X|x)))['"]`,
		"algorithm":       `\W(Base64\.encode|Base64\.decode|btoa|atob|CryptoJS\.AES|CryptoJS\.DES|JSEncrypt|rsa|KJUR|$\.md5|md5|sha1|sha256|sha512)[\(\.]`,
		// 敏感信息提取规则
		"securitykey":  `(?i)(?:['"]?)(?:security[_-]?key|securitykey)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"security_key": `(?i)(?:['"]?)(?:security[_-]?key)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"password":     `(?i)(?:['"]?)(?:password|pwd|passwd|pass)[^\S\r\n]*[=:][^\S\r\n]*['"]?([^'"\s]{6,})['"]?`,
		"username":     `(?i)(?:['"]?)(?:username|user[_-]?name|user|login)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-\.@]{3,})['"]?`,
		"aksk":         `(?i)(?:['"]?)(?:(?:access[_-]?key[_-]?id|ak)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9]{16,})['"]?|(?:access[_-]?key[_-]?secret|sk|secret[_-]?key)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9]{16,})['"]?)`,
		"appid":        `(?i)(?:['"]?)(?:app[_-]?id|appid|application[_-]?id)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{8,})['"]?`,
		"accesskey":    `(?i)(?:['"]?)(?:access[_-]?key|accesskey|access[_-]?token)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"access_key":   `(?i)(?:['"]?)(?:access[_-]?key)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"api_key":      `(?i)(?:['"]?)(?:api[_-]?key|apikey)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"secret_key":   `(?i)(?:['"]?)(?:secret[_-]?key|secretkey|secret)[^\S\r\n]*[=:][^\S\r\n]*['"]?([A-Za-z0-9_\-]{16,})['"]?`,
		"id_card":      `['"]?([1-9]\d{5}(?:18|19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]|[1-9]\d{7}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3})['"]?`,
	}
}

// saveConfig 保存配置
func (bs *BrowserScan) saveConfig(config ExtractRulesConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(bs.configPath, data, 0644)
}

// loadConfig 加载配置
func (bs *BrowserScan) loadConfig() (*ExtractRulesConfig, error) {
	data, err := ioutil.ReadFile(bs.configPath)
	if err != nil {
		return nil, err
	}
	var config ExtractRulesConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}

// CreateTask 创建扫描任务
func (bs *BrowserScan) CreateTask(taskURL string, config *ScanConfig) (string, error) {
	// 如果未提供配置，创建默认配置（默认不启用自动接口发包）
	if config == nil {
		config = &ScanConfig{
			WaitTime:        5,
			ApiRoutes:       []string{},
			GlobalApiRoutes: []string{"/api", "/api/v1", "/api/v2"},
			AllowList:       []string{},
			WorkerNum:       10, // 默认10个线程
		}
	}

	_ = config.AutoTestApi

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	task := &ScanTask{
		ID:         taskID,
		URL:        taskURL,
		Status:     "running",
		StartTime:  time.Now(),
		Results:    &ExtractResults{},
		Logs:       []LogEntry{},
		Config:     config,
		PacketLogs: []PacketLog{},
	}

	bs.tasksMutex.Lock()
	bs.tasks[taskID] = task
	bs.tasksMutex.Unlock()

	bs.addLog(taskID, "info", fmt.Sprintf("创建任务: %s", taskURL))

	// 异步执行扫描
	go bs.runScan(task)

	return taskID, nil
}

// runScan 执行扫描
func (bs *BrowserScan) runScan(task *ScanTask) {
	defer func() {
		if r := recover(); r != nil {
			bs.addLog(task.ID, "error", fmt.Sprintf("扫描异常: %v", r))
			task.Status = "error"
		}
		now := time.Now()
		task.EndTime = &now

		// 任务完成或出错时，自动保存任务数据
		go func() {
			_ = bs.SaveTaskLog(task.ID)
		}()
	}()

	_, cancel := context.WithCancel(context.Background())
	task.cancelFunc = cancel
	defer cancel()

	// 初始化浏览器
	browser, err := bs.initBrowser(task.Config)
	if err != nil {
		bs.addLog(task.ID, "error", fmt.Sprintf("初始化浏览器失败: %v", err))
		task.Status = "error"
		return
	}
	defer browser.MustClose()
	task.browser = browser

	// 创建页面
	page := browser.MustPage()
	defer page.MustClose()
	task.page = page

	// 导航到目标URL
	bs.addLog(task.ID, "info", fmt.Sprintf("访问URL: %s", task.URL))
	page.MustNavigate(task.URL)

	// 等待页面加载
	waitTime := time.Duration(task.Config.WaitTime) * time.Second
	if waitTime == 0 {
		waitTime = 5 * time.Second
	}
	page.MustWaitLoad()
	time.Sleep(waitTime)

	// 提取信息
	bs.addLog(task.ID, "info", "开始提取页面信息...")
	html, err := page.HTML()
	if err != nil {
		bs.addLog(task.ID, "error", fmt.Sprintf("获取HTML失败: %v", err))
		task.Status = "error"
		return
	}

	// 提取JS文件内容（包含外链JS，会自动下载）
	jsContents := bs.extractJSFiles(page, task.URL)

	// 合并所有内容进行提取
	allContent := html + "\n" + strings.Join(jsContents, "\n")

	// 执行提取
	results := bs.extractInfo(allContent, task.URL)
	task.Results = results

	// Vue / Vue Router 分析（与 VueCrack 同源逻辑，在真实页面上下文中执行）
	vueRes := bs.runVueCrackAnalysis(page)
	if vueRes != nil {
		task.Results.VueAnalysis = vueRes
		if vueRes.Error != "" {
			bs.addLog(task.ID, "warning", "Vue 路由分析: "+vueRes.Error)
		} else if vueRes.RouterDetected {
			bs.addLog(task.ID, "info", fmt.Sprintf("Vue Router: 版本=%s, 路由数=%d, 已改 auth 路由数=%d",
				vueRes.VueVersion, len(vueRes.AllRoutes), len(vueRes.ModifiedRoutes)))
		} else if vueRes.VueDetected {
			bs.addLog(task.ID, "info", fmt.Sprintf("检测到 Vue（%s），未识别到 Vue Router", vueRes.VueVersion))
		} else {
			bs.addLog(task.ID, "info", "未检测到 Vue 应用（或页面为静态/非 SPA）")
		}
	}

	// 生成API端点
	bs.generateAPIEndpoints(task)

	bs.addLog(task.ID, "info", fmt.Sprintf("提取完成: URL=%d, IP=%d, Domain=%d, API=%d",
		len(results.URLs), len(results.IPs), len(results.Domains), len(results.APIEndpoints)))

	// 如果启用了自动发包测试，则对API端点进行测试
	if task.Config.AutoTestApi {
		bs.testApiEndpoints(task)
	}

	task.Status = "completed"
}

// initBrowser 初始化浏览器
func (bs *BrowserScan) initBrowser(config *ScanConfig) (*rod.Browser, error) {
	launcher := launcher.New().
		Headless(true).
		NoSandbox(true)

	// 自动使用全局代理配置
	if proxy.GlobalProxyManager != nil {
		pm := proxy.GlobalProxyManager
		globalProxy := pm.GetGlobalProxy()
		if globalProxy.Enabled && globalProxy.Config != nil {
			// 构建代理URL
			proxyType := globalProxy.Config.Type
			if proxyType == "" {
				proxyType = "http"
			}
			proxyURL := fmt.Sprintf("%s://%s:%s", proxyType, globalProxy.Config.Host, globalProxy.Config.Port)
			if globalProxy.Config.Auth != nil {
				proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", proxyType,
					globalProxy.Config.Auth.Username, globalProxy.Config.Auth.Password,
					globalProxy.Config.Host, globalProxy.Config.Port)
			}
			if proxyURL != "" {
				launcher = launcher.Proxy(proxyURL)
			}
		}
	}

	browserURL, err := launcher.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(browserURL)
	if err := browser.Connect(); err != nil {
		return nil, err
	}

	return browser, nil
}

// extractJSFiles 提取JS文件内容
func (bs *BrowserScan) extractJSFiles(page *rod.Page, baseURL string) []string {
	var jsContents []string

	// 获取所有script标签
	scripts, err := page.Elements("script")
	if err != nil {
		return jsContents
	}

	for _, script := range scripts {
		src, err := script.Attribute("src")
		if err == nil && src != nil && *src != "" {
			// 外部JS文件，尝试获取内容
			jsContent := bs.fetchJSContent(baseURL, *src)
			if jsContent != "" {
				jsContents = append(jsContents, jsContent)
			}
		} else {
			// 内联JS
			text, err := script.Text()
			if err == nil {
				jsContents = append(jsContents, text)
			}
		}
	}

	return jsContents
}

// fetchJSContent 获取JS文件内容（使用httpx优化）
func (bs *BrowserScan) fetchJSContent(baseURL string, jsURL string) string {
	fullURL := bs.resolveURL(baseURL, jsURL)
	if fullURL == "" {
		return ""
	}

	// 优先使用httpx客户端（连接复用、自动重试）
	if bs.httpxMgr != nil {
		httpxClient := bs.httpxMgr.GetClient()
		if httpxClient != nil {
			req, err := httpxClient.NewRequest("GET", fullURL)
			if err == nil {
				req.Request.Header.Set("User-Agent", "EasyTools-BrowserScan/1.0")
				req.Request.Header.Set("Accept", "*/*")

				resp, err := httpxClient.Do(req, httpx.UnsafeOptions{})
				if err == nil && resp != nil {
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						// httpx.Response使用Data字段（decoded data）或RawData字段（undecoded data）
						bodyData := resp.Data
						if len(bodyData) == 0 {
							bodyData = resp.RawData
						}
						// 限制大小，避免大文件撑爆内存（1MB）
						const maxSize = 1 << 20
						if len(bodyData) > maxSize {
							return string(bodyData[:maxSize])
						}
						return string(bodyData)
					}
				}
			}
		}
	}

	// Fallback: 使用标准http.Client（兼容性保证）
	cli := (&http.Client{Timeout: 10 * time.Second})
	if proxy.GlobalProxyManager != nil {
		cli = proxy.GlobalProxyManager.GetHTTPClient()
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "EasyTools-BrowserScan/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := cli.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}

	// 限制大小，避免大文件撑爆内存（1MB）
	const maxSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return ""
	}
	return string(body)
}

func (bs *BrowserScan) resolveURL(baseURL string, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}

	// 处理 scheme-relative：//cdn.xxx.com/a.js
	if strings.HasPrefix(ref, "//") {
		u, err := url.Parse(baseURL)
		if err != nil || u.Scheme == "" {
			return "https:" + ref
		}
		return u.Scheme + ":" + ref
	}

	ru, err := url.Parse(ref)
	if err == nil && ru.IsAbs() {
		return ru.String()
	}

	bu, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	// 相对路径解析
	return bu.ResolveReference(&url.URL{Path: ref, RawQuery: ""}).String()
}

// extractInfo 提取信息
func (bs *BrowserScan) extractInfo(content, baseURL string) *ExtractResults {
	results := &ExtractResults{
		IPs:             []string{},
		IPPorts:         []string{},
		Domains:         []string{},
		Paths:           []string{},
		IncompletePaths: []string{},
		URLs:            []string{},
		StaticFiles:     []string{},
		JSFiles:         []string{},
		APIEndpoints:    []string{},
		JWT:             []string{},
		Secrets:         []string{},
		Mobiles:         []string{},
		Emails:          []string{},
		SFZ:             []string{},
		Algorithms:      []string{},
		// 敏感信息分类字段
		SecurityKeys: []string{},
		Passwords:    []string{},
		Usernames:    []string{},
		AKSKs:        []string{},
		AppIDs:       []string{},
		AccessKeys:   []string{},
		APIKeys:      []string{},
		SecretKeys:   []string{},
		IDCards:      []string{},
	}

	// 加载提取规则
	config, err := bs.loadConfig()
	if err != nil {
		config = &ExtractRulesConfig{Rules: bs.getDefaultRules()}
	}

	// 提取各种信息
	results.IPs = bs.extractByRegex(content, config.Rules["ip"])
	results.IPPorts = bs.extractByRegex(content, config.Rules["ip_port"])
	results.Domains = bs.extractByRegex(content, config.Rules["domain"])
	results.Paths = bs.extractByRegex(content, config.Rules["path"])
	results.IncompletePaths = bs.extractByRegex(content, config.Rules["incomplete_path"])
	results.URLs = bs.extractByRegex(content, config.Rules["url"])
	results.JWT = bs.extractByRegex(content, config.Rules["jwt"])
	results.Mobiles = bs.extractByRegex(content, config.Rules["mobile"])
	results.Emails = bs.extractByRegex(content, config.Rules["email"])
	results.SFZ = bs.extractByRegex(content, config.Rules["sfz"])
	results.Algorithms = bs.extractByRegex(content, config.Rules["algorithm"])

	// 提取敏感信息（分类存储）
	if rule, exists := config.Rules["securitykey"]; exists {
		results.SecurityKeys = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["security_key"]; exists {
		results.SecurityKeys = append(results.SecurityKeys, bs.extractByRegex(content, rule)...)
	}
	if rule, exists := config.Rules["password"]; exists {
		results.Passwords = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["username"]; exists {
		results.Usernames = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["aksk"]; exists {
		results.AKSKs = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["appid"]; exists {
		results.AppIDs = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["accesskey"]; exists {
		results.AccessKeys = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["access_key"]; exists {
		results.AccessKeys = append(results.AccessKeys, bs.extractByRegex(content, rule)...)
	}
	if rule, exists := config.Rules["api_key"]; exists {
		results.APIKeys = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["secret_key"]; exists {
		results.SecretKeys = bs.extractByRegex(content, rule)
	}
	if rule, exists := config.Rules["id_card"]; exists {
		results.IDCards = bs.extractByRegex(content, rule)
	}

	// 去重敏感信息
	results.SecurityKeys = bs.unique(results.SecurityKeys)
	results.Passwords = bs.unique(results.Passwords)
	results.Usernames = bs.unique(results.Usernames)
	results.AKSKs = bs.unique(results.AKSKs)
	results.AppIDs = bs.unique(results.AppIDs)
	results.AccessKeys = bs.unique(results.AccessKeys)
	results.APIKeys = bs.unique(results.APIKeys)
	results.SecretKeys = bs.unique(results.SecretKeys)
	results.IDCards = bs.unique(results.IDCards)

	// 合并所有敏感信息到Secrets字段（用于兼容性）
	allSecrets := []string{}
	allSecrets = append(allSecrets, results.SecurityKeys...)
	allSecrets = append(allSecrets, results.Passwords...)
	allSecrets = append(allSecrets, results.Usernames...)
	allSecrets = append(allSecrets, results.AKSKs...)
	allSecrets = append(allSecrets, results.AppIDs...)
	allSecrets = append(allSecrets, results.AccessKeys...)
	allSecrets = append(allSecrets, results.APIKeys...)
	allSecrets = append(allSecrets, results.SecretKeys...)
	allSecrets = append(allSecrets, results.IDCards...)
	results.Secrets = bs.unique(allSecrets)

	// 提取静态文件和JS文件
	results.StaticFiles = bs.extractStaticFiles(content)
	results.JSFiles = bs.extractJSFilesFromContent(content)

	// 去重
	results.IPs = bs.unique(results.IPs)
	results.IPPorts = bs.unique(results.IPPorts)
	results.Domains = bs.unique(results.Domains)
	results.Paths = bs.unique(results.Paths)
	results.IncompletePaths = bs.unique(results.IncompletePaths)
	results.URLs = bs.unique(results.URLs)
	results.JSFiles = bs.unique(results.JSFiles)
	results.StaticFiles = bs.unique(results.StaticFiles)

	return results
}

// extractByRegex 使用正则表达式提取
func (bs *BrowserScan) extractByRegex(content, pattern string) []string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return []string{}
	}

	matches := re.FindAllString(content, -1)
	results := []string{}
	for _, match := range matches {
		// 清理引号
		cleaned := strings.Trim(match, `'"`)
		if cleaned != "" {
			results = append(results, cleaned)
		}
	}
	return results
}

// extractStaticFiles 提取静态文件
func (bs *BrowserScan) extractStaticFiles(content string) []string {
	staticExts := []string{".jpg", ".jpeg", ".png", ".gif", ".css", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	var files []string

	for _, ext := range staticExts {
		pattern := fmt.Sprintf(`['"]([^'"]*%s)['"]`, regexp.QuoteMeta(ext))
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				files = append(files, match[1])
			}
		}
	}

	return files
}

// extractJSFilesFromContent 从内容中提取JS文件
func (bs *BrowserScan) extractJSFilesFromContent(content string) []string {
	pattern := `<script[^>]*src=['"]([^'"]*\.js[^'"]*)['"]`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(content, -1)

	var jsFiles []string
	for _, match := range matches {
		if len(match) > 1 {
			jsFiles = append(jsFiles, match[1])
		}
	}

	return jsFiles
}

// generateAPIEndpoints 生成API端点
func (bs *BrowserScan) generateAPIEndpoints(task *ScanTask) {
	if task.Results == nil {
		return
	}

	baseURL, err := url.Parse(task.URL)
	if err != nil {
		return
	}

	apiRoutes := task.Config.ApiRoutes
	if len(apiRoutes) == 0 {
		apiRoutes = task.Config.GlobalApiRoutes
	}

	// 默认API路由
	if len(apiRoutes) == 0 {
		apiRoutes = []string{
			"/api", "/api/v1", "/api/v2", "/v1", "/v2",
			"/rest", "/rest/api", "/graphql", "/graphql/v1",
		}
	}

	endpoints := []string{}

	// 从提取的路径生成API端点
	for _, path := range task.Results.Paths {
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/v") || strings.HasPrefix(path, "/rest") {
			fullURL := baseURL.Scheme + "://" + baseURL.Host + path
			endpoints = append(endpoints, fullURL)
		}
	}

	// 从API路由生成端点
	for _, route := range apiRoutes {
		for _, path := range task.Results.Paths {
			if strings.HasPrefix(path, route) {
				fullURL := baseURL.Scheme + "://" + baseURL.Host + path
				endpoints = append(endpoints, fullURL)
			}
		}
	}

	task.Results.APIEndpoints = bs.unique(endpoints)
}

// unique 去重
func (bs *BrowserScan) unique(strs []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, str := range strs {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// addLog 添加日志
func (bs *BrowserScan) addLog(taskID, level, message string) {
	bs.tasksMutex.Lock()
	defer bs.tasksMutex.Unlock()

	task, exists := bs.tasks[taskID]
	if !exists {
		return
	}

	logEntry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Message: message,
	}

	// 将日志添加到内存中（保留完整日志）
	task.Logs = append(task.Logs, logEntry)

	// 将日志追加写入文件（异步执行，避免阻塞）
	go bs.appendLogToFile(taskID, logEntry)

	// 发送日志事件
	if bs.GetCtx() != nil {
		runtime.EventsEmit(bs.GetCtx(), "browser_scan_log", map[string]interface{}{
			"task_id": taskID,
			"log":     logEntry,
		})
	}
}

// appendLogToFile 将日志追加写入文件
func (bs *BrowserScan) appendLogToFile(taskID string, logEntry LogEntry) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	_ = os.MkdirAll(logDir, 0755)

	// 日志文件路径：使用任务ID作为文件名
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	// 打开或创建日志文件（追加模式）
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 写入文件失败时不中断程序执行
		return
	}
	defer file.Close()

	// 将日志条目序列化为JSON并写入文件（每行一个JSON）
	logData, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	_, _ = file.Write(logData)
	_, _ = file.WriteString("\n")
}

// GetTask 获取任务（优先从内存，如果不存在则从文件加载）
func (bs *BrowserScan) GetTask(taskID string) (*ScanTask, error) {
	// 先尝试从内存中获取
	bs.tasksMutex.RLock()
	task, exists := bs.tasks[taskID]
	if exists {
		// 返回任务副本
		taskCopy := *task
		bs.tasksMutex.RUnlock()
		return &taskCopy, nil
	}
	bs.tasksMutex.RUnlock()

	// 如果内存中不存在，尝试从文件加载
	return bs.LoadTaskLog(taskID)
}

// TaskBasicInfo 任务基本信息（用于列表展示，不包含大数据字段）
type TaskBasicInfo struct {
	ID        string     `json:"id"`
	URL       string     `json:"url"`
	Status    string     `json:"status"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// GetTasksBasicInfo 获取所有任务的基本信息（优化版本，不加载完整数据）
// 只返回基本信息，用于任务管理列表展示，避免加载大量数据导致卡顿
func (bs *BrowserScan) GetTasksBasicInfo() ([]*TaskBasicInfo, error) {
	bs.tasksMutex.RLock()

	// 先获取内存中的任务基本信息
	memoryTasks := make(map[string]*TaskBasicInfo)
	for _, task := range bs.tasks {
		memoryTasks[task.ID] = &TaskBasicInfo{
			ID:        task.ID,
			URL:       task.URL,
			Status:    task.Status,
			StartTime: task.StartTime,
			EndTime:   task.EndTime,
		}
	}

	bs.tasksMutex.RUnlock()

	// 获取已保存的任务基本信息（只读取文件头信息，不完整加载）
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	savedTasks := make(map[string]*TaskBasicInfo)

	if _, err := os.Stat(logDir); err == nil {
		files, err := ioutil.ReadDir(logDir)
		if err == nil {
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}

				taskID := strings.TrimSuffix(file.Name(), ".json")

				// 如果内存中已有该任务，跳过（内存中的优先）
				if _, exists := memoryTasks[taskID]; exists {
					continue
				}

				// 优化：使用 map[string]json.RawMessage 只解析顶层字段，不加载完整数据
				// 这样可以避免解析 results、packetLogs 等大数据字段
				logFile := filepath.Join(logDir, file.Name())
				data, err := ioutil.ReadFile(logFile)
				if err != nil {
					continue
				}

				// 使用 map 来部分解析 JSON，只提取顶层字段
				var taskData map[string]json.RawMessage
				if err := json.Unmarshal(data, &taskData); err != nil {
					continue
				}

				// 只解析基本信息字段
				var basicInfo TaskBasicInfo
				basicInfo.ID = taskID // 使用文件名作为ID

				// 解析 url 字段
				if urlData, ok := taskData["url"]; ok {
					json.Unmarshal(urlData, &basicInfo.URL)
				}

				// 解析 status 字段
				if statusData, ok := taskData["status"]; ok {
					json.Unmarshal(statusData, &basicInfo.Status)
				}

				// 解析 start_time 字段（尝试直接解析为 time.Time）
				if startTimeData, ok := taskData["start_time"]; ok {
					var startTime time.Time
					if err := json.Unmarshal(startTimeData, &startTime); err == nil && !startTime.IsZero() {
						basicInfo.StartTime = startTime
					}
				}

				// 解析 end_time 字段（可能为 null）
				if endTimeData, ok := taskData["end_time"]; ok && len(endTimeData) > 0 && string(endTimeData) != "null" {
					var endTime *time.Time
					if err := json.Unmarshal(endTimeData, &endTime); err == nil && endTime != nil {
						basicInfo.EndTime = endTime
					}
				}

				savedTasks[taskID] = &basicInfo
			}
		}
	}

	// 合并内存任务和已保存任务（内存中的任务优先）
	allTasks := make(map[string]*TaskBasicInfo)

	// 先添加已保存的任务
	for _, task := range savedTasks {
		allTasks[task.ID] = task
	}

	// 然后添加内存中的任务（覆盖已保存的任务）
	for _, task := range memoryTasks {
		allTasks[task.ID] = task
	}

	// 转换为列表
	tasks := []*TaskBasicInfo{}
	for _, task := range allTasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetAllTasks 获取所有任务（包括内存中的和已保存的）
func (bs *BrowserScan) GetAllTasks() ([]*ScanTask, error) {
	bs.tasksMutex.RLock()

	// 先获取内存中的任务
	memoryTasks := make(map[string]*ScanTask)
	for _, task := range bs.tasks {
		taskCopy := *task
		memoryTasks[task.ID] = &taskCopy
	}

	bs.tasksMutex.RUnlock()

	// 获取已保存的任务
	savedTasks, err := bs.GetAllSavedTasks()
	if err != nil {
		// 如果获取已保存任务失败，只返回内存中的任务
		tasks := []*ScanTask{}
		for _, task := range memoryTasks {
			tasks = append(tasks, task)
		}
		return tasks, nil
	}

	// 合并内存任务和已保存任务（内存中的任务优先）
	allTasks := make(map[string]*ScanTask)

	// 先添加已保存的任务
	for _, task := range savedTasks {
		allTasks[task.ID] = task
	}

	// 然后添加内存中的任务（覆盖已保存的任务）
	for _, task := range memoryTasks {
		allTasks[task.ID] = task
	}

	// 转换为列表
	tasks := []*ScanTask{}
	for _, task := range allTasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// StopTask 停止任务
func (bs *BrowserScan) StopTask(taskID string) error {
	bs.tasksMutex.Lock()
	task, exists := bs.tasks[taskID]
	if !exists {
		bs.tasksMutex.Unlock()
		return fmt.Errorf("任务不存在")
	}

	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	if task.page != nil {
		_ = task.page.Close()
	}
	if task.browser != nil {
		_ = task.browser.Close()
	}

	task.Status = "stopped"
	now := time.Now()
	task.EndTime = &now

	bs.addLog(taskID, "info", "任务已停止")

	// 释放锁后再保存，避免死锁
	bs.tasksMutex.Unlock()

	// 异步保存任务，不阻塞返回
	go func() {
		_ = bs.SaveTaskLog(taskID)
	}()

	return nil
}

// DeleteTask 删除任务（包括内存中的任务和保存的文件）
func (bs *BrowserScan) DeleteTask(taskID string) error {
	bs.tasksMutex.Lock()
	task, exists := bs.tasks[taskID]
	if exists {
		// 停止任务
		if task.Status == "running" {
			if task.cancelFunc != nil {
				task.cancelFunc()
			}
			if task.page != nil {
				_ = task.page.Close()
			}
			if task.browser != nil {
				_ = task.browser.Close()
			}
		}
		delete(bs.tasks, taskID)
	}
	bs.tasksMutex.Unlock()

	// 删除任务文件（.json和.log文件）
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	jsonFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	// 删除JSON文件
	if _, err := os.Stat(jsonFile); err == nil {
		_ = os.Remove(jsonFile)
	}

	// 删除日志文件
	if _, err := os.Stat(logFile); err == nil {
		_ = os.Remove(logFile)
	}

	return nil
}

// SaveTaskLog 保存任务日志
func (bs *BrowserScan) SaveTaskLog(taskID string) error {
	task, err := bs.GetTask(taskID)
	if err != nil {
		return err
	}

	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	_ = os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(logFile, data, 0644)
}

// LoadTaskLog 从文件加载任务数据
func (bs *BrowserScan) LoadTaskLog(taskID string) (*ScanTask, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))

	// 如果文件不存在，返回错误
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("任务文件不存在: %s", taskID)
	}

	// 读取任务文件
	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("读取任务文件失败: %v", err)
	}

	var task ScanTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("解析任务文件失败: %v", err)
	}

	return &task, nil
}

// GetAllSavedTasks 获取所有已保存的任务（从文件系统读取）
func (bs *BrowserScan) GetAllSavedTasks() ([]*ScanTask, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")

	// 如果目录不存在，返回空列表
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return []*ScanTask{}, nil
	}

	// 读取目录中的所有JSON文件
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取任务目录失败: %v", err)
	}

	tasks := []*ScanTask{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 只读取.json文件（跳过.log文件）
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 提取任务ID（文件名去掉.json后缀）
		taskID := strings.TrimSuffix(file.Name(), ".json")

		// 加载任务
		task, err := bs.LoadTaskLog(taskID)
		if err != nil {
			// 如果加载失败，跳过该文件
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskLogsCount 获取任务日志总数
func (bs *BrowserScan) GetTaskLogsCount(taskID string) (int, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	// 如果文件不存在，返回0
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return 0, nil
	}

	// 读取日志文件
	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		return 0, fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 统计非空行数
	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			count++
		}
	}

	return count, nil
}

// GetTaskLogs 读取任务日志文件（分页）
// offset: 偏移量（从第几条开始，从0开始）
// limit: 返回的日志数量，0表示返回所有日志
func (bs *BrowserScan) GetTaskLogs(taskID string, offset, limit int) ([]LogEntry, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "browser_scan")
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	// 如果文件不存在，返回空列表
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	// 读取日志文件
	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 按行解析JSON
	lines := strings.Split(string(data), "\n")
	logs := []LogEntry{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var logEntry LogEntry
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			// 解析失败时跳过该行
			continue
		}

		logs = append(logs, logEntry)
	}

	// 分页处理
	total := len(logs)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []LogEntry{}, nil
	}

	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}

	return logs[offset:end], nil
}

// testApiEndpoints 对API端点进行发包测试
func (bs *BrowserScan) testApiEndpoints(task *ScanTask) {
	if task.Results == nil {
		return
	}

	// 使用用户选择的方法，默认GET
	methods := task.Config.TestMethods
	if len(methods) == 0 {
		methods = []string{"GET"}
	}

	// 获取敏感接口关键词（用于过滤）
	sensitiveKeywords := bs.getSensitiveKeywords()

	baseURL, err := url.Parse(task.URL)
	if err != nil {
		bs.addLog(task.ID, "error", "解析目标URL失败")
		return
	}

	allEndpoints := []string{}

	// 1. URL完整路径直接进行发包
	for _, fullURL := range task.Results.URLs {
		// 检查是否是完整的URL
		if strings.HasPrefix(fullURL, "http://") || strings.HasPrefix(fullURL, "https://") {
			allEndpoints = append(allEndpoints, fullURL)
		} else if strings.HasPrefix(fullURL, "//") {
			// 处理协议相对URL
			allEndpoints = append(allEndpoints, baseURL.Scheme+":"+fullURL)
		} else if strings.HasPrefix(fullURL, "/") {
			// 处理绝对路径
			allEndpoints = append(allEndpoints, baseURL.Scheme+"://"+baseURL.Host+fullURL)
		} else {
			// 处理相对路径
			allEndpoints = append(allEndpoints, baseURL.Scheme+"://"+baseURL.Host+"/"+fullURL)
		}
	}

	// 2. API端点处理
	// 获取常见API接口列表
	apiRoutes := task.Config.ApiRoutes
	if len(apiRoutes) == 0 {
		apiRoutes = task.Config.GlobalApiRoutes
	}
	if len(apiRoutes) == 0 {
		apiRoutes = []string{"/api", "/api/v1", "/api/v2", "/rest", "/graphql"}
	}

	// 2.1 先处理已经是完整URL的API端点
	for _, endpoint := range task.Results.APIEndpoints {
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			allEndpoints = append(allEndpoints, endpoint)
		}
	}

	// 2.2 拼接URL地址进行发包（相对路径的API端点）
	for _, endpoint := range task.Results.APIEndpoints {
		// 跳过已经是完整URL的端点（已在上面处理）
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			continue
		}

		// 确保端点以/开头
		if !strings.HasPrefix(endpoint, "/") {
			endpoint = "/" + endpoint
		}

		// 拼接baseURL + API端点
		fullURL := baseURL.Scheme + "://" + baseURL.Host + endpoint
		allEndpoints = append(allEndpoints, fullURL)

		// 2.3 拼接常见API接口作为一级路径，然后再拼接API端点进行发包
		for _, apiRoute := range apiRoutes {
			if apiRoute == "" {
				continue
			}

			// 确保apiRoute以/开头
			if !strings.HasPrefix(apiRoute, "/") {
				apiRoute = "/" + apiRoute
			}

			// 拼接 baseURL + 常见API接口 + API端点
			combinedURL := baseURL.Scheme + "://" + baseURL.Host + apiRoute + endpoint
			allEndpoints = append(allEndpoints, combinedURL)
		}
	}

	// 过滤敏感接口关键词
	allEndpoints = bs.filterSensitiveEndpoints(allEndpoints, sensitiveKeywords)

	// 过滤静态文件（不对png、jpg等静态资源进行发包测试）
	allEndpoints = bs.filterStaticFiles(allEndpoints)

	// 去重
	allEndpoints = bs.unique(allEndpoints)

	if len(allEndpoints) == 0 {
		bs.addLog(task.ID, "info", "没有需要测试的端点")
		return
	}

	bs.addLog(task.ID, "info", fmt.Sprintf("开始对 %d 个端点进行发包测试（包含完整URL和API端点）", len(allEndpoints)))

	// 并发控制：使用配置的线程数，默认10
	workerNum := task.Config.WorkerNum
	if workerNum <= 0 {
		workerNum = 10
	}
	if workerNum > 50 {
		workerNum = 50 // 最大50个线程
	}
	semaphore := make(chan struct{}, workerNum)
	var wg sync.WaitGroup

	for _, endpoint := range allEndpoints {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			bs.testSingleEndpoint(task, url, methods)
		}(endpoint)
	}

	wg.Wait()
	bs.addLog(task.ID, "info", fmt.Sprintf("端点发包测试完成，共测试 %d 个端点", len(allEndpoints)))
}

// extractFirstLevelRoutesFromJS 从JS中提取一级路由
func (bs *BrowserScan) extractFirstLevelRoutesFromJS(task *ScanTask) []string {
	if task.Results == nil {
		return []string{}
	}

	routes := []string{}
	seen := make(map[string]bool)

	// 从Paths中提取一级路由（如 /api/user -> /user）
	for _, path := range task.Results.Paths {
		// 移除开头的斜杠并分割
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			route := "/" + parts[0]
			if !seen[route] {
				routes = append(routes, route)
				seen[route] = true
			}
		}
	}

	// 从IncompletePaths中提取
	for _, path := range task.Results.IncompletePaths {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			route := "/" + parts[0]
			if !seen[route] {
				routes = append(routes, route)
				seen[route] = true
			}
		}
	}

	return routes
}

// testSingleEndpoint 测试单个端点（使用httpx优化）
func (bs *BrowserScan) testSingleEndpoint(task *ScanTask, endpointURL string, methods []string) {
	// 优先使用httpx客户端
	var useHTTPX bool
	var httpxClient *httpx.HTTPX
	if bs.httpxMgr != nil {
		httpxClient = bs.httpxMgr.GetClient()
		useHTTPX = (httpxClient != nil)
	}

	// Fallback: 使用标准http.Client
	var stdClient *http.Client
	if !useHTTPX {
		stdClient = proxy.GlobalProxyManager.GetHTTPClient()
	}

	for _, method := range methods {
		startTime := time.Now()
		var statusCode int
		var bodyContent string
		var requestPacket string
		var responsePacket string

		if useHTTPX && httpxClient != nil {
			// 使用httpx发送请求
			req, err := httpxClient.NewRequest(method, endpointURL)
			if err != nil {
				continue
			}
			req.Request.Header.Set("User-Agent", "EasyTools-BrowserScan/1.0")
			req.Request.Header.Set("Accept", "*/*")

			// 构建请求包
			var reqBuilder strings.Builder
			reqBuilder.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", req.Request.Method, req.Request.URL.RequestURI()))
			reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", req.Request.Host))
			for k, v := range req.Request.Header {
				for _, val := range v {
					reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
				}
			}
			reqBuilder.WriteString("\r\n")
			requestPacket = reqBuilder.String()

			httpxResp, err := httpxClient.Do(req, httpx.UnsafeOptions{})
			if err != nil {
				continue
			}

			statusCode = httpxResp.StatusCode
			// httpx.Response使用Data字段（decoded data）或RawData字段（undecoded data）
			bodyData := httpxResp.Data
			if len(bodyData) == 0 {
				bodyData = httpxResp.RawData
			}
			bodyContent = string(bodyData)
			if len(bodyContent) > 1024*1024 {
				bodyContent = bodyContent[:1024*1024] // 1MB限制
			}

			// 构建响应包
			var respBuilder strings.Builder
			respBuilder.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", httpxResp.StatusCode, http.StatusText(httpxResp.StatusCode)))
			for k, v := range httpxResp.Headers {
				for _, val := range v {
					respBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
				}
			}
			respBuilder.WriteString("\r\n")
			respBuilder.WriteString(bodyContent)
			responsePacket = respBuilder.String()
		} else {
			// 使用标准http.Client（Fallback）
			req, err := http.NewRequest(method, endpointURL, nil)
			if err != nil {
				continue
			}

			req.Header.Set("User-Agent", "EasyTools-BrowserScan/1.0")
			req.Header.Set("Accept", "*/*")

			// 构建请求包
			var reqBuilder strings.Builder
			reqBuilder.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", req.Method, req.URL.RequestURI()))
			reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", req.Host))
			for k, v := range req.Header {
				for _, val := range v {
					reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
				}
			}
			reqBuilder.WriteString("\r\n")
			requestPacket = reqBuilder.String()

			resp, err := stdClient.Do(req)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			statusCode = resp.StatusCode

			// 读取响应体（限制大小）
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB限制
			bodyContent = string(bodyBytes)

			// 构建响应包
			var respBuilder strings.Builder
			respBuilder.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", resp.StatusCode, resp.Status))
			for k, v := range resp.Header {
				for _, val := range v {
					respBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
				}
			}
			respBuilder.WriteString("\r\n")
			respBuilder.Write(bodyBytes)
			responsePacket = respBuilder.String()
		}

		duration := time.Since(startTime).String()

		// 提取响应包信息
		extractedInfo := bs.extractResponseInfo(bodyContent)

		// 创建数据包日志
		packetLog := PacketLog{
			URL:            endpointURL,
			Method:         method,
			StatusCode:     statusCode,
			Duration:       duration,
			RequestPacket:  requestPacket,
			ResponsePacket: responsePacket,
			ExtractedInfo:  extractedInfo,
		}

		// 添加到任务的数据包日志
		bs.tasksMutex.Lock()
		if task.PacketLogs == nil {
			task.PacketLogs = []PacketLog{}
		}
		task.PacketLogs = append(task.PacketLogs, packetLog)
		bs.tasksMutex.Unlock()

		// 发送数据包事件
		if bs.GetCtx() != nil {
			runtime.EventsEmit(bs.GetCtx(), "browser_scan_packet", map[string]interface{}{
				"task_id": task.ID,
				"packet":  packetLog,
			})
		}

		// 只测试第一个成功的方法
		if statusCode < 500 {
			break
		}
	}
}

// extractResponseInfo 使用小程序反编译规则提取响应包信息
func (bs *BrowserScan) extractResponseInfo(content string) []ExtractedInfoItem {
	// 获取小程序反编译规则
	rules, err := bs.getWechatRules()
	if err != nil {
		// 如果获取规则失败，使用默认规则
		rules = bs.getDefaultExtractRules()
	}

	var results []ExtractedInfoItem

	for _, rule := range rules {
		re, err := regexp.Compile(rule)
		if err != nil {
			continue
		}

		matches := re.FindAllString(content, -1)
		for _, match := range matches {
			// 清理引号
			cleaned := strings.Trim(match, `'"`)
			if cleaned != "" {
				results = append(results, ExtractedInfoItem{
					Rule:  rule,
					Value: cleaned,
				})
			}
		}
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueResults []ExtractedInfoItem
	for _, item := range results {
		key := item.Rule + ":" + item.Value
		if !seen[key] {
			seen[key] = true
			uniqueResults = append(uniqueResults, item)
		}
	}

	return uniqueResults
}

// getWechatRules 获取小程序反编译规则（需要调用unwxapp模块）
func (bs *BrowserScan) getWechatRules() ([]string, error) {
	// 这里需要调用unwxapp模块的GetWechatRules方法
	// 由于跨模块调用，我们直接读取配置文件
	configPath := filepath.Join("EasyToolsFiles", "config", "wechat_config.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config struct {
		Rules string `json:"rules"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var rules []string
	if err := json.Unmarshal([]byte(config.Rules), &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

// getDefaultExtractRules 获取默认提取规则
func (bs *BrowserScan) getDefaultExtractRules() []string {
	return []string{
		`(?i)["']?access[_-]?token["']?[^\S\r\n]*[=:][^\S\r\n]*[^(null)]["']?[\w-]+["']?`,
		`(?i)["']?access[_-]?secret["']?[^\S\r\n]*[=:][^\S\r\n]*["']?[\w-]+["']?`,
		`(?i)["']?api[_-]?key["']?[^\S\r\n]*[=:][^\S\r\n]*["']?[\w-]+["']?`,
		`(?i)["']?secret["']?[^\S\r\n]*[=:][^\S\r\n]*["']?[\w-]+["']?`,
		`(?:[1-9]\d{5}(?:(?:18|19|20)\d{2})(?:(?:0[1-9]|1[0-2]))(?:(?:0[1-9]|[12]\d|3[01]))\d{3}[\dXx]|[1-9]\d{7}(?:(?:0[1-9]|1[0-2]))(?:(?:0[1-9]|[12]\d|3[01]))\d{3})`,
		`(?:\+?86)?1[3-9]\d{9}`,
		`(?i)http(s)?://(?:[\w-]+/)*(?:[\w-]+)/*\??(?:[\w\=&-]+)?["']?\)*`,
	}
}

// GetExtractRules 获取所有提取规则
func (bs *BrowserScan) GetExtractRules() (map[string]string, error) {
	config, err := bs.loadConfig()
	if err != nil {
		// 如果加载失败，返回默认规则
		return bs.getDefaultRules(), nil
	}
	return config.Rules, nil
}

// SaveExtractRules 保存所有提取规则
func (bs *BrowserScan) SaveExtractRules(rules map[string]string) error {
	config := ExtractRulesConfig{
		Rules: rules,
	}
	return bs.saveConfig(config)
}

// UpdateExtractRule 更新单个提取规则
func (bs *BrowserScan) UpdateExtractRule(ruleName, rulePattern string) error {
	config, err := bs.loadConfig()
	if err != nil {
		config = &ExtractRulesConfig{Rules: bs.getDefaultRules()}
	}
	if config.Rules == nil {
		config.Rules = make(map[string]string)
	}
	config.Rules[ruleName] = rulePattern
	return bs.saveConfig(*config)
}

// DeleteExtractRule 删除单个提取规则
func (bs *BrowserScan) DeleteExtractRule(ruleName string) error {
	config, err := bs.loadConfig()
	if err != nil {
		return err
	}
	if config.Rules == nil {
		return nil
	}
	delete(config.Rules, ruleName)
	return bs.saveConfig(*config)
}
