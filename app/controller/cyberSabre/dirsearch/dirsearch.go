package dirsearch

import (
	"EasyTools/app/controller/proxy"
	extractor "EasyTools/app/embed"
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

// 扫描结果结构
type ScanResult struct {
	URL               string            `json:"url"`
	Status            int               `json:"status"`
	Size              int64             `json:"size"`              // 响应体大小（文件大小）
	ResponsePacketSize int64            `json:"responsePacketSize"` // 响应包大小（包括响应头）
	Time              int64             `json:"time"`
	Title             string            `json:"title,omitempty"`
	Headers           map[string]string `json:"headers"`
	Content           string            `json:"content,omitempty"`
	Request           RequestInfo       `json:"request"`
	FoundBy           string            `json:"found_by,omitempty"`
	RequestPacket     string            `json:"requestPacket,omitempty"`  // 请求包原始数据
	ResponsePacket    string            `json:"responsePacket,omitempty"` // 响应包原始数据
}

type RequestInfo struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// 扫描配置
type ScanConfig struct {
	Target          string            `json:"target"`
	Targets         []string          `json:"targets,omitempty"` // 支持多URL扫描
	Method          string            `json:"method"`
	Threads         int               `json:"threads"`
	Timeout         int               `json:"timeout"`
	RecursionDepth  int               `json:"recursionDepth"`
	FollowRedirects bool              `json:"followRedirects"`
	Fuzz            FuzzConfig        `json:"fuzz"`
	Filter          FilterConfig      `json:"filter"`
	BuiltinDicts    []string          `json:"builtinDicts"`
	CustomDicts     []string          `json:"customDicts"`
	CustomDict      string            `json:"customDict"`
	Extensions      []string          `json:"extensions,omitempty"` // 用于替换%EXT%占位符的扩展名列表
	Headers         map[string]string `json:"headers"`
	Cookies         []Cookie          `json:"cookies"`
	TabId           string            `json:"tabId,omitempty"` // 前端标签页ID，用于事件路由
}

type FuzzConfig struct {
	Enabled       bool     `json:"enabled"`
	Position      string   `json:"position"`
	Length        int      `json:"length"`
	Charset       []string `json:"charset"`
	CustomChars   string   `json:"customChars"`
	GenerateCount int      `json:"generateCount"`
}

type FilterConfig struct {
	StatusCodes []int    `json:"statusCodes"`
	SizeRange   []int64  `json:"sizeRange"`
	Keywords    []string `json:"keywords"`
	Regex       string   `json:"regex"`
}

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// 字典管理器
type DictManager struct {
	builtinDicts map[string][]string
	customDicts  map[string][]string
	fuzzCache    map[string][]string
	mu           sync.RWMutex
}

// 扫描器
type Scanner struct {
	config     *ScanConfig
	dictMgr    *DictManager
	client     *http.Client      // 保留用于兼容性，但不再使用
	httpxMgr   *proxy.HTTPXManager // 使用httpx管理器
	results    chan ScanResult
	errors     chan error
	done       chan bool
	stats      ScanStats
	ctx        context.Context
	cancel     context.CancelFunc
	visitedURL sync.Map
	tabId      string // 前端标签页ID，用于事件路由
}

type ScanStats struct {
	TotalRequests     int64   // 已发送/完成的请求数
	PlannedTotal      int64   // 预估总请求数（根据字典+Fuzz路径数量计算）
	Success           int64   // 成功请求数
	Failed            int64   // 失败请求数
	Found             int64   // 发现的路径数
	StartTime         time.Time
	RequestsPerSecond float64 // 每秒请求数
	// 新增字段
	ProgressPercent  float64 `json:"progressPercent"`  // 进度百分比
	ElapsedSeconds   float64 `json:"elapsedSeconds"`  // 已用时间（秒）
	EstimatedSeconds float64 `json:"estimatedSeconds"` // 预计剩余时间（秒）
}

// 初始化扫描器
func NewScanner(config *ScanConfig) (*Scanner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 为每个扫描器创建独立的 Transport，避免连接冲突
	baseClient := proxy.GlobalProxyManager.GetHTTPClient()
	baseTransport := baseClient.Transport

	// 使用前端配置的超时时间
	timeoutDuration := time.Duration(config.Timeout) * time.Second
	// 如果超时时间未设置或为0，使用默认值
	if timeoutDuration <= 0 {
		timeoutDuration = 10 * time.Second
	}

	// 创建独立的 Dialer，避免 Windows 下的端口占用问题
	// 使用前端传入的配置超时时间
	dialer := &net.Dialer{
		Timeout:   timeoutDuration,  // 使用前端配置的超时时间
		KeepAlive: 30 * time.Second, // KeepAlive 时间
	}

	// 克隆 Transport 配置但创建新的实例
	var newTransport *http.Transport
	if existingTransport, ok := baseTransport.(*http.Transport); ok {
		// 从现有 Transport 克隆配置，但创建新的 DialContext 和连接池
		// 关键：不共享 DialContext，为每个 Transport 创建独立的 Dialer
		var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
		if existingTransport.DialContext != nil {
			// 如果存在代理或特殊配置，包装原有的 DialContext
			originalDialContext := existingTransport.DialContext
			dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 先尝试使用原始 DialContext（可能包含代理配置）
				conn, err := originalDialContext(ctx, network, addr)
				if err == nil {
					return conn, nil
				}
				// 如果失败，使用独立的 Dialer 作为后备
				return dialer.DialContext(ctx, network, addr)
			}
		} else {
			// 没有特殊的 DialContext，直接使用独立的 Dialer
			dialContext = dialer.DialContext
		}

		newTransport = &http.Transport{
			Proxy:                 existingTransport.Proxy, // 保留代理配置
			DialContext:           dialContext,             // 使用新的独立 DialContext
			TLSClientConfig:       existingTransport.TLSClientConfig,
			MaxIdleConns:          200, // 增加最大空闲连接数
			MaxIdleConnsPerHost:   50,  // 增加每个主机的最大空闲连接数（Windows 需要更多）
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeoutDuration, // 使用前端配置的超时时间
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false, // 启用连接复用
			ForceAttemptHTTP2:     false, // 禁用 HTTP/2，减少复杂性
		}
	} else {
		// 如果类型不匹配，创建默认 Transport
		newTransport = &http.Transport{
			DialContext:           dialer.DialContext, // 使用独立的 DialContext
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeoutDuration, // 使用前端配置的超时时间
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
			ForceAttemptHTTP2:     false,
		}
	}

	client := &http.Client{
		Timeout:   timeoutDuration, // 使用前端配置的超时时间
		Transport: newTransport,
	}

	// 跳转跟随判断
	if config.FollowRedirects {
		client.CheckRedirect = nil // 默认跟随
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不自动跳转
		}
	}

	// 初始化字典管理器
	dictMgr, err := NewDictManager()
	if err != nil {
		cancel() // 确保在错误情况下也取消 context
		return nil, err
	}

	// 加载选中的字典
	for _, dictName := range config.BuiltinDicts {
		dictMgr.LoadBuiltinDict(dictName)
	}

	for _, dictFile := range config.CustomDicts {
		dictMgr.LoadCustomDict(dictFile)
	}

	// 加载自定义字典文本
	if config.CustomDict != "" {
		dictMgr.LoadCustomDictText(config.CustomDict)
	}

	// 获取httpx管理器
	httpxMgr := proxy.GetHTTPXManager()

	scanner := &Scanner{
		config:   config,
		dictMgr:  dictMgr,
		client:   client, // 保留用于兼容性，但不再使用
		httpxMgr: httpxMgr,
		results:  make(chan ScanResult, 1000),
		errors:   make(chan error, 100),
		done:     make(chan bool, 1),
		ctx:      ctx,
		cancel:   cancel,
		stats: ScanStats{
			StartTime: time.Now(),
		},
		tabId: config.TabId,
	}

	// 预先计算预估总请求数量，用于进度展示
	paths := scanner.generateScanPaths()
	scanner.stats.PlannedTotal = int64(len(paths))

	// cancel 函数被存储在 scanner 中，会在 Stop() 方法中被调用
	_ = cancel // 标记 cancel 被使用，避免 linter 警告

	return scanner, nil
}

// 开始扫描
func (s *Scanner) Start() (<-chan ScanResult, <-chan error) {
	go s.scanWorker()
	return s.results, s.errors
}

// 停止扫描
func (s *Scanner) Stop() {
	s.cancel()
	// 安全地发送停止信号，如果 channel 已关闭则忽略
	select {
	case s.done <- true:
	default:
		// channel 可能已关闭或已满，忽略
	}
}

// 扫描工作器
func (s *Scanner) scanWorker() {
	defer close(s.results)
	defer close(s.errors)

	// 检查是否在启动前就已经取消
	select {
	case <-s.ctx.Done():
		select {
		case s.done <- true:
		default:
		}
		return
	default:
	}

	// 生成扫描路径
	paths := s.generateScanPaths()

	// 再次检查是否在生成路径后取消
	select {
	case <-s.ctx.Done():
		select {
		case s.done <- true:
		default:
		}
		return
	default:
	}

	// 创建工作池
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.Threads)

	// 用于等待所有 goroutine 完成或取消
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 启动所有扫描任务
	for _, path := range paths {
		// 检查是否已取消
		select {
		case <-s.ctx.Done():
			// 如果已取消，等待当前正在运行的任务完成
			wg.Wait()
			return
		default:
		}

		// 尝试获取信号量，如果已取消则退出
		select {
		case <-s.ctx.Done():
			wg.Wait()
			return
		case semaphore <- struct{}{}:
			// 成功获取信号量，启动任务
		}

		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			s.scanPath(p)
		}(path)
	}

	// 等待所有任务完成或被取消
	select {
	case <-done:
		// 所有任务完成，发送完成信号，并结束上下文，通知其他协程退出
		select {
		case s.done <- true:
		default:
			// channel 可能已关闭，忽略
		}
		// 正常完成时也触发取消，确保进度上报等使用 context 的协程能及时退出
		s.cancel()
	case <-s.ctx.Done():
		// 被取消，等待当前任务完成
		wg.Wait()
		// 发送取消信号
		select {
		case s.done <- true:
		default:
		}
	}
}

// 生成扫描路径
func (s *Scanner) generateScanPaths() []string {
	var paths []string

	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		return paths
	default:
	}

	// 从字典生成路径
	dictPaths := s.dictMgr.GetPaths()
	paths = append(paths, dictPaths...)

	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		return paths
	default:
	}

	// 如果启用Fuzz，生成Fuzz路径
	if s.config.Fuzz.Enabled {
		fuzzPaths := s.generateFuzzPaths()
		paths = append(paths, fuzzPaths...)
	}

	// 处理%EXT%占位符替换
	if len(s.config.Extensions) > 0 {
		paths = s.expandExtensions(paths)
	}

	return paths
}

// expandExtensions 扩展包含%EXT%占位符的路径
func (s *Scanner) expandExtensions(paths []string) []string {
	var expandedPaths []string

	for _, path := range paths {
		// 检查路径是否包含%EXT%占位符
		if strings.Contains(path, "%EXT%") {
			// 对每个扩展名，替换%EXT%生成新的路径
			for _, ext := range s.config.Extensions {
				// 清理扩展名（去除可能的点号前缀和空格）
				ext = strings.TrimSpace(ext)
				ext = strings.TrimPrefix(ext, ".")
				if ext == "" {
					continue
				}

				// 替换%EXT%占位符
				newPath := strings.ReplaceAll(path, "%EXT%", ext)

				// 检查是否已取消
				select {
				case <-s.ctx.Done():
					return expandedPaths
				default:
				}

				expandedPaths = append(expandedPaths, newPath)
			}
		} else {
			// 不包含%EXT%，直接添加原路径
			expandedPaths = append(expandedPaths, path)
		}
	}

	return expandedPaths
}

// 生成Fuzz路径
func (s *Scanner) generateFuzzPaths() []string {
	var paths []string

	// 生成字符集
	charset := s.buildCharset()

	// 生成Fuzz字符串
	for i := 0; i < s.config.Fuzz.GenerateCount; i++ {
		// 检查是否已取消
		select {
		case <-s.ctx.Done():
			return paths
		default:
		}

		fuzzStr := generateFuzzString(charset, s.config.Fuzz.Length)

		// 根据位置添加到路径
		switch s.config.Fuzz.Position {
		case "prefix":
			paths = append(paths, fuzzStr+"/path")
		case "suffix":
			paths = append(paths, "/path"+fuzzStr)
		case "middle":
			paths = append(paths, "/path/"+fuzzStr+"/file")
		case "full":
			paths = append(paths, fuzzStr)
		}
	}

	return paths
}

// 构建字符集
func (s *Scanner) buildCharset() string {
	var charset string

	for _, c := range s.config.Fuzz.Charset {
		switch c {
		case "lower":
			charset += "abcdefghijklmnopqrstuvwxyz"
		case "upper":
			charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		case "digits":
			charset += "0123456789"
		case "special":
			charset += "!@#$%^&*()_+-=[]{}|;:,.<>?"
		}
	}

	charset += s.config.Fuzz.CustomChars
	return charset
}

// 扫描单个路径
func (s *Scanner) scanPath(path string) {
	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	// 构建完整URL
	fullURL := s.buildURL(path)

	// 检查是否已访问
	if _, visited := s.visitedURL.Load(fullURL); visited {
		return
	}
	s.visitedURL.Store(fullURL, true)

	// 发送请求
	start := time.Now()
	resp, requestPacket, err := s.makeRequest(fullURL)
	elapsed := time.Since(start).Milliseconds()

	atomic.AddInt64(&s.stats.TotalRequests, 1)

	if err != nil {
		// 检查是否因为取消而失败
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		atomic.AddInt64(&s.stats.Failed, 1)

		// 检查错误类型，对常见的网络错误进行更优雅的处理
		errMsg := err.Error()
		isCommonNetworkError := false

		// 检查是否为常见的网络连接错误（连接被关闭、重置等）
		if strings.Contains(errMsg, "connection was forcibly closed") ||
			strings.Contains(errMsg, "wsarecv") ||
			strings.Contains(errMsg, "connection reset") ||
			strings.Contains(errMsg, "broken pipe") ||
			strings.Contains(errMsg, "EOF") ||
			strings.Contains(errMsg, "Only one usage of each socket address") ||
			strings.Contains(errMsg, "connectex") {
			isCommonNetworkError = true
		}

		// 检查是否为网络超时错误
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			isCommonNetworkError = true
		}

		// 对于常见的网络错误，使用更简洁的错误信息，不发送到错误通道
		// 这些是正常的网络情况，不应该作为严重错误处理
		if isCommonNetworkError {
			// 不发送错误事件，只统计失败次数
			return
		}

		// 对于其他错误（如URL解析错误等），正常发送错误
		// 尝试发送错误，如果已取消则忽略
		select {
		case s.errors <- fmt.Errorf("请求失败 %s: %v", fullURL, err):
		case <-s.ctx.Done():
			return
		}
		return
	}

	atomic.AddInt64(&s.stats.Success, 1)

	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		resp.Body.Close()
		return
	default:
	}

	// 检查是否需要过滤
	if s.shouldFilter(resp, int(elapsed)) {
		resp.Body.Close()
		return
	}

	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		resp.Body.Close()
		return
	default:
	}

	// 处理响应
	result := s.processResponse(resp, fullURL, elapsed, requestPacket)
	resp.Body.Close()

	// 发送结果
	select {
	case s.results <- result:
		atomic.AddInt64(&s.stats.Found, 1)
	case <-s.ctx.Done():
		return
	}

	// 如果成功且有内容，检查是否需要递归扫描
	if resp.StatusCode == 200 && s.config.RecursionDepth > 0 {
		s.handleRecursion(result)
	}
}

// 构建URL
func (s *Scanner) buildURL(path string) string {
	baseURL := strings.TrimSuffix(s.config.Target, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 对路径进行URL编码，处理%EXT%等占位符，避免URL解析错误
	// 将路径分段编码，保留斜杠等路径分隔符
	pathSegments := strings.Split(path, "/")
	encodedSegments := make([]string, len(pathSegments))
	for i, segment := range pathSegments {
		if segment == "" {
			encodedSegments[i] = ""
		} else {
			// 使用PathEscape对路径段进行编码
			encodedSegments[i] = url.PathEscape(segment)
		}
	}
	encodedPath := strings.Join(encodedSegments, "/")

	return baseURL + encodedPath
}

// 发送HTTP请求，返回响应和请求包字符串
// 优先使用httpx，如果不可用则回退到标准库
func (s *Scanner) makeRequest(urlStr string) (*http.Response, string, error) {
	// 尝试使用httpx（如果可用且配置正确）
	// 当前先使用标准库实现，确保功能正常
	// httpx集成将在确认Response结构后完善
	return s.makeRequestWithHTTPX(urlStr)
}

// makeRequestWithHTTPX 使用httpx发送请求（当前为占位实现）
func (s *Scanner) makeRequestWithHTTPX(urlStr string) (*http.Response, string, error) {
	// 暂时回退到标准库，httpx集成待完善
	return s.makeRequestFallback(urlStr)
}

// makeRequestFallback 回退到标准库的实现（当httpx不可用时）
func (s *Scanner) makeRequestFallback(urlStr string) (*http.Response, string, error) {
	// 使用 context 创建请求，支持取消
	req, err := http.NewRequestWithContext(s.ctx, s.config.Method, urlStr, nil)
	if err != nil {
		return nil, "", err
	}

	// 添加自定义头部
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	// 添加Cookie
	for _, cookie := range s.config.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	// 设置默认User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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
		return nil, requestPacket, err
	}

	return resp, requestPacket, nil
}

// 检查是否需要过滤
func (s *Scanner) shouldFilter(resp *http.Response, elapsed int) bool {
	// 状态码过滤 - 只过滤用户明确指定的状态码
	// 默认情况下，所有状态码（200, 301, 302, 403, 404等）都应该显示
	if len(s.config.Filter.StatusCodes) > 0 {
		for _, code := range s.config.Filter.StatusCodes {
			if resp.StatusCode == code {
				return true
			}
		}
	}

	// 大小过滤
	size := resp.ContentLength
	if len(s.config.Filter.SizeRange) == 2 {
		min, max := s.config.Filter.SizeRange[0], s.config.Filter.SizeRange[1]
		if size < min || size > max {
			return true
		}
	}

	return false
}

// 处理响应
func (s *Scanner) processResponse(resp *http.Response, urlStr string, elapsed int64, requestPacket string) ScanResult {
	// 检查是否已取消
	select {
	case <-s.ctx.Done():
		// 已取消，返回基本信息
		return ScanResult{
			URL:               urlStr,
			Status:            resp.StatusCode,
			Size:              resp.ContentLength,
			ResponsePacketSize: 0, // 取消时无法计算响应包大小
			Time:              elapsed,
		}
	default:
	}

	// 读取响应体（如果已取消，响应体可能已被关闭，忽略错误）
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 限制1MB
	contentType := resp.Header.Get("Content-Type")
	decodedBody := decodeBodyToUTF8(body, contentType)

	// 计算响应体大小（文件大小）
	// 如果 ContentLength 为 -1（chunked 传输），使用实际读取的 body 长度
	bodySize := resp.ContentLength
	if bodySize < 0 {
		// ContentLength 为 -1 时，使用实际读取的 body 字节长度
		bodySize = int64(len(body))
	}

	// 提取标题
	title := s.extractTitle([]byte(decodedBody))

	// 收集头部信息
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

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
	
	// 计算响应包大小（包括响应头）
	responsePacketSize := int64(len(responsePacket))

	return ScanResult{
		URL:               urlStr,
		Status:            resp.StatusCode,
		Size:              bodySize,              // 响应体大小（文件大小）
		ResponsePacketSize: responsePacketSize,   // 响应包大小（包括响应头）
		Time:              elapsed,
		Title:             title,
		Headers:           headers,
		Content:           decodedBody,
		Request: RequestInfo{
			Method:  s.config.Method,
			Headers: s.config.Headers,
		},
		RequestPacket:  requestPacket,
		ResponsePacket: responsePacket,
	}
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

// 提取标题
func (s *Scanner) extractTitle(body []byte) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return ""
	}

	return doc.Find("title").Text()
}

// 处理递归扫描
func (s *Scanner) handleRecursion(result ScanResult) {
	// TODO: 从HTML中提取链接进行递归扫描
	// 可以使用goquery解析HTML，提取href和src属性
}

// 字典管理器实现
func NewDictManager() (*DictManager, error) {
	dm := &DictManager{
		builtinDicts: make(map[string][]string),
		customDicts:  make(map[string][]string),
		fuzzCache:    make(map[string][]string),
	}

	//// 加载内置字典
	//dm.loadDefaultDicts()

	return dm, nil
}

func (dm *DictManager) LoadBuiltinDict(name string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 首先尝试从文件系统加载字典文件
	// 使用统一的 extractor 获取应用基础目录，与 GetDictList 保持一致
	baseDir := extractor.GetExtractor().GetAppBaseDir()
	dictPath := filepath.Join(baseDir, "tools", "dict", "dirsearch", name)

	// 如果文件存在，从文件加载
	if _, err := os.Stat(dictPath); err == nil {
		file, err := os.Open(dictPath)
		if err == nil {
			defer file.Close()

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					lines = append(lines, line)
				}
			}

			if err := scanner.Err(); err == nil && len(lines) > 0 {
				dm.builtinDicts[name] = lines
				return nil
			}
		}
	}

	// 如果文件不存在，使用硬编码的字典
	switch name {
	case "common", "common.txt":
		dm.builtinDicts["common"] = []string{
			"/admin", "/login", "/dashboard", "/admin.php",
			"/wp-admin", "/administrator", "/backup",
			"/config", "/test", "/debug", "/api",
			"/.git", "/.svn", "/.env", "/robots.txt",
			"/sitemap.xml", "/.htaccess", "/web.config",
		}
	case "api", "api.txt":
		dm.builtinDicts["api"] = []string{
			"/api", "/api/v1", "/api/v2", "/rest",
			"/graphql", "/soap", "/json", "/xml",
			"/api/user", "/api/admin", "/api/auth",
		}
	case "sensitive", "sensitive.txt":
		dm.builtinDicts["sensitive"] = []string{
			"/.env", "/.git/config", "/.svn/entries",
			"/backup.sql", "/database.sql", "/dump.sql",
			"/config.php", "/config.ini", "/.htpasswd",
		}
	case "backup", "backup.txt":
		dm.builtinDicts["backup"] = []string{
			"/backup", "/backups", "/backup.zip",
			"/old", "/bak", "/.bak", "/.old",
		}
	case "admin", "admin.txt":
		dm.builtinDicts["admin"] = []string{
			"/admin", "/administrator", "/wp-admin",
			"/admin.php", "/admin.html", "/admin/login",
		}
	case "dicc", "dicc.txt":
		dm.builtinDicts["dicc"] = []string{
			"/admin", "/administrator", "/wp-admin",
			"/admin.php", "/admin.html", "/admin/login",
			"/.env", "/.git/config", "/.svn/entries",
			"/backup.sql", "/database.sql", "/dump.sql",
			"/config.php", "/config.ini", "/.htpasswd",
			"/api", "/api/v1", "/api/v2", "/rest",
			"/graphql", "/soap", "/json", "/xml",
			"/api/user", "/api/admin", "/api/auth",
			"/backup", "/backups", "/backup.zip",
			"/old", "/bak", "/.bak", "/.old",
			"/admin", "/administrator", "/wp-admin",
			"/admin.php", "/admin.html", "/admin/login",
		}
	}

	return nil
}

func (dm *DictManager) LoadCustomDict(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	dm.mu.Lock()
	dm.customDicts[filepath.Base(filePath)] = lines
	dm.mu.Unlock()

	return scanner.Err()
}

func (dm *DictManager) LoadCustomDictText(text string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	dm.customDicts["custom"] = lines
	return nil
}

func (dm *DictManager) GetPaths() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var allPaths []string
	for _, paths := range dm.builtinDicts {
		allPaths = append(allPaths, paths...)
	}
	for _, paths := range dm.customDicts {
		allPaths = append(allPaths, paths...)
	}

	// 去重
	return uniquePaths(allPaths)
}

// 辅助函数
func generateFuzzString(charset string, length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}
	return unique
}

// GetStats 获取扫描统计信息
func (s *Scanner) GetStats() ScanStats {
	elapsed := time.Since(s.stats.StartTime).Seconds()
	stats := s.stats
	
	if elapsed > 0 {
		stats.RequestsPerSecond = float64(atomic.LoadInt64(&s.stats.TotalRequests)) / elapsed
	}
	
	stats.ElapsedSeconds = elapsed
	
	// 计算进度百分比
	if stats.PlannedTotal > 0 {
		totalRequests := atomic.LoadInt64(&s.stats.TotalRequests)
		stats.ProgressPercent = float64(totalRequests) / float64(stats.PlannedTotal) * 100
		if stats.ProgressPercent > 100 {
			stats.ProgressPercent = 100
		}
	}
	
	// 计算预计剩余时间
	if stats.RequestsPerSecond > 0 && stats.PlannedTotal > atomic.LoadInt64(&s.stats.TotalRequests) {
		remaining := float64(stats.PlannedTotal - atomic.LoadInt64(&s.stats.TotalRequests))
		stats.EstimatedSeconds = remaining / stats.RequestsPerSecond
	}
	
	// 更新原子值
	stats.TotalRequests = atomic.LoadInt64(&s.stats.TotalRequests)
	stats.Success = atomic.LoadInt64(&s.stats.Success)
	stats.Failed = atomic.LoadInt64(&s.stats.Failed)
	stats.Found = atomic.LoadInt64(&s.stats.Found)
	
	return stats
}

// NewScannerFromJSON 从JSON配置创建扫描器
func NewScannerFromJSON(configJSON string) (*Scanner, error) {
	var config ScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	return NewScanner(&config)
}

// 主要API函数（保留向后兼容）
func StartDirectoryScan(configJSON string) (*Scanner, error) {
	return NewScannerFromJSON(configJSON)
}
