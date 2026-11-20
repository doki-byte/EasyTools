package fileDumper

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type IndexDumperApp struct {
	ctx context.Context
}

func NewIndexDumperApp() *IndexDumperApp {
	return &IndexDumperApp{}
}

func (a *IndexDumperApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

type IndexScanResult struct {
	URL       string `json:"url"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type IndexBatchScanRequest struct {
	URLs        []string          `json:"urls"`
	OutputDir   string            `json:"outputDir"`
	Jobs        int               `json:"jobs"`
	Timeout     int               `json:"timeout"`
	Retry       int               `json:"retry"`
	CancelToken string            `json:"cancelToken,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type RetryIndexScanRequest struct {
	URL       string            `json:"url"`
	OutputDir string            `json:"outputDir"`
	Jobs      int               `json:"jobs"`
	Timeout   int               `json:"timeout"`
	Retry     int               `json:"retry"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type ScanIndexResult IndexScanResult

// 全局取消令牌管理
var (
	cancelIndexTokens = make(map[string]context.CancelFunc)
	cancelIndexMutex  sync.Mutex
)

type IndexDumper struct {
	client         *http.Client
	outputDir      string
	jobs           int
	timeout        int
	retry          int
	progressChan   chan<- string
	emitter        func(eventName string, data interface{})
	cancelToken    string
	ctx            context.Context
	cancel         context.CancelFunc
	visitedURLs    sync.Map
	customHeaders  map[string]string
}

func NewIndexDumper(outputDir string, jobs, timeout, retry int, progressChan chan<- string, emitter func(eventName string, data interface{}), cancelToken string, customHeaders map[string]string) *IndexDumper {
	client := proxy.GlobalProxyManager.GetHTTPClient()
	if timeout <= 0 {
		timeout = 30
	}
	client.Timeout = time.Duration(timeout) * time.Second

	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 注册取消函数
	if cancelToken != "" {
		cancelIndexMutex.Lock()
		cancelIndexTokens[cancelToken] = cancel
		cancelIndexMutex.Unlock()
	}

	return &IndexDumper{
		client:        client,
		outputDir:     outputDir,
		jobs:          jobs,
		timeout:       timeout,
		retry:         retry,
		progressChan:  progressChan,
		emitter:       emitter,
		cancelToken:   cancelToken,
		ctx:           ctx,
		cancel:        cancel,
		customHeaders: customHeaders,
	}
}

func (i *IndexDumper) cleanup() {
	if i.cancelToken != "" {
		cancelIndexMutex.Lock()
		delete(cancelIndexTokens, i.cancelToken)
		cancelIndexMutex.Unlock()
	}
}

func (a *IndexDumperApp) CancelIndexScan(cancelToken string) error {
	cancelIndexMutex.Lock()
	defer cancelIndexMutex.Unlock()

	if cancel, exists := cancelIndexTokens[cancelToken]; exists {
		cancel()
		delete(cancelIndexTokens, cancelToken)
		return nil
	}
	return fmt.Errorf("未找到对应的扫描任务")
}

func (a *IndexDumperApp) BatchScan(request IndexBatchScanRequest) ([]IndexScanResult, error) {
	results := make([]IndexScanResult, len(request.URLs))

	if request.Jobs <= 0 {
		request.Jobs = 5
	}
	if request.Timeout <= 0 {
		request.Timeout = 30
	}
	if request.Retry <= 0 {
		request.Retry = 3
	}

	if err := os.MkdirAll(request.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	sem := make(chan struct{}, request.Jobs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var completed int32

	for i, targetURL := range request.URLs {
		// 检查是否已取消
		select {
		case <-context.Background().Done():
			return nil, fmt.Errorf("扫描已取消")
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := a.scanSingleURL(url, request.OutputDir, request.Jobs, request.Timeout, request.Retry, request.CancelToken, request.Headers)

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			// 更新进度
			atomic.AddInt32(&completed, 1)
			progress := float32(completed) / float32(len(request.URLs)) * 100

			if a.ctx != nil {
				msg := fmt.Sprintf("进度更新: %.1f%% (%d/%d)", progress, completed, len(request.URLs))
				runtime.EventsEmit(a.ctx, "index-dumper-log", map[string]interface{}{
					"message": msg,
					"level":   "info",
				})
				// 同步到全局日志
				runtime.EventsEmit(a.ctx, "log_event", map[string]interface{}{
					"message": msg,
					"level":   "info",
					"time":    time.Now().Format("15:04:05"),
					"source":  "信息处理/Index目录遍历",
				})
			}
		}(i, targetURL)
	}

	wg.Wait()
	return results, nil
}

func (a *IndexDumperApp) scanSingleURL(targetURL, outputDir string, jobs, timeout, retry int, cancelToken string, headers map[string]string) IndexScanResult {
	progressChan := make(chan string, 100)
	defer close(progressChan)

	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	dumper := NewIndexDumper(outputDir, jobs, timeout, retry, progressChan, emitter, cancelToken, headers)
	defer dumper.cleanup()

	urlDir := sanitizeFilename(targetURL)
	fullOutputDir := filepath.Join(outputDir, urlDir)

	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return IndexScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("创建目录失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	go func() {
		for progress := range progressChan {
			if a.ctx != nil {
				msg := fmt.Sprintf("[%s] %s", targetURL, progress)
				runtime.EventsEmit(a.ctx, "index-dumper-log", map[string]interface{}{
					"message": msg,
					"level":   "info",
				})
				runtime.EventsEmit(a.ctx, "log_event", map[string]interface{}{
					"message": msg,
					"level":   "info",
					"time":    time.Now().Format("15:04:05"),
					"source":  "信息处理/Index目录遍历",
				})
			}
		}
	}()

	err := dumper.Dump(targetURL, fullOutputDir)
	if err != nil {
		if err.Error() == "context canceled" {
			return IndexScanResult{
				URL:       targetURL,
				Success:   false,
				Message:   "扫描被用户取消",
				Timestamp: time.Now().Format(time.RFC3339),
			}
		}
		return IndexScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("下载失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return IndexScanResult{
		URL:       targetURL,
		Success:   true,
		Message:   "成功下载目录内容",
		OutputDir: fullOutputDir,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (i *IndexDumper) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if i.progressChan != nil {
		select {
		case i.progressChan <- msg:
		default:
		}
	}
	if i.emitter != nil {
		i.emitter("index-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}

func (i *IndexDumper) logSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if i.progressChan != nil {
		select {
		case i.progressChan <- msg:
		default:
		}
	}
	if i.emitter != nil {
		i.emitter("index-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "success",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}

func (i *IndexDumper) logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if i.progressChan != nil {
		select {
		case i.progressChan <- msg:
		default:
		}
	}
	if i.emitter != nil {
		i.emitter("index-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}

// indexListingFetchURL 构造用于获取 Apache/Nginx 目录索引页的 URL。
// 许多站点对「无尾部 /」的路径返回 301，且 Location 指向内网或错误主机；
// 浏览器访问的通常是带尾部斜杠的地址，故目录请求必须带 /。
func indexListingFetchURL(base string, relPath string) string {
	base = strings.TrimSpace(base)
	base = strings.TrimSuffix(base, "/")
	relPath = strings.Trim(relPath, "/")
	if relPath == "" {
		return base + "/"
	}
	return base + "/" + relPath + "/"
}

// indexFetchMode 区分目录 HTML 与文件下载，便于设置 Sec-Fetch-*（部分 nginx/WAF 会校验）。
type indexFetchMode int

const (
	indexFetchDirListing indexFetchMode = iota
	indexFetchFile
)

// applyIndexDumperBrowserHeaders 模拟 Chrome 常见头；不含 Sec-Fetch/Referer（见 setSecFetchAndReferer）。
// 注意：若服务端按 TLS/JA3 指纹拦截，仅靠请求头无法与真 Chrome 完全一致，此时需自定义代理或 curl-impersonate 等。
func applyIndexDumperBrowserHeaders(req *http.Request, mode indexFetchMode) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	// 勿手动设置 Accept-Encoding：Go Transport 在自行添加 gzip 时会自动解压响应体；
	// 若在此处写 Accept-Encoding，部分环境下响应仍为 gzip 二进制，目录特征匹配会失败。
	req.Header.Set("Connection", "keep-alive")
	// Client Hints，与 User-Agent 主版本大致一致即可
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	switch mode {
	case indexFetchDirListing:
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	case indexFetchFile:
		req.Header.Set("Accept", "*/*")
	}
}

// setSecFetchAndReferer 补齐导航类头。rootListing 表示用户入口目录（地址栏直达），与点进子目录区分。
func setSecFetchAndReferer(req *http.Request, absURL string, mode indexFetchMode, rootListing bool) {
	u, err := url.Parse(absURL)
	if err != nil || u.Host == "" {
		return
	}
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	switch mode {
	case indexFetchDirListing:
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-User", "?1")
		if rootListing {
			req.Header.Set("Sec-Fetch-Site", "none")
			return
		}
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		pathTrim := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
		if pathTrim == "" || !strings.Contains(pathTrim, "/") {
			return
		}
		last := strings.LastIndex(pathTrim, "/")
		if last >= 0 {
			parent := pathTrim[:last]
			req.Header.Set("Referer", fmt.Sprintf("%s://%s/%s/", u.Scheme, u.Host, parent))
		}
	case indexFetchFile:
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "no-cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		p := u.Path
		if p == "" || p == "/" {
			return
		}
		dir := path.Dir(p)
		if dir == "." || dir == "/" {
			req.Header.Set("Referer", fmt.Sprintf("%s://%s/", u.Scheme, u.Host))
			return
		}
		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		req.Header.Set("Referer", fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, dir))
	}
}

func applyCustomHeaders(req *http.Request, custom map[string]string) {
	if len(custom) == 0 {
		return
	}
	for k, v := range custom {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		req.Header.Set(textproto.CanonicalMIMEHeaderKey(k), strings.TrimSpace(v))
	}
}

func (i *IndexDumper) applyRequestHeaders(req *http.Request, absURL string, mode indexFetchMode, rootListing bool) {
	applyIndexDumperBrowserHeaders(req, mode)
	setSecFetchAndReferer(req, absURL, mode, rootListing)
	applyCustomHeaders(req, i.customHeaders)
}

func (i *IndexDumper) Dump(targetURL, outputDir string) error {
	i.log("开始Index目录下载: %s", targetURL)

	targetURL = strings.TrimSpace(targetURL)
	targetURL = strings.TrimSuffix(targetURL, "/")

	listURL := indexListingFetchURL(targetURL, "")
	// 检查目录列表是否可用
	if !i.isDirectoryListingAvailable(listURL) {
		return fmt.Errorf("目录列表不可用或已禁用")
	}

	i.logSuccess("目录列表可用，开始递归下载")

	// 开始递归下载（base 统一为无尾部斜杠，便于拼接文件 URL）
	return i.recursiveDownloadIndex(targetURL, outputDir, "", "")
}

func (i *IndexDumper) isDirectoryListingAvailable(targetURL string) bool {
	// 检查是否已取消
	select {
	case <-i.ctx.Done():
		return false
	default:
	}

	req, err := http.NewRequestWithContext(i.ctx, "GET", targetURL, nil)
	if err != nil {
		return false
	}
	i.applyRequestHeaders(req, targetURL, indexFetchDirListing, true)

	resp, err := i.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	content := string(body)

	// 检查常见的目录列表特征
	directoryIndicators := []string{
		"<title>Index of",
		"<title>Directory listing for",
		"Parent Directory",
		"To Parent Directory",
		"[To Parent Directory]",
		"<h1>Index of",
		"<h1>Directory of",
	}

	for _, indicator := range directoryIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	return false
}

func (i *IndexDumper) recursiveDownloadIndex(baseURL, outputDir, urlPath, localPath string) error {
	// 检查是否已取消
	select {
	case <-i.ctx.Done():
		return i.ctx.Err()
	default:
	}

	// 防止目录循环
	visitedKey := baseURL + "|" + urlPath
	if _, visited := i.visitedURLs.LoadOrStore(visitedKey, true); visited {
		i.log("跳过已访问的目录: %s", localPath)
		return nil
	}

	i.log("处理目录: %s", localPath)

	dirURL := indexListingFetchURL(baseURL, urlPath)

	// 确保本地目录存在
	localDir := filepath.Join(outputDir, localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("创建本地目录失败 %s: %v", localDir, err)
	}

	req, err := http.NewRequestWithContext(i.ctx, "GET", dirURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败 %s: %v", urlPath, err)
	}
	i.applyRequestHeaders(req, dirURL, indexFetchDirListing, urlPath == "")

	resp, err := i.client.Do(req)
	if err != nil {
		return fmt.Errorf("获取目录失败 %s: %v", urlPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("目录 %s 返回状态码 %d", urlPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取目录内容失败: %v", err)
	}

	// 解析目录列表
	items := i.parseDirectoryListing(string(body), urlPath)

	fileCount := 0
	dirCount := 0

	for _, item := range items {
		// 检查是否已取消
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		default:
		}

		// URL路径（编码的，用于HTTP请求）
		nextURLPath := item.URLPath
		if urlPath != "" {
			nextURLPath = fmt.Sprintf("%s/%s", urlPath, item.URLPath)
		}

		// 本地路径（解码的，用于文件系统）
		nextLocalPath := filepath.Join(localPath, item.LocalName)

		if item.IsDirectory {
			dirCount++
			// 递归处理子目录
			if err := i.recursiveDownloadIndex(baseURL, outputDir, nextURLPath, nextLocalPath); err != nil {
				i.logError("处理目录失败 %s: %v", nextLocalPath, err)
			}
		} else {
			fileCount++
			// 下载文件
			if i.downloadFile(baseURL, nextURLPath, outputDir, nextLocalPath) {
				i.logSuccess("已下载: %s", nextLocalPath)
			} else {
				i.logError("下载失败: %s", nextLocalPath)
			}
		}
	}

	// 只有在目录为空时才创建标记文件
	if fileCount == 0 && dirCount == 0 {
		emptyDirMarker := filepath.Join(localDir, ".empty_directory")
		if _, err := os.Stat(emptyDirMarker); os.IsNotExist(err) {
			if file, err := os.Create(emptyDirMarker); err == nil {
				file.Close()
				i.log("创建空目录标记: %s", localPath)
			}
		}
	}

	i.log("目录 %s 处理完成: %d 个文件, %d 个子目录", localPath, fileCount, dirCount)
	return nil
}

type DirectoryItem struct {
	URLPath     string // URL路径（编码的）
	LocalName   string // 本地文件名（解码的）
	IsDirectory bool
}

func (i *IndexDumper) parseDirectoryListing(htmlContent, currentPath string) []DirectoryItem {
	var items []DirectoryItem

	// 改进的目录列表解析，处理多种格式
	patterns := []struct {
		regex       *regexp.Regexp
		isDirectory bool
	}{
		// Apache 格式
		{regexp.MustCompile(`<a\s+href="([^"]+)"\s*>(?:<img[^>]+>\s*)?([^<]+)</a>`), false},
		// Nginx 格式
		{regexp.MustCompile(`href="([^"]+)"\s*>(?:<img[^>]+>\s*)?([^<]+)</a>`), false},
		// 简单的链接格式
		{regexp.MustCompile(`<a\s+href="([^"]+)">([^<]+)</a>`), false},
	}

	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matches := pattern.regex.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				href := match[1]
				text := match[2]

				// 跳过无效链接
				if href == "" || href == "../" || href == "./" ||
					strings.HasPrefix(href, "?") || strings.HasPrefix(href, "/") ||
					strings.Contains(href, "://") || seen[href] {
					continue
				}

				seen[href] = true

				// 改进的目录判断逻辑
				isDir := false
				if strings.HasSuffix(href, "/") {
					isDir = true
					// 移除目录末尾的斜杠
					href = strings.TrimSuffix(href, "/")
				} else {
					// 检查链接文本是否包含目录特征
					if strings.Contains(text, "/") ||
						strings.Contains(strings.ToLower(text), "directory") ||
						strings.Contains(strings.ToLower(text), "folder") {
						isDir = true
					} else {
						// 检查文件扩展名，没有扩展名可能是目录
						if !strings.Contains(href, ".") {
							isDir = true
						}
					}
				}

				// 解码URL编码的文件名
				decodedName, err := url.QueryUnescape(href)
				if err != nil {
					// 如果解码失败，使用原始名称
					decodedName = href
				}

				// 清理文件名，移除可能的问题字符
				decodedName = sanitizeFilename(decodedName)

				item := DirectoryItem{
					URLPath:     href,        // 保持URL编码用于HTTP请求
					LocalName:   decodedName, // 解码后的名称用于本地文件系统
					IsDirectory: isDir,
				}

				items = append(items, item)
			}
		}
	}

	return items
}

func (i *IndexDumper) downloadFile(baseURL, urlPath, outputDir, localPath string) bool {
	// 检查是否已取消
	select {
	case <-i.ctx.Done():
		return false
	default:
	}

	fileURL := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(urlPath, "/")

	localFilePath := filepath.Join(outputDir, localPath)

	// 检查文件是否已存在且大小合理
	if info, err := os.Stat(localFilePath); err == nil {
		// 文件已存在，检查大小
		if info.Size() > 0 {
			return true
		}
		// 如果文件大小为0，重新下载
		os.Remove(localFilePath)
	}

	req, err := http.NewRequestWithContext(i.ctx, "GET", fileURL, nil)
	if err != nil {
		return false
	}
	i.applyRequestHeaders(req, fileURL, indexFetchFile, false)

	resp, err := i.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		return false
	}

	file, err := os.Create(localFilePath)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localFilePath)
		return false
	}

	// 验证文件是否成功下载
	if info, err := file.Stat(); err == nil && info.Size() == 0 {
		os.Remove(localFilePath)
		return false
	}

	return true
}

func (a *IndexDumperApp) GetDefaultOutputDir() (string, error) {
	defaultDir := filepath.Join(system.GetAppBaseDir(), "file", "index-dumper-output")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		return "", err
	}
	return defaultDir, nil
}

func (a *IndexDumperApp) ParseURLsFromText(text string) []string {
	var urls []string
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if u, err := url.Parse(line); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			urls = append(urls, line)
		}
	}

	return urls
}

func (a *IndexDumperApp) RetryScan(request RetryIndexScanRequest) (IndexScanResult, error) {
	return a.scanSingleURL(request.URL, request.OutputDir, request.Jobs, request.Timeout, request.Retry, "", request.Headers), nil
}
