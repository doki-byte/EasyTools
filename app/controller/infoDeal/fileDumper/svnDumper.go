package fileDumper

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"bufio"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type SvnDumperApp struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	isScanning bool
}

func NewSvnDumperApp() *SvnDumperApp {
	return &SvnDumperApp{}
}

func (a *SvnDumperApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

type SvnScanResult struct {
	URL       string `json:"url"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type SvnBatchScanRequest struct {
	URLs      []string `json:"urls"`
	OutputDir string   `json:"outputDir"`
	Jobs      int      `json:"jobs"`
	Timeout   int      `json:"timeout"`
	Retry     int      `json:"retry"`
}

type RetrySvnScanRequest struct {
	URL       string `json:"url"`
	OutputDir string `json:"outputDir"`
	Jobs      int    `json:"jobs"`
	Timeout   int    `json:"timeout"`
	Retry     int    `json:"retry"`
}

type SvnDumper struct {
	client       *http.Client
	outputDir    string
	jobs         int
	timeout      int
	retry        int
	progressChan chan<- string
	emitter      func(eventName string, data interface{})
	ctx          context.Context
	cancelFunc   context.CancelFunc
	baseURL      string
	scmURL       string
	regURL       string
}

// SVN 元数据文件列表
var svnFiles = []string{
	"all-wcprops",
	"entries",
	"format",
	"wc.db",
}

func NewSvnDumper(outputDir string, jobs, timeout, retry int, progressChan chan<- string, emitter func(eventName string, data interface{}), ctx context.Context) *SvnDumper {
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

	dumperCtx, cancelFunc := context.WithCancel(ctx)

	return &SvnDumper{
		client:       client,
		outputDir:    outputDir,
		jobs:         jobs,
		timeout:      timeout,
		retry:        retry,
		progressChan: progressChan,
		emitter:      emitter,
		ctx:          dumperCtx,
		cancelFunc:   cancelFunc,
	}
}

func (a *SvnDumperApp) BatchScan(request SvnBatchScanRequest) ([]SvnScanResult, error) {
	a.mu.Lock()
	if a.isScanning {
		a.mu.Unlock()
		return nil, fmt.Errorf("扫描正在进行中，请等待当前扫描完成")
	}
	a.isScanning = true
	ctx, cancelFunc := context.WithCancel(a.ctx)
	a.cancelFunc = cancelFunc
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isScanning = false
		a.cancelFunc = nil
		a.mu.Unlock()
	}()

	results := make([]SvnScanResult, len(request.URLs))

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
		select {
		case <-ctx.Done():
			a.log("扫描已被取消")
			return results, nil
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := a.scanSingleURL(ctx, url, request.OutputDir, request.Jobs, request.Timeout, request.Retry)

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, targetURL)
	}

	wg.Wait()
	return results, nil
}

func (a *SvnDumperApp) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "svn-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}

func (a *SvnDumperApp) scanSingleURL(ctx context.Context, targetURL, outputDir string, jobs, timeout, retry int) SvnScanResult {
	progressChan := make(chan string, 100)

	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	dumper := NewSvnDumper(outputDir, jobs, timeout, retry, progressChan, emitter, ctx)

	urlDir := sanitizeSvnFilename(targetURL)
	fullOutputDir := filepath.Join(outputDir, urlDir)

	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return SvnScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("Failed to create directory: %v", err),
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
		if ctx.Err() != nil {
			return SvnScanResult{
				URL:       targetURL,
				Success:   false,
				Message:   "扫描已被取消",
				Timestamp: time.Now().Format(time.RFC3339),
			}
		}
		return SvnScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("Dump failed: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return SvnScanResult{
		URL:       targetURL,
		Success:   true,
		Message:   "Successfully dumped SVN repository",
		OutputDir: fullOutputDir,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (s *SvnDumper) log(format string, args ...interface{}) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	msg := fmt.Sprintf(format, args...)
	if s.progressChan != nil {
		select {
		case s.progressChan <- msg:
		default:
		}
	}
	if s.emitter != nil {
		s.emitter("svn-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		})
		s.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/SVN目录遍历",
		})
	}
}

func (s *SvnDumper) logSuccess(format string, args ...interface{}) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	msg := fmt.Sprintf(format, args...)
	if s.progressChan != nil {
		select {
		case s.progressChan <- msg:
		default:
		}
	}
	if s.emitter != nil {
		s.emitter("svn-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "success",
			"time":    time.Now().Format(time.RFC3339),
		})
		s.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/SVN目录遍历",
		})
	}
}

func (s *SvnDumper) logError(format string, args ...interface{}) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	msg := fmt.Sprintf(format, args...)
	if s.progressChan != nil {
		select {
		case s.progressChan <- msg:
		default:
		}
	}
	if s.emitter != nil {
		s.emitter("svn-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format(time.RFC3339),
		})
		s.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/SVN目录遍历",
		})
	}
}

func (s *SvnDumper) normalizeURL(targetURL string) {
	targetURL = strings.TrimSuffix(targetURL, "/")

	if strings.Contains(targetURL, "/.svn") {
		s.scmURL = targetURL
		s.regURL = strings.Replace(targetURL, "/.svn", "", 1)
	} else {
		s.scmURL = targetURL + "/.svn"
		s.regURL = targetURL
	}
	s.baseURL = targetURL
}

func (s *SvnDumper) Dump(targetURL, outputDir string) error {
	s.log("Starting SVN dump for: %s", targetURL)
	s.normalizeURL(targetURL)

	// 创建 SVN 目录结构
	s.createSvnDirs(outputDir)

	// 下载 SVN 元数据文件
	s.downloadSvnFiles(outputDir)

	// 检查 SVN 格式并处理
	wcDbPath := filepath.Join(outputDir, ".svn", "wc.db")
	entriesPath := filepath.Join(outputDir, ".svn", "entries")

	if s.fileExists(wcDbPath) {
		s.logSuccess("Found new SVN client storage format!")
		return s.processNewFormat(outputDir)
	} else if s.fileExists(entriesPath) {
		s.logSuccess("Found old SVN client storage format!")
		return s.processOldFormat(outputDir)
	} else {
		return fmt.Errorf("could not identify SVN format")
	}
}

func (s *SvnDumper) createSvnDirs(outputDir string) {
	dirs := []string{
		".svn",
		".svn/text-base",
		".svn/pristine",
		".svn/tmp",
		".svn/props",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(outputDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			s.logError("Failed to create directory %s: %v", dir, err)
		}
	}
}

func (s *SvnDumper) processNewFormat(outputDir string) error {
	dbPath := filepath.Join(outputDir, ".svn", "wc.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// 查询 repository 信息
	rows, err := db.Query("SELECT id, root, uuid FROM repository")
	if err != nil {
		s.logError("Failed to query repository: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var root, uuid string
			if err := rows.Scan(&id, &root, &uuid); err != nil {
				s.logError("Failed to scan repository row: %v", err)
				continue
			}
			s.log("REP INFO => %d:%s:%s", id, root, uuid)
		}
	}

	// 查询 NODES 表获取文件列表并恢复文件
	nodesRows, err := db.Query(`
		SELECT local_relpath, checksum, contents_checksum 
		FROM nodes 
		WHERE op_depth = 0 AND kind = 'file'
	`)
	if err != nil {
		s.logError("Failed to query nodes: %v", err)
	} else {
		defer nodesRows.Close()

		fileCount := 0
		for nodesRows.Next() {
			select {
			case <-s.ctx.Done():
				return fmt.Errorf("processing cancelled")
			default:
			}

			var localRelpath string
			var checksum, contentsChecksum sql.NullString

			if err := nodesRows.Scan(&localRelpath, &checksum, &contentsChecksum); err != nil {
				s.logError("Failed to scan nodes row: %v", err)
				continue
			}

			// 使用 contents_checksum 或 checksum 来恢复文件
			fileHash := ""
			if contentsChecksum.Valid && strings.HasPrefix(contentsChecksum.String, "$sha1$") {
				fileHash = strings.TrimPrefix(contentsChecksum.String, "$sha1$")
			} else if checksum.Valid && strings.HasPrefix(checksum.String, "$sha1$") {
				fileHash = strings.TrimPrefix(checksum.String, "$sha1$")
			}

			if fileHash != "" {
				if s.restoreFileFromPristine(outputDir, localRelpath, fileHash) {
					s.logSuccess("Restored: %s", localRelpath)
					fileCount++
				} else {
					s.logError("Failed to restore: %s", localRelpath)
				}
			} else {
				s.log("No valid checksum for: %s", localRelpath)
			}
		}
		s.logSuccess("Restored %d files from pristine", fileCount)
	}

	// 查询 pristine 表获取其他文件对象
	pristineRows, err := db.Query("SELECT checksum FROM pristine")
	if err != nil {
		s.logError("Failed to query pristine: %v", err)
	} else {
		defer pristineRows.Close()

		objectCount := 0
		for pristineRows.Next() {
			select {
			case <-s.ctx.Done():
				return fmt.Errorf("processing cancelled")
			default:
			}

			var checksum string

			if err := pristineRows.Scan(&checksum); err != nil {
				s.logError("Failed to scan pristine row: %v", err)
				continue
			}

			if strings.HasPrefix(checksum, "$sha1$") {
				fileHash := strings.TrimPrefix(checksum, "$sha1$")
				s.getObject(outputDir, fileHash)
				objectCount++
			} else if checksum != "" {
				s.logError("Unknown checksum format: %s", checksum)
			}
		}
		s.logSuccess("Downloaded %d pristine objects", objectCount)
	}

	return s.checkout(outputDir)
}

func (s *SvnDumper) processOldFormat(outputDir string) error {
	return s.svnEntries("", outputDir)
}

func (s *SvnDumper) svnEntries(urlPath, outputDir string) error {
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("processing cancelled")
	default:
	}

	s.createSvnDirs(outputDir)
	s.downloadSvnFilePaths(urlPath, outputDir)

	entriesPath := filepath.Join(outputDir, ".svn", "entries")
	content, err := os.ReadFile(entriesPath)
	if err != nil {
		return fmt.Errorf("failed to read entries file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var prevLine string

	for _, line := range lines {
		select {
		case <-s.ctx.Done():
			return fmt.Errorf("processing cancelled")
		default:
		}

		line = strings.TrimSpace(line)
		if line == "dir" && prevLine != "" {
			newDir := prevLine
			fullPath := filepath.Join(outputDir, newDir)

			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				if err := os.MkdirAll(fullPath, 0755); err != nil {
					s.logError("Failed to create directory %s: %v", newDir, err)
					continue
				}
				s.svnEntries(urlPath+"/"+newDir, fullPath)
			}
		}

		if line == "file" && prevLine != "" {
			newFile := prevLine
			fileURL := urlPath + "/.svn/text-base/" + newFile + ".svn-base"
			localPath := filepath.Join(outputDir, ".svn", "text-base", newFile+".svn-base")
			if s.downloadFileWithRetry(fileURL, localPath) {
				// 恢复文件到工作副本
				workingCopyPath := filepath.Join(outputDir, newFile)
				s.copyFile(localPath, workingCopyPath)
			}
		}

		prevLine = line
	}

	return s.checkout(outputDir)
}

// 从 pristine 恢复文件到工作副本
func (s *SvnDumper) restoreFileFromPristine(outputDir, filePath, fileHash string) bool {
	if len(fileHash) < 2 {
		return false
	}

	dirName := fileHash[:2]
	fileName := fileHash + ".svn-base"

	pristinePath := filepath.Join(outputDir, ".svn", "pristine", dirName, fileName)
	workingCopyPath := filepath.Join(outputDir, filePath)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(workingCopyPath), 0755); err != nil {
		s.logError("Failed to create directory for %s: %v", filePath, err)
		return false
	}

	// 复制文件
	return s.copyFile(pristinePath, workingCopyPath)
}

// 复制文件
func (s *SvnDumper) copyFile(src, dst string) bool {
	srcFile, err := os.Open(src)
	if err != nil {
		return false
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return false
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err == nil
}

func (s *SvnDumper) getObject(outputDir, fileHash string) {
	if len(fileHash) < 2 {
		return
	}

	dirName := fileHash[:2]
	fileName := fileHash + ".svn-base"

	pristineDir := filepath.Join(outputDir, ".svn", "pristine", dirName)
	if err := os.MkdirAll(pristineDir, 0755); err != nil {
		s.logError("Failed to create pristine directory: %v", err)
		return
	}

	fileURL := s.scmURL + "/pristine/" + dirName + "/" + fileName
	localPath := filepath.Join(pristineDir, fileName)

	s.downloadFileWithRetry(fileURL, localPath)
}

func (s *SvnDumper) checkout(outputDir string) error {
	s.log("Trying to revert the tree, if you get error, upgrade your SVN client!")

	// 尝试使用系统 svn 命令进行恢复
	if s.trySystemSvnRevert(outputDir) {
		s.logSuccess("SVN revert completed successfully")
	} else {
		s.log("SVN revert not available, using internal file restoration")
		// 如果系统 svn 不可用，使用内部恢复机制
		s.restoreFilesManually(outputDir)
	}

	return nil
}

// 尝试使用系统 svn revert 命令
func (s *SvnDumper) trySystemSvnRevert(outputDir string) bool {
	// 检查是否在输出目录中
	originalDir, err := os.Getwd()
	if err != nil {
		return false
	}
	defer os.Chdir(originalDir)

	// 切换到输出目录
	if err := os.Chdir(outputDir); err != nil {
		return false
	}

	// 执行 svn revert 命令
	cmd := exec.Command("svn", "revert", "-R", ".")
	output, err := cmd.CombinedOutput()

	if err != nil {
		s.log("System SVN revert failed: %v", err)
		s.log("SVN output: %s", string(output))
		return false
	}

	s.log("SVN revert output: %s", string(output))
	return true
}

// 手动恢复文件
func (s *SvnDumper) restoreFilesManually(outputDir string) {
	// 检查 wc.db 是否存在
	wcDbPath := filepath.Join(outputDir, ".svn", "wc.db")
	if !s.fileExists(wcDbPath) {
		return
	}

	db, err := sql.Open("sqlite3", wcDbPath)
	if err != nil {
		s.logError("Failed to open database for manual restore: %v", err)
		return
	}
	defer db.Close()

	// 查询所有文件并恢复
	rows, err := db.Query(`
		SELECT local_relpath, contents_checksum 
		FROM nodes 
		WHERE op_depth = 0 AND kind = 'file'
	`)
	if err != nil {
		s.logError("Failed to query nodes for manual restore: %v", err)
		return
	}
	defer rows.Close()

	restoredCount := 0
	for rows.Next() {
		var localRelpath string
		var contentsChecksum sql.NullString

		if err := rows.Scan(&localRelpath, &contentsChecksum); err != nil {
			s.logError("Failed to scan nodes row for manual restore: %v", err)
			continue
		}

		if contentsChecksum.Valid && strings.HasPrefix(contentsChecksum.String, "$sha1$") {
			fileHash := strings.TrimPrefix(contentsChecksum.String, "$sha1$")
			if s.restoreFileFromPristine(outputDir, localRelpath, fileHash) {
				s.logSuccess("Restored: %s", localRelpath)
				restoredCount++
			}
		}
	}

	s.logSuccess("Manually restored %d files", restoredCount)
}

// 下载根目录的SVN文件
func (s *SvnDumper) downloadSvnFiles(outputDir string) {
	for _, file := range svnFiles {
		select {
		case <-s.ctx.Done():
			s.log("下载被取消")
			return
		default:
		}

		fileURL := s.scmURL + "/" + file
		localPath := filepath.Join(outputDir, ".svn", file)

		if s.downloadFileWithRetry(fileURL, localPath) {
			s.logSuccess("Downloaded: %s", file)
		} else {
			s.logError("Failed to download: %s", file)
		}
	}
}

// 下载子目录的SVN文件，urlPath是子目录的路径
func (s *SvnDumper) downloadSvnFilePaths(urlPath, outputDir string) {
	for _, file := range svnFiles {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		fileURL := s.regURL + urlPath + "/.svn/" + file
		localPath := filepath.Join(outputDir, ".svn", file)
		s.downloadFileWithRetry(fileURL, localPath)
	}
}

func (s *SvnDumper) downloadFileWithRetry(fileURL, localPath string) bool {
	for attempt := 0; attempt < s.retry; attempt++ {
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		if s.downloadFile(fileURL, localPath) {
			return true
		}

		if attempt < s.retry-1 {
			s.log("Retry %d/%d for: %s", attempt+1, s.retry, fileURL)
			time.Sleep(2 * time.Second)
		}
	}
	return false
}

func (s *SvnDumper) downloadFile(fileURL, localPath string) bool {
	select {
	case <-s.ctx.Done():
		return false
	default:
	}

	if s.fileExists(localPath) {
		return true
	}

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return false
	}

	req = req.WithContext(s.ctx)
	resp, err := s.client.Do(req)
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

func (s *SvnDumper) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sanitizeSvnFilename(filename string) string {
	// 移除协议头
	reg := regexp.MustCompile(`^https?://`)
	filename = reg.ReplaceAllString(filename, "")

	// 替换非法字符
	reg = regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	filename = reg.ReplaceAllString(filename, "_")

	// 限制长度
	if len(filename) > 100 {
		filename = filename[:100]
	}

	return filename
}

func (a *SvnDumperApp) GetDefaultOutputDir() (string, error) {
	defaultDir := filepath.Join(system.GetAppBaseDir(), "file", "svn-dumper-output")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		return "", err
	}
	return defaultDir, nil
}

func (a *SvnDumperApp) ParseURLsFromText(text string) []string {
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

func (a *SvnDumperApp) RetryScan(request RetrySvnScanRequest) (SvnScanResult, error) {
	ctx := context.Background() // 重试使用新的上下文
	return a.scanSingleURL(ctx, request.URL, request.OutputDir, request.Jobs, request.Timeout, request.Retry), nil
}

func (a *SvnDumperApp) CancelSvnScan() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isScanning && a.cancelFunc != nil {
		a.cancelFunc()
		a.isScanning = false
		a.log("扫描已取消")
	}
}
