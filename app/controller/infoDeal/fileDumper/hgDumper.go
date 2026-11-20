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
)

type HgDumperApp struct {
	ctx context.Context
	// 用于存储取消函数
	cancelFuncs sync.Map
}

func NewHgDumperApp() *HgDumperApp {
	return &HgDumperApp{}
}

func (a *HgDumperApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

type HgScanResult struct {
	URL       string `json:"url"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type HgBatchScanRequest struct {
	URLs      []string `json:"urls"`
	OutputDir string   `json:"outputDir"`
	Jobs      int      `json:"jobs"`
	Timeout   int      `json:"timeout"`
	Retry     int      `json:"retry"`
	TaskID    string   `json:"taskId,omitempty"` // 用于取消任务
}

type HgRetryScanRequest struct {
	URL       string `json:"url"`
	OutputDir string `json:"outputDir"`
	Jobs      int    `json:"jobs"`
	Timeout   int    `json:"timeout"`
	Retry     int    `json:"retry"`
	TaskID    string `json:"taskId,omitempty"`
}

type HgDumper struct {
	client       *http.Client
	outputDir    string
	jobs         int
	timeout      int
	retry        int
	progressChan chan string // 改为非缓冲通道，由dumper自己管理
	emitter      func(eventName string, data interface{})
	ctx          context.Context    // 添加context用于取消
	cancel       context.CancelFunc // 取消函数
	isCancelled  *atomic.Bool       // 原子操作标志
	wg           sync.WaitGroup     // 等待组，用于等待日志处理完成
}

func NewHgDumper(outputDir string, jobs, timeout, retry int, emitter func(eventName string, data interface{})) *HgDumper {
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

	// 创建context和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	isCancelled := &atomic.Bool{}
	isCancelled.Store(false)

	dumper := &HgDumper{
		client:       client,
		outputDir:    outputDir,
		jobs:         jobs,
		timeout:      timeout,
		retry:        retry,
		progressChan: make(chan string, 100), // 使用缓冲通道
		emitter:      emitter,
		ctx:          ctx,
		cancel:       cancel,
		isCancelled:  isCancelled,
	}

	// 启动日志处理goroutine
	dumper.wg.Add(1)
	go dumper.processLogs()

	return dumper
}

// 处理日志的goroutine
func (h *HgDumper) processLogs() {
	defer h.wg.Done()

	for {
		select {
		case msg, ok := <-h.progressChan:
			if !ok {
				// 通道关闭，退出
				return
			}
			if h.emitter != nil {
				h.emitter("hg-dumper-log", map[string]interface{}{
					"message": msg,
					"level":   "info",
					"time":    time.Now().Format(time.RFC3339),
				})
			}
		case <-h.ctx.Done():
			// 上下文被取消，退出
			return
		}
	}
}

// 安全关闭dumper
func (h *HgDumper) Close() {
	h.Cancel()
	close(h.progressChan)
	h.wg.Wait() // 等待日志处理完成
}

// 检查是否已取消
func (h *HgDumper) isCanceled() bool {
	return h.isCancelled.Load()
}

// 取消操作
func (h *HgDumper) Cancel() {
	if h.isCancelled.CompareAndSwap(false, true) {
		h.cancel()
		h.log("操作已被用户取消")
	}
}

func (a *HgDumperApp) BatchScan(request HgBatchScanRequest) ([]HgScanResult, error) {
	// 生成任务ID
	if request.TaskID == "" {
		request.TaskID = fmt.Sprintf("batch-%d", time.Now().UnixNano())
	}

	results := make([]HgScanResult, len(request.URLs))

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

	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	// 创建dumper实例（用于取消功能）
	dumper := NewHgDumper(request.OutputDir, request.Jobs, request.Timeout, request.Retry, emitter)
	defer dumper.Close() // 确保dumper被正确关闭

	// 存储取消函数
	a.cancelFuncs.Store(request.TaskID, dumper.Cancel)

	// 记录任务开始
	dumper.log("开始批量扫描，任务ID: %s, URL数量: %d", request.TaskID, len(request.URLs))

	for i, targetURL := range request.URLs {
		// 检查是否已取消
		if dumper.isCanceled() {
			dumper.log("批量扫描已被取消")
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			// 检查是否已取消
			if dumper.isCanceled() {
				dumper.log("跳过URL（已取消）: %s", url)
				return
			}

			result := a.scanSingleURL(url, request.OutputDir, request.Jobs, request.Timeout, request.Retry, dumper)

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			// 发送进度更新
			completed := 0
			mu.Lock()
			for _, r := range results {
				if r.URL != "" {
					completed++
				}
			}
			mu.Unlock()

			progress := float64(completed) / float64(len(request.URLs)) * 100
			dumper.log("批量扫描进度: %.1f%% (%d/%d)", progress, completed, len(request.URLs))
		}(i, targetURL)
	}

	wg.Wait()

	// 清理取消函数
	a.cancelFuncs.Delete(request.TaskID)

	// 检查是否被取消
	if dumper.isCanceled() {
		return nil, fmt.Errorf("扫描被用户取消")
	}

	dumper.logSuccess("批量扫描完成")
	return results, nil
}

func (a *HgDumperApp) scanSingleURL(targetURL, outputDir string, jobs, timeout, retry int, dumper *HgDumper) HgScanResult {
	// 检查是否已取消
	if dumper.isCanceled() {
		return HgScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   "扫描被用户取消",
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	dumper.log("开始扫描URL: %s", targetURL)

	urlDir := sanitizeHgFilename(targetURL)
	fullOutputDir := filepath.Join(outputDir, urlDir)

	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return HgScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("创建目录失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	err := dumper.Dump(targetURL, fullOutputDir)
	if err != nil {
		if dumper.isCanceled() {
			return HgScanResult{
				URL:       targetURL,
				Success:   false,
				Message:   "扫描被用户取消",
				Timestamp: time.Now().Format(time.RFC3339),
			}
		}
		return HgScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("下载失败: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return HgScanResult{
		URL:       targetURL,
		Success:   true,
		Message:   "成功下载HG仓库",
		OutputDir: fullOutputDir,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// 取消扫描任务
func (a *HgDumperApp) CancelHgScan(taskID string) error {
	if cancelFunc, ok := a.cancelFuncs.Load(taskID); ok {
		if cancel, ok := cancelFunc.(func()); ok {
			cancel()
			a.cancelFuncs.Delete(taskID)
			return nil
		}
	}
	return fmt.Errorf("未找到任务ID: %s", taskID)
}

func (h *HgDumper) log(format string, args ...interface{}) {
	if h.isCanceled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	select {
	case h.progressChan <- msg:
		// 成功发送
	case <-h.ctx.Done():
		// 上下文被取消，不发送
	default:
		// 通道满，跳过这条日志
	}
}

func (h *HgDumper) logSuccess(format string, args ...interface{}) {
	if h.isCanceled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	select {
	case h.progressChan <- msg:
		// 成功发送
	case <-h.ctx.Done():
		// 上下文被取消，不发送
	default:
		// 通道满，跳过这条日志
	}
}

func (h *HgDumper) logError(format string, args ...interface{}) {
	if h.isCanceled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	select {
	case h.progressChan <- msg:
		// 成功发送
	case <-h.ctx.Done():
		// 上下文被取消，不发送
	default:
		// 通道满，跳过这条日志
	}
}

func (h *HgDumper) Dump(targetURL, outputDir string) error {
	// 检查是否已取消
	if h.isCanceled() {
		return fmt.Errorf("操作已被取消")
	}

	h.log("开始HG下载: %s", targetURL)

	targetURL = strings.TrimSuffix(targetURL, "/")
	if strings.HasSuffix(targetURL, "/.hg") {
		targetURL = strings.TrimSuffix(targetURL, "/.hg")
	}

	// 检查HG目录是否存在
	hgDirURL := fmt.Sprintf("%s/.hg/", targetURL)
	h.log("检测HG目录: %s", hgDirURL)

	// 使用context创建请求
	req, err := http.NewRequestWithContext(h.ctx, "GET", hgDirURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("获取HG目录失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HG目录返回状态 %d", resp.StatusCode)
	}

	h.logSuccess("发现HG目录，开始递归下载")

	// 递归下载HG目录
	return h.recursiveDownloadHG(targetURL, outputDir, ".hg")
}

func (h *HgDumper) recursiveDownloadHG(baseURL, outputDir, currentPath string) error {
	// 检查是否已取消
	if h.isCanceled() {
		return fmt.Errorf("操作已被取消")
	}

	h.log("扫描HG目录: %s", currentPath)

	dirURL := fmt.Sprintf("%s/%s", baseURL, currentPath)

	req, err := http.NewRequestWithContext(h.ctx, "GET", dirURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("获取目录 %s 失败: %v", currentPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("目录 %s 返回状态 %d", currentPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取目录内容失败: %v", err)
	}

	// 解析目录列表
	files := h.extractFilesFromDirectoryListing(string(body), currentPath)

	for _, file := range files {
		// 检查是否已取消
		if h.isCanceled() {
			return fmt.Errorf("操作已被取消")
		}

		fullPath := filepath.Join(currentPath, file)

		if strings.HasSuffix(file, "/") {
			// 递归处理子目录
			if err := h.recursiveDownloadHG(baseURL, outputDir, fullPath); err != nil {
				h.logError("下载目录 %s 失败: %v", fullPath, err)
			}
		} else {
			// 下载文件
			if h.downloadFile(baseURL, fullPath, outputDir) {
				h.logSuccess("已下载: %s", fullPath)
			} else {
				h.logError("下载失败: %s", fullPath)
			}
		}
	}

	// 检查是否已取消
	if h.isCanceled() {
		return fmt.Errorf("操作已被取消")
	}

	// 下载HG特定文件
	hgFiles := []string{
		"00changelog.i",
		"branch",
		"branchheads.cache",
		"dirstate",
		"hgrc",
		"last-message.txt",
		"requires",
		"store/",
		"undo.backup.dirstate",
		"undo.dirstate",
		"undo.branch",
		"undo.bookmarks",
		"undo.desc",
		"undo.phaseroots",
	}

	for _, file := range hgFiles {
		// 检查是否已取消
		if h.isCanceled() {
			return fmt.Errorf("操作已被取消")
		}

		hgFilePath := filepath.Join(currentPath, file)
		if strings.HasSuffix(file, "/") {
			h.recursiveDownloadHG(baseURL, outputDir, hgFilePath)
		} else {
			h.downloadFile(baseURL, hgFilePath, outputDir)
		}
	}

	// 检查是否已取消
	if h.isCanceled() {
		return fmt.Errorf("操作已被取消")
	}

	// 下载store目录内容
	h.downloadStoreContents(baseURL, outputDir, filepath.Join(currentPath, "store"))

	return nil
}

func (h *HgDumper) downloadStoreContents(baseURL, outputDir, storePath string) {
	// 检查是否已取消
	if h.isCanceled() {
		return
	}

	h.log("下载HG存储内容")

	storeFiles := []string{
		"00changelog.i",
		"00manifest.i",
		"bookmarks",
		"bookmarks.current",
		"fncache",
		"phaseroots",
		"undo",
		"undo.phaseroots",
		"data/",
		"meta/",
	}

	for _, file := range storeFiles {
		// 检查是否已取消
		if h.isCanceled() {
			return
		}

		fullPath := filepath.Join(storePath, file)
		if strings.HasSuffix(file, "/") {
			h.recursiveDownloadHG(baseURL, outputDir, fullPath)
		} else {
			h.downloadFile(baseURL, fullPath, outputDir)
		}
	}

	// 下载data目录中的文件
	h.downloadDataFiles(baseURL, outputDir, filepath.Join(storePath, "data"))
}

func (h *HgDumper) downloadDataFiles(baseURL, outputDir, dataPath string) {
	// 检查是否已取消
	if h.isCanceled() {
		return
	}

	h.log("下载HG数据文件")

	// 尝试下载一些常见的数据文件
	dataFiles := []string{
		"00manifest.d",
		"00changelog.d",
	}

	for _, file := range dataFiles {
		// 检查是否已取消
		if h.isCanceled() {
			return
		}

		fullPath := filepath.Join(dataPath, file)
		h.downloadFile(baseURL, fullPath, outputDir)
	}

	// 递归下载data目录下的所有子目录
	dirURL := fmt.Sprintf("%s/%s", baseURL, dataPath)

	req, err := http.NewRequestWithContext(h.ctx, "GET", dirURL, nil)
	if err != nil {
		h.logError("创建请求失败: %v", err)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logError("获取数据目录失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logError("读取数据目录内容失败: %v", err)
			return
		}

		files := h.extractFilesFromDirectoryListing(string(body), dataPath)
		for _, file := range files {
			// 检查是否已取消
			if h.isCanceled() {
				return
			}

			fullPath := filepath.Join(dataPath, file)
			if strings.HasSuffix(file, "/") {
				h.recursiveDownloadHG(baseURL, outputDir, fullPath)
			} else {
				h.downloadFile(baseURL, fullPath, outputDir)
			}
		}
	}
}

func (h *HgDumper) extractFilesFromDirectoryListing(htmlContent, currentPath string) []string {
	var files []string

	// 匹配目录列表中的文件和文件夹
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<a\s+href="([^"]+)"`),
		regexp.MustCompile(`href="([^"]+)"`),
		regexp.MustCompile(`>([^<]+)</a>`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) > 1 {
				filename := match[1]
				// 过滤无效文件
				if filename != "" && filename != "../" && filename != "./" &&
					!strings.HasPrefix(filename, "?") && !strings.HasPrefix(filename, "/") {

					// 检查是否是目录
					if strings.HasSuffix(filename, "/") {
						files = append(files, filename)
					} else {
						files = append(files, filename)
					}
				}
			}
		}
	}

	return files
}

func (h *HgDumper) downloadFile(baseURL, filePath, outputDir string) bool {
	// 检查是否已取消
	if h.isCanceled() {
		return false
	}

	fileURL := fmt.Sprintf("%s/%s", baseURL, filePath)

	localPath := filepath.Join(outputDir, filePath)
	if _, err := os.Stat(localPath); err == nil {
		return true
	}

	req, err := http.NewRequestWithContext(h.ctx, "GET", fileURL, nil)
	if err != nil {
		return false
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

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

func sanitizeHgFilename(filename string) string {
	reg := regexp.MustCompile(`^https?://`)
	filename = reg.ReplaceAllString(filename, "")

	reg = regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	filename = reg.ReplaceAllString(filename, "_")

	if len(filename) > 100 {
		filename = filename[:100]
	}

	return filename
}

func (a *HgDumperApp) GetDefaultOutputDir() (string, error) {
	defaultDir := filepath.Join(system.GetAppBaseDir(), "file", "hg-dumper-output")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		return "", err
	}
	return defaultDir, nil
}

func (a *HgDumperApp) ParseURLsFromText(text string) []string {
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

func (a *HgDumperApp) RetryScan(request HgRetryScanRequest) (HgScanResult, error) {
	// 生成任务ID
	if request.TaskID == "" {
		request.TaskID = fmt.Sprintf("retry-%d", time.Now().UnixNano())
	}

	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	// 创建dumper实例
	dumper := NewHgDumper(request.OutputDir, request.Jobs, request.Timeout, request.Retry, emitter)
	defer dumper.Close()

	// 存储取消函数
	a.cancelFuncs.Store(request.TaskID, dumper.Cancel)

	result := a.scanSingleURL(request.URL, request.OutputDir, request.Jobs, request.Timeout, request.Retry, dumper)

	// 清理取消函数
	a.cancelFuncs.Delete(request.TaskID)

	return result, nil
}
