package finger

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"EasyTools/app/controller/proxy"
	extractor "EasyTools/app/embed"
	embedconfig "EasyTools/app/embed/config"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
	wappalyzergo "github.com/projectdiscovery/wappalyzergo"
	"github.com/twmb/murmur3"
	"golang.org/x/net/html/charset"
	socksproxy "golang.org/x/net/proxy"
)

const (
	defaultFingerprintTimeout       = 8
	defaultFingerprintThreads       = 20 // 提高默认线程数，加快扫描速度
	defaultDirThreads               = 6
	defaultMaxDirChecks             = 400
	maxBaseBodyBytes          int64 = 2 * 1024 * 1024
	maxDirBodyBytes                 = 512 * 1024
	defaultUserAgent                = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
)

var poweredByRegex = regexp.MustCompile(`(?i)powered\s*by\s*([^\s<|"]+)`)

type FingerprintScanConfig struct {
	Targets        []string          `json:"targets"`
	Timeout        int               `json:"timeout"`
	Threads        int               `json:"threads"`
	DirThreads     int               `json:"dirThreads"`
	MaxDirChecks   int               `json:"maxDirChecks"`
	DirMode        bool              `json:"dirMode"`
	UseWappalyzer  bool              `json:"useWappalyzer"`
	UseTideRules   bool              `json:"useTideRules"`
	UseCmsDict     bool              `json:"useCmsDict"`
	UsePoweredBy   bool              `json:"usePoweredBy"`
	CheckAlive     bool              `json:"checkAlive"` // URL存活判断，开启时只获取站点标题等信息，不进行指纹识别
	Headers        map[string]string `json:"headers"`
	FollowRedirect bool              `json:"followRedirect"`
	TabId          string            `json:"tabId,omitempty"` // 前端标签页ID，用于事件路由
}

type FingerprintResult struct {
	URL            string                `json:"url"`
	Status         int                   `json:"status"`
	Title          string                `json:"title"`
	Technologies   []string              `json:"technologies"`
	CMS            string                `json:"cms"`
	TideMatches    []string              `json:"tideMatches,omitempty"`
	PoweredBy      []string              `json:"poweredBy,omitempty"`
	Server         string                `json:"server,omitempty"`
	ResponseTime   int64                 `json:"responseTime"`
	Length         int                   `json:"length"`
	Headers        map[string]string     `json:"headers,omitempty"`
	Evidence       []FingerprintEvidence `json:"evidence,omitempty"`
	DirectoryHits  []DirectoryHit        `json:"directoryHits,omitempty"`
	Timestamp      int64                 `json:"timestamp"`
	Error          string                `json:"error,omitempty"`
	RequestPacket  string                `json:"requestPacket,omitempty"`  // 请求包原始数据
	ResponsePacket string                `json:"responsePacket,omitempty"` // 响应包原始数据
}

type DirectoryHit struct {
	Path    string `json:"path"`
	Rule    string `json:"rule"`
	Pattern string `json:"pattern"`
}

type FingerprintEvidence struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Extra  string `json:"extra,omitempty"`
}

type FingerprintStats struct {
	TotalTargets      int64   `json:"totalTargets"`
	Completed         int64   `json:"completed"`
	Success           int64   `json:"success"`
	Failed            int64   `json:"failed"`
	StartTime         time.Time `json:"startTime"`
	TargetsPerSecond  float64 `json:"targetsPerSecond"`
	LastProgressRatio float64 `json:"lastProgressRatio"`
	// 新增字段
	ProgressPercent  float64 `json:"progressPercent"`  // 进度百分比
	ElapsedSeconds   float64 `json:"elapsedSeconds"`  // 已用时间（秒）
	EstimatedSeconds float64 `json:"estimatedSeconds"` // 预计剩余时间（秒）
}

type FingerprintScanner struct {
	config      *FingerprintScanConfig
	client      *http.Client
	results     chan FingerprintResult
	errors      chan error
	done        chan bool
	ctx         context.Context
	cancel      context.CancelFunc
	stats       FingerprintStats
	wappalyzer  *wappalyzergo.Wappalyze
	tideRules   []tideRule
	cmsRules    []cmsRule
	fingerRules []fingerRule
	needFavicon bool // 预计算是否需要favicon，避免每次扫描都检查
	cmsDB       *sql.DB
	appBasePath string
	tabId       string // 前端标签页ID，用于事件路由
}

type tideRule struct {
	Name       string
	Expression string
}

type cmsRule struct {
	ID      int
	Name    string
	Path    string
	Pattern string
	Option  string
	Hit     int
}

type fingerRule struct {
	CMS      string   `json:"cms"`
	Method   string   `json:"method"`
	Location string   `json:"location"`
	Keyword  []string `json:"keyword"`
}

type fingerJSON struct {
	Fingerprint []fingerRule `json:"fingerprint"`
}

var (
	datasetMu  sync.RWMutex
	loadedTide []tideRule
	loadedCMS  []cmsRule
	loadedDB   *sql.DB
	loadedPath string
)

const customFingerFile = "finger_custom.json"

func NewFingerprintScannerFromJSON(configJSON string, appPath string) (*FingerprintScanner, error) {
	var config FingerprintScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	config.applyDefaults()
	return NewFingerprintScanner(&config, appPath)
}

func (c *FingerprintScanConfig) applyDefaults() {
	if c.Timeout <= 0 {
		c.Timeout = defaultFingerprintTimeout
	}
	if c.Threads <= 0 {
		c.Threads = defaultFingerprintThreads
	}
	if c.DirThreads <= 0 {
		c.DirThreads = defaultDirThreads
	}
	if c.MaxDirChecks <= 0 {
		c.MaxDirChecks = defaultMaxDirChecks
	}
	if c.Headers == nil {
		c.Headers = map[string]string{}
	}
}

func NewFingerprintScanner(config *FingerprintScanConfig, appPath string) (*FingerprintScanner, error) {
	targets := normalizeTargets(config.Targets)
	if len(targets) == 0 {
		return nil, errors.New("至少需要一个有效的 URL")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 为每个扫描器创建独立的 Transport，避免连接冲突
	baseClient := proxy.GlobalProxyManager.GetHTTPClient()
	baseTransport := baseClient.Transport
	if baseTransport == nil {
		baseTransport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 克隆 Transport 配置但创建新的实例
	var newTransport *http.Transport
	if existingTransport, ok := baseTransport.(*http.Transport); ok {
		// 从现有 Transport 克隆配置，但创建新的连接池
		newTransport = &http.Transport{
			Proxy:                 existingTransport.Proxy,
			DialContext:           existingTransport.DialContext,
			TLSClientConfig:       existingTransport.TLSClientConfig,
			MaxIdleConns:          100,              // 最大空闲连接数
			MaxIdleConnsPerHost:   10,               // 每个主机的最大空闲连接数
			IdleConnTimeout:       90 * time.Second, // 空闲连接超时
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	} else {
		// 如果类型不匹配，创建默认 Transport
		newTransport = &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	client := &http.Client{
		Timeout:   time.Duration(config.Timeout) * time.Second,
		Transport: newTransport,
	}

	if !config.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var wapp *wappalyzergo.Wappalyze
	var err error
	if config.UseWappalyzer {
		wapp, err = wappalyzergo.New()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("初始化 Wappalyzer 失败: %w", err)
		}
	}

	tideRules := []tideRule{}
	cmsRules := []cmsRule{}
	fingerRules := []fingerRule{}
	var cmsDB *sql.DB

	if config.UseTideRules || config.UseCmsDict || config.DirMode {
		dbPath := filepath.Join(appPath, "config", "cms_finger.db")
		if err := ensureFingerprintDataset(dbPath); err != nil {
			cancel()
			return nil, err
		}

		datasetMu.RLock()
		tideRules = append(tideRules, loadedTide...)
		cmsRules = append(cmsRules, loadedCMS...)
		cmsDB = loadedDB
		datasetMu.RUnlock()
	}

	// 自动加载内置与自定义指纹规则（如果存在）
	// 每次调用时读取，确保新增规则即时生效
	fingerRules = loadFingerRules(appPath)
	_ = fingerRules // 避免未使用警告，实际在scanTarget中使用
	// fmt.Printf("[Finger] 初始化扫描器，加载了 %d 条 finger 规则\n", len(fingerRules))

	// 预计算是否需要favicon，避免每次扫描都检查
	needFavicon := false
	for _, rule := range fingerRules {
		if strings.ToLower(strings.TrimSpace(rule.Method)) == "faviconhash" {
			needFavicon = true
			break
		}
	}

	scanner := &FingerprintScanner{
		config:  config,
		client:  client,
		results: make(chan FingerprintResult, len(targets)),
		errors:  make(chan error, len(targets)),
		done:    make(chan bool, 1),
		ctx:     ctx,
		cancel:  cancel,
		stats: FingerprintStats{
			TotalTargets: int64(len(targets)),
			StartTime:    time.Now(),
		},
		wappalyzer:  wapp,
		tideRules:   tideRules,
		cmsRules:    cmsRules,
		fingerRules: fingerRules,
		needFavicon: needFavicon,
		cmsDB:       cmsDB,
		appBasePath: appPath,
		tabId:       config.TabId,
	}

	scanner.config.Targets = targets

	return scanner, nil
}

func ensureFingerprintDataset(dbPath string) error {
	// 如果文件不存在，尝试从嵌入资源中释放
	if _, err := os.Stat(dbPath); err != nil {
		// 先尝试释放整个 config 目录
		extractor.InitExtractor()
		_ = extractor.GetExtractor().ExtractOne("config")
		
		// 无论解压是否成功，都检查文件是否存在
		// 如果仍然不存在，直接从嵌入资源中读取并写入
		if _, err := os.Stat(dbPath); err != nil {
			configDir := filepath.Dir(dbPath)
			if mkdirErr := os.MkdirAll(configDir, 0755); mkdirErr != nil {
				return fmt.Errorf("创建配置目录失败: %w", mkdirErr)
			}
			
			// 从嵌入资源中读取并写入 cms_finger.db
			data, readErr := embedconfig.Assets.ReadFile("cms_finger.db")
			if readErr != nil {
				return fmt.Errorf("无法从嵌入资源读取指纹数据库: %w", readErr)
			}
			
			if writeErr := os.WriteFile(dbPath, data, 0644); writeErr != nil {
				return fmt.Errorf("写入指纹数据库失败: %w", writeErr)
			}
		}
	}

	datasetMu.Lock()
	defer datasetMu.Unlock()

	if loadedPath == dbPath && loadedDB != nil {
		return nil
	}

	if loadedDB != nil {
		_ = loadedDB.Close()
		loadedDB = nil
	}

	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=5000&_mutex=full", filepath.ToSlash(dbPath)))
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	tides, err := loadTideRules(conn)
	if err != nil {
		_ = conn.Close()
		return err
	}

	cms, err := loadCMSRules(conn)
	if err != nil {
		_ = conn.Close()
		return err
	}

	loadedTide = tides
	loadedCMS = cms
	loadedDB = conn
	loadedPath = dbPath

	return nil
}

func loadFingerRules(appPath string) []fingerRule {
	data, err := embedconfig.Assets.ReadFile("finger.json")
	if err != nil {
		// 文件不存在，返回空列表（不报错）
		// 使用 log 输出调试信息
		// fmt.Printf("[Finger] 读取 finger.json 失败: %v\n", err)
		return []fingerRule{}
	}

	var fingerData fingerJSON
	if err := json.Unmarshal(data, &fingerData); err != nil {
		// 解析失败，返回空列表（不报错）
		// fmt.Printf("[Finger] 解析 finger.json 失败: %v\n", err)
		return []fingerRule{}
	}

	// 追加用户自定义指纹（若存在）
	customRules := loadCustomFingerRules(appPath)
	return append(fingerData.Fingerprint, customRules...)
}

// loadCustomFingerRules 读取用户自定义指纹（位于应用数据目录 config/finger_custom.json）
func loadCustomFingerRules(appPath string) []fingerRule {
	path := filepath.Join(appPath, "config", customFingerFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return []fingerRule{}
	}

	var fingerData fingerJSON
	if err := json.Unmarshal(data, &fingerData); err != nil {
		return []fingerRule{}
	}
	return fingerData.Fingerprint
}

// saveCustomFingerRules 覆盖写入用户自定义指纹
func saveCustomFingerRules(appPath string, rules []fingerRule) error {
	path := filepath.Join(appPath, "config", customFingerFile)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(fingerJSON{Fingerprint: rules}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0644)
}

// addCustomFingerRule 追加自定义指纹，避免重复
func addCustomFingerRule(appPath string, rule fingerRule) error {
	// 基础校验
	if strings.TrimSpace(rule.CMS) == "" || strings.TrimSpace(rule.Method) == "" || strings.TrimSpace(rule.Location) == "" {
		return fmt.Errorf("指纹字段缺失：CMS/Method/Location 不能为空")
	}
	trimmedKeywords := make([]string, 0, len(rule.Keyword))
	for _, kw := range rule.Keyword {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			trimmedKeywords = append(trimmedKeywords, kw)
		}
	}
	if len(trimmedKeywords) == 0 {
		return fmt.Errorf("指纹关键词不能为空")
	}
	rule.Keyword = trimmedKeywords

	// 读取现有规则并去重
	existing := loadCustomFingerRules(appPath)
	for _, r := range existing {
		if strings.EqualFold(r.CMS, rule.CMS) &&
			strings.EqualFold(r.Method, rule.Method) &&
			strings.EqualFold(r.Location, rule.Location) &&
			len(r.Keyword) == len(rule.Keyword) {
			match := true
			for idx := range r.Keyword {
				if r.Keyword[idx] != rule.Keyword[idx] {
					match = false
					break
				}
			}
			if match {
				return fmt.Errorf("该指纹已存在")
			}
		}
	}
	existing = append(existing, rule)
	return saveCustomFingerRules(appPath, existing)
}

func loadTideRules(db *sql.DB) ([]tideRule, error) {
	rows, err := db.Query("SELECT name, keys FROM tide")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []tideRule
	for rows.Next() {
		var r tideRule
		if err := rows.Scan(&r.Name, &r.Expression); err != nil {
			return nil, err
		}
		if strings.TrimSpace(r.Expression) == "" {
			continue
		}
		rules = append(rules, r)
	}

	return rules, nil
}

func loadCMSRules(db *sql.DB) ([]cmsRule, error) {
	rows, err := db.Query("SELECT finger_id, cms_name, path, match_pattern, options, hit FROM cms ORDER BY hit ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []cmsRule
	for rows.Next() {
		var r cmsRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.Pattern, &r.Option, &r.Hit); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, nil
}

func (s *FingerprintScanner) Start() (<-chan FingerprintResult, <-chan error) {
	go s.scanWorker()
	return s.results, s.errors
}

func (s *FingerprintScanner) Stop() {
	s.cancel()
	select {
	case s.done <- true:
	default:
	}
}

func (s *FingerprintScanner) GetStats() FingerprintStats {
	elapsed := time.Since(s.stats.StartTime).Seconds()
	total := atomic.LoadInt64(&s.stats.TotalTargets)
	completed := atomic.LoadInt64(&s.stats.Completed)
	success := atomic.LoadInt64(&s.stats.Success)
	failed := atomic.LoadInt64(&s.stats.Failed)

	stats := FingerprintStats{
		TotalTargets: total,
		Completed:    completed,
		Success:      success,
		Failed:       failed,
		StartTime:    s.stats.StartTime,
		ElapsedSeconds: elapsed,
	}
	
	if elapsed > 0 {
		stats.TargetsPerSecond = float64(completed) / elapsed
	}
	
	if total > 0 {
		stats.LastProgressRatio = float64(completed) / float64(total)
		stats.ProgressPercent = stats.LastProgressRatio * 100
		if stats.ProgressPercent > 100 {
			stats.ProgressPercent = 100
		}
	}
	
	// 计算预计剩余时间
	if stats.TargetsPerSecond > 0 && total > completed {
		remaining := float64(total - completed)
		stats.EstimatedSeconds = remaining / stats.TargetsPerSecond
	}
	
	return stats
}

func (s *FingerprintScanner) scanWorker() {
	defer close(s.results)
	defer close(s.errors)

	sem := make(chan struct{}, s.config.Threads)
	var wg sync.WaitGroup

	for _, target := range s.config.Targets {
		select {
		case <-s.ctx.Done():
			wg.Wait()
			s.signalDone()
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(t string) {
			defer wg.Done()
			defer func() { <-sem }()
			s.scanTarget(t)
		}(target)
	}

	wg.Wait()
	s.signalDone()
}

func (s *FingerprintScanner) signalDone() {
	select {
	case s.done <- true:
	default:
	}
}

func (s *FingerprintScanner) scanTarget(target string) {
	if s.ctx.Err() != nil {
		return
	}

	start := time.Now()
	result := FingerprintResult{
		URL:       target,
		Timestamp: time.Now().UnixMilli(),
	}

	resp, body, headerText, requestPacket, responsePacket, err := s.fetch(target, maxBaseBodyBytes)
	if err != nil {
		result.Error = err.Error()
		select {
		case s.results <- result:
		case <-s.ctx.Done():
		}
		select {
		case s.errors <- err:
		default:
		}
		atomic.AddInt64(&s.stats.Completed, 1)
		atomic.AddInt64(&s.stats.Failed, 1)
		return
	}
	defer resp.Body.Close()

	result.Status = resp.StatusCode
	result.Server = resp.Header.Get("Server")
	result.Headers = headersToMap(resp.Header)
	result.ResponseTime = time.Since(start).Milliseconds()
	result.RequestPacket = requestPacket
	result.ResponsePacket = responsePacket

	bodyString := decodeBodyToUTF8(body, resp.Header.Get("Content-Type"))
	title := extractTitle([]byte(bodyString))
	result.Title = title
	// 设置响应体长度
	result.Length = len(body)

	// 如果开启了URL存活判断模式，只获取基本信息，不进行指纹识别
	if s.config.CheckAlive {
		select {
		case s.results <- result:
		case <-s.ctx.Done():
		}
		atomic.AddInt64(&s.stats.Completed, 1)
		if result.Error != "" {
			atomic.AddInt64(&s.stats.Failed, 1)
		} else {
			atomic.AddInt64(&s.stats.Success, 1)
		}
		return
	}

	// 优先使用 finger.json 指纹库进行识别（最高优先级）
	if len(s.fingerRules) > 0 {
		var favhash string
		// 只在需要时获取favicon（已在初始化时预计算）
		if s.needFavicon {
			faviconCtx, faviconCancel := context.WithTimeout(s.ctx, 3*time.Second)
			favhash = getFaviconHash(faviconCtx, bodyString, target)
			faviconCancel()
		}

		matches := s.matchFingerRules(headerText, bodyString, title, favhash)
		if len(matches) > 0 {
			// 优先设置 CMS 字段
			result.CMS = matches[0]
			result.Technologies = appendUnique(result.Technologies, matches...)
			for _, m := range matches {
				result.Evidence = append(result.Evidence, FingerprintEvidence{
					Type:   "finger-rule",
					Name:   m,
					Detail: "finger.json 规则匹配",
				})
			}
			// fmt.Printf("[Finger] URL: %s, 匹配到 CMS: %s\n", target, result.CMS)
		} else {
			// fmt.Printf("[Finger] URL: %s, finger.json 未匹配到任何规则\n", target)
		}
	}

	if s.config.UsePoweredBy {
		result.PoweredBy = detectPoweredBy(bodyString, resp.Header)
		for _, item := range result.PoweredBy {
			result.Evidence = append(result.Evidence, FingerprintEvidence{
				Type:   "powered-by",
				Name:   item,
				Detail: "Powered-By header/markup",
			})
		}
	}

	if s.wappalyzer != nil {
		detections := s.wappalyzer.FingerprintWithInfo(resp.Header, body)
		for name, info := range detections {
			result.Technologies = appendUnique(result.Technologies, formatTechnologyName(name, info))
			result.Evidence = append(result.Evidence, FingerprintEvidence{
				Type:   "wappalyzer",
				Name:   name,
				Detail: strings.Join(info.Categories, ", "),
				Extra:  info.Website,
			})
		}
	}

	if s.config.UseTideRules && len(s.tideRules) > 0 {
		matches := s.matchTideRules(headerText, bodyString, title)
		if len(matches) > 0 {
			result.TideMatches = matches
			result.Technologies = appendUnique(result.Technologies, matches...)
			// 只有在 finger.json 没有匹配到 CMS 时才设置
			if result.CMS == "" {
				result.CMS = matches[0]
				// fmt.Printf("[Finger] URL: %s, TideRules 匹配到 CMS: %s\n", target, result.CMS)
			}
			for _, m := range matches {
				result.Evidence = append(result.Evidence, FingerprintEvidence{
					Type:   "tide-rule",
					Name:   m,
					Detail: "header/title/body rule matched",
				})
			}
		}
	}

	if (s.config.DirMode || s.config.UseCmsDict) && len(s.cmsRules) > 0 {
		if hit := s.matchCmsDirectory(target); hit != nil {
			result.DirectoryHits = append(result.DirectoryHits, *hit)
			// 只有在 finger.json 没有匹配到 CMS 时才设置
			if result.CMS == "" {
				result.CMS = hit.Rule
				// fmt.Printf("[Finger] URL: %s, CmsDirectory 匹配到 CMS: %s\n", target, result.CMS)
			}
			result.Technologies = appendUnique(result.Technologies, hit.Rule)
		}
	}

	select {
	case s.results <- result:
	case <-s.ctx.Done():
	}

	atomic.AddInt64(&s.stats.Completed, 1)
	if result.Error != "" {
		atomic.AddInt64(&s.stats.Failed, 1)
	} else {
		atomic.AddInt64(&s.stats.Success, 1)
	}
}

func (s *FingerprintScanner) fetch(target string, limit int64) (*http.Response, []byte, string, string, string, error) {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, "", "", "", err
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "*/*")
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	// 构建请求包字符串
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", req.Host))
	for k, v := range req.Header {
		for _, val := range v {
			reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	reqBuilder.WriteString("\r\n")
	requestPacket := reqBuilder.String()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, "", requestPacket, "", err
	}

	bodyReader := io.LimitReader(resp.Body, limit)
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		resp.Body.Close()
		return nil, nil, "", requestPacket, "", err
	}

	headerStr := headerToString(resp.Header)
	contentType := resp.Header.Get("Content-Type")
	decodedBody := decodeBodyToUTF8(body, contentType)

	// 构建响应包字符串
	var respBuilder strings.Builder
	respBuilder.WriteString(fmt.Sprintf("%s %d %s\r\n", resp.Proto, resp.StatusCode, resp.Status))
	for k, v := range resp.Header {
		for _, val := range v {
			respBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	respBuilder.WriteString("\r\n")
	respBuilder.WriteString(decodedBody)
	responsePacket := respBuilder.String()

	return resp, body, headerStr, requestPacket, responsePacket, nil
}

func decodeBodyToUTF8(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	charsetName := getCharsetFromContentType(contentType)
	if charsetName != "" && !strings.EqualFold(charsetName, "utf-8") {
		if decoded, ok := convertBytes(body, charsetName); ok {
			return decoded
		}
	}

	if utf8.Valid(body) {
		return string(body)
	}

	if decoded, ok := convertBytes(body, "gbk"); ok {
		return decoded
	}

	if decoded, ok := convertBytes(body, "gb18030"); ok {
		return decoded
	}

	return string(body)
}

func getCharsetFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	if cs, ok := params["charset"]; ok {
		return strings.ToLower(strings.TrimSpace(cs))
	}
	return ""
}

func convertBytes(body []byte, charsetName string) (string, bool) {
	reader, err := charset.NewReaderLabel(charsetName, bytes.NewReader(body))
	if err != nil {
		return "", false
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

func extractTitle(body []byte) string {
	reader := bytes.NewReader(body)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Find("title").First().Text())
}

func detectPoweredBy(body string, headers http.Header) []string {
	results := map[string]struct{}{}

	for _, values := range headers["X-Powered-By"] {
		for _, item := range strings.Split(values, ",") {
			val := strings.TrimSpace(item)
			if val != "" {
				results[val] = struct{}{}
			}
		}
	}

	matches := poweredByRegex.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) > 1 {
			value := strings.TrimSpace(match[1])
			if value != "" {
				results[value] = struct{}{}
			}
		}
	}

	var list []string
	for item := range results {
		list = append(list, item)
	}
	return list
}

func (s *FingerprintScanner) matchTideRules(header, body, title string) []string {
	var matches []string
	ctx := tideContext{
		header: header,
		body:   body,
		title:  title,
	}
	for _, rule := range s.tideRules {
		if evaluateTideExpression(rule.Expression, ctx) {
			matches = append(matches, rule.Name)
		}
	}
	return matches
}

func (s *FingerprintScanner) matchFingerRules(header, body, title, favhash string) []string {
	var matches []string
	seen := make(map[string]struct{})

	for _, rule := range s.fingerRules {
		if _, ok := seen[rule.CMS]; ok {
			continue
		}

		var matched bool
		method := strings.ToLower(strings.TrimSpace(rule.Method))

		// ===================== favicon哈希匹配（优先处理）=====================
		if method == "faviconhash" {
			// favicon 哈希匹配：直接比较哈希值（字符串格式）
			// 只有当favicon哈希值有效时才进行匹配
			if favhash != "" && favhash != "0" {
				if len(rule.Keyword) > 0 {
					expectedHash := strings.TrimSpace(rule.Keyword[0])
					// 直接进行字符串比较（favicon哈希值在finger.json中已经是字符串格式）
					// 支持精确匹配，同时支持数字格式的比较（处理可能的类型差异）
					if favhash == expectedHash {
						matched = true
					} else {
						// 尝试将两者都转换为整数进行比较（处理可能的格式差异）
						if dataHashInt, err1 := strconv.Atoi(favhash); err1 == nil {
							if expectedHashInt, err2 := strconv.Atoi(expectedHash); err2 == nil {
								if dataHashInt == expectedHashInt {
									matched = true
								}
							}
						}
					}
				}
			}

			if matched {
				matches = append(matches, rule.CMS)
				seen[rule.CMS] = struct{}{}
			}
			continue // favicon匹配后直接继续下一个规则
		}

		// ===================== 其他匹配方法（keyword、regular等）=====================
		var content string
		location := strings.ToLower(strings.TrimSpace(rule.Location))

		// 根据位置获取要匹配的内容
		switch location {
		case "body":
			content = body
		case "title":
			content = title
		case "header":
			content = header
		default:
			continue
		}

		// 根据匹配方法进行匹配
		if content != "" {
			switch method {
			case "keyword":
				matched = matchKeywords(content, rule.Keyword)
			case "regex", "regx":
				matched = matchRegex(content, rule.Keyword)
			default:
				continue
			}
		}

		if matched {
			matches = append(matches, rule.CMS)
			seen[rule.CMS] = struct{}{}
		}
	}

	return matches
}

func matchKeywords(content string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}

	contentLower := strings.ToLower(content)
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		if !strings.Contains(contentLower, strings.ToLower(keyword)) {
			return false
		}
	}
	return true
}

func matchRegex(content string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matched, err := regexp.MatchString(pattern, content)
		if err != nil {
			continue
		}
		if !matched {
			return false
		}
	}
	return true
}

func (s *FingerprintScanner) matchCmsDirectory(target string) *DirectoryHit {
	rules := s.cmsRules
	if s.config.MaxDirChecks > 0 && len(rules) > s.config.MaxDirChecks {
		rules = rules[:s.config.MaxDirChecks]
	}

	sem := make(chan struct{}, s.config.DirThreads)
	resultCh := make(chan *DirectoryHit, 1)
	var wg sync.WaitGroup
	var once sync.Once

	for _, rule := range rules {
		select {
		case <-s.ctx.Done():
			wg.Wait()
			close(resultCh)
			return nil
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(r cmsRule) {
			defer wg.Done()
			defer func() { <-sem }()

			if hit := s.checkCmsRule(target, r); hit != nil {
				once.Do(func() {
					updateCmsHit(s.cmsDB, r.ID, r.Hit)
					resultCh <- hit
				})
			}
		}(rule)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	return <-resultCh
}

func (s *FingerprintScanner) checkCmsRule(target string, rule cmsRule) *DirectoryHit {
	fullURL, err := resolveURL(target, rule.Path)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	bodyReader := io.LimitReader(resp.Body, maxDirBodyBytes)
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil
	}

	bodyText := string(body)
	switch strings.ToLower(rule.Option) {
	case "md5":
		hash := md5.Sum(body)
		if strings.EqualFold(hex.EncodeToString(hash[:]), strings.TrimSpace(rule.Pattern)) {
			return &DirectoryHit{Path: rule.Path, Rule: rule.Name, Pattern: "MD5"}
		}
	case "keyword":
		if strings.Contains(strings.ToLower(bodyText), strings.ToLower(rule.Pattern)) {
			return &DirectoryHit{Path: rule.Path, Rule: rule.Name, Pattern: rule.Pattern}
		}
	case "regx", "regex":
		match, _ := regexp.MatchString(rule.Pattern, bodyText)
		if match {
			return &DirectoryHit{Path: rule.Path, Rule: rule.Name, Pattern: rule.Pattern}
		}
	}

	return nil
}

func resolveURL(baseURL, addition string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(addition)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(ref).String(), nil
}

func updateCmsHit(db *sql.DB, id int, current int) {
	if db == nil {
		return
	}
	_, _ = db.Exec("UPDATE cms SET hit = ? WHERE finger_id = ?", current+1, id)
}

type tideContext struct {
	header string
	body   string
	title  string
}

func evaluateTideExpression(expr string, ctx tideContext) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}

	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evaluateTideExpression(strings.TrimSpace(expr[1:len(expr)-1]), ctx)
	}

	if parts := splitExpression(expr, "||"); len(parts) > 1 {
		for _, part := range parts {
			if evaluateTideExpression(part, ctx) {
				return true
			}
		}
		return false
	}

	if parts := splitExpression(expr, "&&"); len(parts) > 1 {
		for _, part := range parts {
			if !evaluateTideExpression(part, ctx) {
				return false
			}
		}
		return true
	}

	return evaluateSingleRule(expr, ctx)
}

func splitExpression(expr, operator string) []string {
	var parts []string
	depth := 0
	start := 0
	i := 0

	for i < len(expr) {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}

		if depth == 0 && strings.HasPrefix(expr[i:], operator) {
			part := strings.TrimSpace(expr[start:i])
			if part != "" {
				parts = append(parts, part)
			}
			i += len(operator)
			start = i
			continue
		}
		i++
	}

	if start < len(expr) {
		part := strings.TrimSpace(expr[start:])
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}

func evaluateSingleRule(rule string, ctx tideContext) bool {
	rule = strings.TrimSpace(rule)
	if strings.HasPrefix(rule, "(") && strings.HasSuffix(rule, ")") {
		return evaluateTideExpression(strings.TrimSpace(rule[1:len(rule)-1]), ctx)
	}

	if strings.HasPrefix(rule, "title=\"") {
		value := extractQuoted(rule)
		return strings.Contains(strings.ToLower(ctx.title), strings.ToLower(value))
	}

	if strings.HasPrefix(rule, "body=\"") {
		value := extractQuoted(rule)
		return strings.Contains(ctx.body, value)
	}

	if strings.HasPrefix(rule, "header=\"") {
		value := extractQuoted(rule)
		return strings.Contains(ctx.header, value)
	}

	return false
}

func extractQuoted(rule string) string {
	start := strings.Index(rule, "\"")
	end := strings.LastIndex(rule, "\"")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return rule[start+1 : end]
}

func normalizeTargets(targets []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, raw := range targets {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "http://") && !strings.HasPrefix(t, "https://") {
			t = "http://" + t
		}
		if _, err := url.ParseRequestURI(t); err != nil {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

func headersToMap(header http.Header) map[string]string {
	result := make(map[string]string, len(header))
	for k, v := range header {
		result[k] = strings.Join(v, "; ")
	}
	return result
}

func headerToString(header http.Header) string {
	var builder strings.Builder
	for k, v := range header {
		builder.WriteString(k)
		builder.WriteString(": ")
		builder.WriteString(strings.Join(v, "; "))
		builder.WriteString("\n")
	}
	return builder.String()
}

func appendUnique(list []string, items ...string) []string {
	exist := map[string]struct{}{}
	for _, v := range list {
		exist[v] = struct{}{}
	}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := exist[item]; ok {
			continue
		}
		exist[item] = struct{}{}
		list = append(list, item)
	}
	return list
}

func formatTechnologyName(name string, info wappalyzergo.AppInfo) string {
	return name
}

// ===================== Favicon哈希计算 =====================

// getFaviconHash 获取favicon哈希 (使用可取消的请求)
func getFaviconHash(ctx context.Context, httpbody string, turl string) string {
	// 尝试多种方式获取favicon路径
	var faviconpath string
	u, err := url.Parse(turl)
	if err != nil {
		return "0"
	}
	baseURL := u.Scheme + "://" + u.Host

	// 方法1: 从HTML中查找favicon链接（支持多种格式）
	faviconpaths := regexpjs(`<link[^>]*rel=["'](?:shortcut\s+)?icon["'][^>]*href=["']([^"']+)["']`, httpbody)
	if len(faviconpaths) == 0 {
		// 尝试更宽松的匹配
		faviconpaths = regexpjs(`href=["']([^"']*favicon[^"']*)["']`, httpbody)
	}

	if len(faviconpaths) > 0 && len(faviconpaths[0]) > 1 {
		fav := faviconpaths[0][1]
		fav = strings.TrimSpace(fav)
		if strings.HasPrefix(fav, "//") {
			faviconpath = u.Scheme + ":" + fav
		} else if strings.HasPrefix(fav, "http://") || strings.HasPrefix(fav, "https://") {
			faviconpath = fav
		} else if strings.HasPrefix(fav, "/") {
			faviconpath = baseURL + fav
		} else {
			faviconpath = baseURL + "/" + fav
		}
	} else {
		// 默认尝试 /favicon.ico
		faviconpath = baseURL + "/favicon.ico"
	}

	return faviconHash(ctx, faviconpath)
}

// regexpjs 正则匹配
func regexpjs(reg string, resp string) [][]string {
	reg1 := regexp.MustCompile(reg)
	if reg1 == nil {
		return nil
	}
	return reg1.FindAllStringSubmatch(resp, -1)
}

// Mmh3Hash32 计算Murmur3哈希
func Mmh3Hash32(raw []byte) string {
	h32 := murmur3.New32()
	_, err := h32.Write(raw)
	if err != nil {
		return "0"
	}
	return fmt.Sprintf("%d", int32(h32.Sum32()))
}

// StandBase64 标准化Base64编码
func StandBase64(braw []byte) []byte {
	bckd := base64.StdEncoding.EncodeToString(braw)
	var buffer strings.Builder
	for i := 0; i < len(bckd); i++ {
		buffer.WriteByte(bckd[i])
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	return []byte(buffer.String())
}

// faviconHash 获取favicon哈希 (支持 ctx 取消)
// 优先使用全局代理
func faviconHash(ctx context.Context, host string) string {
	// 创建带超时的Transport（优化超时时间，加快失败速度）
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		},
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second, // 连接超时
			KeepAlive: 15 * time.Second,
			DualStack: true,
		}).DialContext,
		ResponseHeaderTimeout: 3 * time.Second, // 响应头超时
		TLSHandshakeTimeout:   2 * time.Second, // TLS握手超时
		MaxIdleConns:          500,              // 优化：增加最大空闲连接数
		MaxIdleConnsPerHost:  100,              // 优化：增加每个主机的连接数
		IdleConnTimeout:      90 * time.Second, // 优化：增加空闲连接超时时间
		DisableKeepAlives:    false,            // 启用连接复用
	}

	// 优先使用全局代理
	globalProxyConfig := proxy.GlobalProxyManager.GetGlobalProxy()
	if globalProxyConfig.Enabled && globalProxyConfig.Config != nil {
		config := globalProxyConfig.Config
		switch config.Type {
		case "socks5":
			// SOCKS5 代理特殊处理
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
		default:
			// HTTP/HTTPS 代理
			var proxyURLStr string
			scheme := config.Type
			if scheme != "http" && scheme != "https" {
				scheme = "http"
			}
			proxyURLStr = fmt.Sprintf("%s://%s:%s", scheme, config.Host, config.Port)
			if config.Auth != nil && config.Auth.Username != "" {
				proxyURLStr = fmt.Sprintf("%s://%s:%s@%s:%s", scheme,
					url.QueryEscape(config.Auth.Username),
					url.QueryEscape(config.Auth.Password),
					config.Host, config.Port)
			}
			if proxyURLStr != "" {
				proxyURL, err := url.Parse(proxyURLStr)
				if err == nil {
					transport.Proxy = http.ProxyURL(proxyURL)
				}
			}
		}
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout:   3 * time.Second, // 总超时时间
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 不跟随重定向，直接返回当前响应
			return http.ErrUseLastResponse
		},
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", host, nil)
	if err != nil {
		return "0"
	}

	// 设置请求头
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "close")
	req.Header.Set("Cache-Control", "no-cache")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			return "0"
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "0"
		}
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return "0"
		}
		return "0"
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// 只处理200状态码的响应
	if resp.StatusCode != 200 {
		return "0"
	}

	// 检查内容类型是否为图标
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image") &&
		!strings.Contains(contentType, "icon") &&
		!strings.Contains(contentType, "octet-stream") {
		return "0"
	}

	// 读取响应体（带超时和大小限制）
	var body []byte
	var readErr error

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second) // 读取超时
	defer readCancel()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		// 限制图标文件大小为100KB
		limitReader := &io.LimitedReader{R: resp.Body, N: 100 * 1024}
		body, readErr = io.ReadAll(limitReader)
	}()

	select {
	case <-readCtx.Done():
		return "0"
	case <-readDone:
		if readErr != nil {
			return "0"
		}
	}

	// 检查图标数据有效性
	if len(body) == 0 {
		return "0"
	}

	// 简单验证是否为有效的图标文件（ICO、PNG等）
	if len(body) >= 4 {
		// 检查ICO文件头
		if body[0] == 0x00 && body[1] == 0x00 &&
			body[2] == 0x01 && body[3] == 0x00 {
			// 有效ICO文件
		} else if len(body) >= 8 {
			// 检查PNG文件头
			pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			isPNG := true
			for i := 0; i < 8; i++ {
				if body[i] != pngHeader[i] {
					isPNG = false
					break
				}
			}
			if !isPNG {
				// 检查GIF文件头
				if !(body[0] == 'G' && body[1] == 'I' && body[2] == 'F' && body[3] == '8') {
					// 非标准图标格式，但仍计算哈希
				}
			}
		}
	}

	// 计算并返回哈希值
	return Mmh3Hash32(StandBase64(body))
}
