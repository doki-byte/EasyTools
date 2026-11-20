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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

type DSStoreDumperApp struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	isScanning atomic.Bool
}

func NewDSStoreDumperApp() *DSStoreDumperApp {
	return &DSStoreDumperApp{}
}

func (a *DSStoreDumperApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

type DSStoreScanResult struct {
	URL       string `json:"url"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type DSStoreBatchScanRequest struct {
	URLs      []string `json:"urls"`
	OutputDir string   `json:"outputDir"`
	Jobs      int      `json:"jobs"`
	Timeout   int      `json:"timeout"`
	Retry     int      `json:"retry"`
}

type RetryDSStoreScanRequest struct {
	URL       string `json:"url"`
	OutputDir string `json:"outputDir"`
	Jobs      int    `json:"jobs"`
	Timeout   int    `json:"timeout"`
	Retry     int    `json:"retry"`
}

type DSStoreDumper struct {
	client       *http.Client
	outputDir    string
	jobs         int
	timeout      int
	retry        int
	progressChan chan<- string
	emitter      func(eventName string, data interface{})
	ctx          context.Context
	isCancelled  func() bool
}

func NewDSStoreDumper(outputDir string, jobs, timeout, retry int, progressChan chan<- string, emitter func(eventName string, data interface{}), ctx context.Context, isCancelled func() bool) *DSStoreDumper {
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

	return &DSStoreDumper{
		client:       client,
		outputDir:    outputDir,
		jobs:         jobs,
		timeout:      timeout,
		retry:        retry,
		progressChan: progressChan,
		emitter:      emitter,
		ctx:          ctx,
		isCancelled:  isCancelled,
	}
}

func (a *DSStoreDumperApp) BatchScan(request DSStoreBatchScanRequest) ([]DSStoreScanResult, error) {
	if a.isScanning.Load() {
		return nil, fmt.Errorf("扫描正在进行中，请先取消当前扫描")
	}

	a.isScanning.Store(true)
	defer a.isScanning.Store(false)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFunc = cancel
	defer func() {
		a.cancelFunc = nil
	}()

	results := make([]DSStoreScanResult, len(request.URLs))

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

	for i, targetURL := range request.URLs {
		// 检查是否已取消
		select {
		case <-ctx.Done():
			return results, fmt.Errorf("扫描已被取消")
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			// 为每个URL创建独立的取消检查函数
			isCancelled := func() bool {
				select {
				case <-ctx.Done():
					return true
				default:
					return false
				}
			}

			result := a.scanSingleURL(url, request.OutputDir, request.Jobs, request.Timeout, request.Retry, isCancelled)

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, targetURL)
	}

	wg.Wait()
	return results, nil
}

func (a *DSStoreDumperApp) scanSingleURL(targetURL, outputDir string, jobs, timeout, retry int, isCancelled func() bool) DSStoreScanResult {
	if isCancelled() {
		return DSStoreScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   "扫描已被取消",
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	progressChan := make(chan string, 100)
	defer close(progressChan)

	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	dumper := NewDSStoreDumper(outputDir, jobs, timeout, retry, progressChan, emitter, a.ctx, isCancelled)

	urlDir := sanitizeDSStoreFilename(targetURL)
	fullOutputDir := filepath.Join(outputDir, urlDir)

	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return DSStoreScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("创建目录失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	go func() {
		for progress := range progressChan {
			fmt.Printf("[%s] %s\n", targetURL, progress)
		}
	}()

	err := dumper.Dump(targetURL, fullOutputDir)
	if err != nil {
		if strings.Contains(err.Error(), "扫描已被取消") {
			return DSStoreScanResult{
				URL:       targetURL,
				Success:   false,
				Message:   "扫描已被取消",
				Timestamp: time.Now().Format(time.RFC3339),
			}
		}
		return DSStoreScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("解析失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return DSStoreScanResult{
		URL:       targetURL,
		Success:   true,
		Message:   "成功从 DS_Store 提取文件",
		OutputDir: fullOutputDir,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (d *DSStoreDumper) log(format string, args ...interface{}) {
	if d.isCancelled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	if d.progressChan != nil {
		select {
		case d.progressChan <- msg:
		default:
		}
	}
	if d.emitter != nil {
		d.emitter("dsstore-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		})
		d.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/DS_Store提取",
		})
	}
}

func (d *DSStoreDumper) logSuccess(format string, args ...interface{}) {
	if d.isCancelled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	if d.progressChan != nil {
		select {
		case d.progressChan <- msg:
		default:
		}
	}
	if d.emitter != nil {
		d.emitter("dsstore-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "success",
			"time":    time.Now().Format(time.RFC3339),
		})
		d.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/DS_Store提取",
		})
	}
}

func (d *DSStoreDumper) logError(format string, args ...interface{}) {
	if d.isCancelled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	if d.progressChan != nil {
		select {
		case d.progressChan <- msg:
		default:
		}
	}
	if d.emitter != nil {
		d.emitter("dsstore-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format(time.RFC3339),
		})
		d.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/DS_Store提取",
		})
	}
}

// 检查是否已取消
func (d *DSStoreDumper) checkCancelled() error {
	if d.isCancelled() {
		return fmt.Errorf("扫描已被取消")
	}
	return nil
}

func (d *DSStoreDumper) Dump(targetURL, outputDir string) error {
	if err := d.checkCancelled(); err != nil {
		return err
	}

	d.log("开始 DS_Store 解析: %s", targetURL)

	targetURL = strings.TrimSuffix(targetURL, "/")

	// 首先尝试下载 .DS_Store 文件
	dsStoreURL := fmt.Sprintf("%s/.DS_Store", targetURL)
	d.log("测试 DS_Store 文件: %s", dsStoreURL)

	// 下载 .DS_Store 文件
	dsStorePath, err := d.downloadDSStoreFile(dsStoreURL, outputDir)
	if err != nil {
		if err.Error() == "扫描已被取消" {
			return err
		}
		return fmt.Errorf("DS_Store 文件未找到或无法访问: %v", err)
	}

	d.logSuccess("找到 DS_Store 文件，开始解析目录结构")

	// 解析 DS_Store 文件
	files, err := d.parseDSStoreFile(dsStorePath)
	if err != nil {
		os.Remove(dsStorePath)
		return fmt.Errorf("解析 DS_Store 文件失败: %v", err)
	}

	// 下载找到的文件
	successCount := 0
	for _, file := range files {
		if err := d.checkCancelled(); err != nil {
			os.Remove(dsStorePath)
			return err
		}

		if d.downloadFile(targetURL, file, outputDir) {
			d.logSuccess("下载文件: %s", file)
			successCount++
		} else {
			d.logError("下载文件失败: %s", file)
		}
	}

	// 清理临时文件
	os.Remove(dsStorePath)

	d.logSuccess("解析完成，成功下载 %d 个文件", successCount)
	return nil
}

// 下载 DS_Store 文件并返回本地路径
func (d *DSStoreDumper) downloadDSStoreFile(dsStoreURL, outputDir string) (string, error) {
	if err := d.checkCancelled(); err != nil {
		return "", err
	}

	localPath := filepath.Join(outputDir, ".DS_Store")

	// 创建请求
	req, err := http.NewRequestWithContext(d.ctx, "GET", dsStoreURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	// 创建文件
	file, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 复制内容
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return "", err
	}

	return localPath, nil
}

// 解析 DS_Store 文件 - 简化版本
func (d *DSStoreDumper) parseDSStoreFile(filePath string) ([]string, error) {
	if err := d.checkCancelled(); err != nil {
		return nil, err
	}

	d.log("解析 DS_Store 文件: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取整个文件
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var files []string

	// 方法1: 查找常见的文件名模式
	files = append(files, d.extractFilenamesFromContent(content)...)

	// 方法2: 查找以特定模式开头的字符串（DS_Store 常见模式）
	files = append(files, d.extractDSStoreFilenames(content)...)

	// 去重和过滤
	files = d.filterValidFilenames(files)

	d.log("成功解析出 %d 个文件", len(files))
	for _, file := range files {
		d.log("发现文件: %s", file)
	}

	return files, nil
}

// 从内容中提取文件名
func (d *DSStoreDumper) extractFilenamesFromContent(content []byte) []string {
	var files []string

	// 匹配看起来像文件名的字符串
	// 这个正则表达式匹配包含字母、数字、点、下划线、连字符的连续字符串
	re := regexp.MustCompile(`[a-zA-Z0-9._-]{1,100}`)
	matches := re.FindAll(content, -1)

	for _, match := range matches {
		filename := string(match)
		if d.isLikelyFilename(filename) {
			files = append(files, filename)
		}
	}

	return files
}

// 提取 DS_Store 特定的文件名
func (d *DSStoreDumper) extractDSStoreFilenames(content []byte) []string {
	var files []string

	// DS_Store 文件通常有特定的结构，我们可以尝试解析已知的模式
	// 这里我们查找以常见文件扩展名结尾的字符串
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".txt", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".html", ".htm", ".php", ".js", ".css"}

	for _, ext := range extensions {
		pattern := regexp.MustCompile(`([a-zA-Z0-9._-]{1,50}` + regexp.QuoteMeta(ext) + `)`)
		matches := pattern.FindAllSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				filename := string(match[1])
				files = append(files, filename)
			}
		}
	}

	return files
}

// 过滤有效的文件名
func (d *DSStoreDumper) filterValidFilenames(filenames []string) []string {
	var valid []string
	seen := make(map[string]bool)

	for _, filename := range filenames {
		// 去重
		if seen[filename] {
			continue
		}
		seen[filename] = true

		// 过滤无效文件名
		if d.isValidFilename(filename) {
			valid = append(valid, filename)
		}
	}

	return valid
}

// 检查是否为有效的文件名
func (d *DSStoreDumper) isValidFilename(name string) bool {
	if len(name) < 1 || len(name) > 255 {
		return false
	}

	// 排除明显不是文件名的字符串
	invalidNames := []string{
		".", "..", ".DS_Store", "DS_Store",
		"Bud1", "bwsp", "filp", "fwi0", "fwi1", "fwsw", "fwvh",
		"icnv", "icsb", "icsd", "icsp", "logS", "lg1S", "lg2S",
		"moDD", "ph1S", "ph2S", "vSrn", "vstl", "vstn",
	}

	for _, invalid := range invalidNames {
		if name == invalid {
			return false
		}
	}

	// 不能包含路径分隔符
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}

	// 不能全是点号
	if strings.Trim(name, ".") == "" {
		return false
	}

	return true
}

// 检查是否可能是文件名
func (d *DSStoreDumper) isLikelyFilename(name string) bool {
	if !d.isValidFilename(name) {
		return false
	}

	// 检查是否包含至少一个字母或数字
	hasAlphaNum := false
	for _, ch := range name {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			hasAlphaNum = true
			break
		}
	}

	if !hasAlphaNum {
		return false
	}

	// 如果包含扩展名，更可能是文件名
	if strings.Contains(name, ".") && len(name) > 3 {
		return true
	}

	// 长度适中的字符串更可能是文件名
	return len(name) >= 2 && len(name) <= 50
}

func (d *DSStoreDumper) downloadFile(baseURL, filePath, outputDir string) bool {
	if err := d.checkCancelled(); err != nil {
		return false
	}

	fileURL := fmt.Sprintf("%s/%s", baseURL, filePath)

	localPath := filepath.Join(outputDir, filePath)
	if _, err := os.Stat(localPath); err == nil {
		return true // 文件已存在
	}

	// 创建请求
	req, err := http.NewRequestWithContext(d.ctx, "GET", fileURL, nil)
	if err != nil {
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return false
	}

	file, err := os.Create(localPath)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return false
	}

	return true
}

func sanitizeDSStoreFilename(filename string) string {
	reg := regexp.MustCompile(`^https?://`)
	filename = reg.ReplaceAllString(filename, "")

	reg = regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	filename = reg.ReplaceAllString(filename, "_")

	if len(filename) > 100 {
		filename = filename[:100]
	}

	return filename
}

func (a *DSStoreDumperApp) GetDefaultOutputDir() (string, error) {
	defaultDir := filepath.Join(system.GetAppBaseDir(), "file", "dsstore-dumper-output")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		return "", err
	}
	return defaultDir, nil
}

func (a *DSStoreDumperApp) ParseURLsFromText(text string) []string {
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

func (a *DSStoreDumperApp) RetryScan(request RetryDSStoreScanRequest) (DSStoreScanResult, error) {
	if a.isScanning.Load() {
		return DSStoreScanResult{}, fmt.Errorf("扫描正在进行中，请先取消当前扫描")
	}

	isCancelled := func() bool { return false }
	return a.scanSingleURL(request.URL, request.OutputDir, request.Jobs, request.Timeout, request.Retry, isCancelled), nil
}

// CancelDsStoreScan 取消当前扫描
func (a *DSStoreDumperApp) CancelDsStoreScan() error {
	if a.cancelFunc != nil {
		a.cancelFunc()
		a.log("扫描取消请求已发送")
		return nil
	}
	return fmt.Errorf("没有正在进行的扫描")
}

// IsScanning 检查是否正在扫描
func (a *DSStoreDumperApp) IsScanning() bool {
	return a.isScanning.Load()
}

func (a *DSStoreDumperApp) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("[DSStoreDumperApp]", msg)

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "dsstore-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}
