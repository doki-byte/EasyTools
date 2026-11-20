package fileDumper

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
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

	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
)

type GitDumperApp struct {
	ctx context.Context
}

func NewGitDumperApp() *GitDumperApp {
	return &GitDumperApp{}
}

func (a *GitDumperApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

type GitScanResult struct {
	URL       string `json:"url"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type GitBatchScanRequest struct {
	URLs        []string `json:"urls"`
	OutputDir   string   `json:"outputDir"`
	Jobs        int      `json:"jobs"`
	Timeout     int      `json:"timeout"`
	Retry       int      `json:"retry"`
	CancelToken string   `json:"cancelToken,omitempty"`
}

type RetryGitScanRequest struct {
	URL       string `json:"url"`
	OutputDir string `json:"outputDir"`
	Jobs      int    `json:"jobs"`
	Timeout   int    `json:"timeout"`
	Retry     int    `json:"retry"`
}

type GitDumper struct {
	client       *http.Client
	outputDir    string
	jobs         int
	timeout      int
	retry        int
	progressChan chan<- string
	emitter      func(eventName string, data interface{})
	cancelToken  string
	ctx          context.Context
	cancel       context.CancelFunc
	visitedURLs  sync.Map
}

// 全局取消令牌管理
var (
	cancelGitTokens = make(map[string]context.CancelFunc)
	cancelGitMutex  sync.Mutex
)

func NewGitDumper(outputDir string, jobs, timeout, retry int, progressChan chan<- string, emitter func(eventName string, data interface{}), cancelToken string) *GitDumper {
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
		cancelGitMutex.Lock()
		cancelGitTokens[cancelToken] = cancel
		cancelGitMutex.Unlock()
	}

	return &GitDumper{
		client:       client,
		outputDir:    outputDir,
		jobs:         jobs,
		timeout:      timeout,
		retry:        retry,
		progressChan: progressChan,
		emitter:      emitter,
		cancelToken:  cancelToken,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (i *GitDumper) cleanup() {
	if i.cancelToken != "" {
		cancelGitMutex.Lock()
		delete(cancelGitTokens, i.cancelToken)
		cancelGitMutex.Unlock()
	}
}

func (a *GitDumperApp) CancelGitScan(cancelToken string) error {
	cancelGitMutex.Lock()
	defer cancelGitMutex.Unlock()

	if cancel, exists := cancelGitTokens[cancelToken]; exists {
		cancel()
		delete(cancelGitTokens, cancelToken)
		return nil
	}
	return fmt.Errorf("未找到对应的扫描任务")
}

func (a *GitDumperApp) BatchScan(request GitBatchScanRequest) ([]GitScanResult, error) {
	results := make([]GitScanResult, len(request.URLs))

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
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := a.scanSingleURL(url, request.OutputDir, request.Jobs, request.Timeout, request.Retry, request.CancelToken)

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, targetURL)
	}

	wg.Wait()
	return results, nil
}

func (a *GitDumperApp) scanSingleURL(targetURL, outputDir string, jobs, timeout, retry int, cancelToken string) GitScanResult {
	progressChan := make(chan string, 100)

	// 创建事件发射器
	emitter := func(eventName string, data interface{}) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	}

	dumper := NewGitDumper(outputDir, jobs, timeout, retry, progressChan, emitter, cancelToken)

	urlDir := sanitizeFilename(targetURL)
	fullOutputDir := filepath.Join(outputDir, urlDir)

	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return GitScanResult{
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
		return GitScanResult{
			URL:       targetURL,
			Success:   false,
			Message:   fmt.Sprintf("Dump failed: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return GitScanResult{
		URL:       targetURL,
		Success:   true,
		Message:   "Successfully dumped git repository",
		OutputDir: fullOutputDir,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (g *GitDumper) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if g.progressChan != nil {
		g.progressChan <- msg
	}
	// 发送模块日志事件到前端，同时同步到全局日志总线
	if g.emitter != nil {
		payload := map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format(time.RFC3339),
		}
		g.emitter("git-dumper-log", payload)
		g.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/Git目录遍历",
		})
	}
}

// 添加不同级别的日志方法
func (g *GitDumper) logSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if g.progressChan != nil {
		g.progressChan <- msg
	}
	if g.emitter != nil {
		g.emitter("git-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "success",
			"time":    time.Now().Format(time.RFC3339),
		})
		g.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "info",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/Git目录遍历",
		})
	}
}

func (g *GitDumper) logWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if g.progressChan != nil {
		g.progressChan <- msg
	}
	if g.emitter != nil {
		g.emitter("git-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "warning",
			"time":    time.Now().Format(time.RFC3339),
		})
		g.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "warning",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/Git目录遍历",
		})
	}
}

func (g *GitDumper) logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if g.progressChan != nil {
		g.progressChan <- msg
	}
	if g.emitter != nil {
		g.emitter("git-dumper-log", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format(time.RFC3339),
		})
		g.emitter("log_event", map[string]interface{}{
			"message": msg,
			"level":   "error",
			"time":    time.Now().Format("15:04:05"),
			"source":  "信息处理/Git目录遍历",
		})
	}
}

func (g *GitDumper) Dump(targetURL, outputDir string) error {
	defer g.cleanup()

	g.log("Starting git dump for: %s", targetURL)

	// 规范化URL
	targetURL = strings.TrimSuffix(targetURL, "/")
	if strings.HasSuffix(targetURL, "/.git") {
		targetURL = strings.TrimSuffix(targetURL, "/.git")
	}
	targetURL = strings.TrimSuffix(targetURL, "/")

	// 检查HEAD文件
	headURL := fmt.Sprintf("%s/.git/HEAD", targetURL)
	g.log("Testing %s/.git/HEAD", targetURL)
	resp, err := g.client.Get(headURL)
	if err != nil {
		return errors.Wrap(err, "failed to fetch HEAD")
	}
	defer resp.Body.Close()

	g.log(" [%d]\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		return fmt.Errorf("HEAD returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read HEAD response")
	}

	headContent := strings.TrimSpace(string(body))
	if !isValidHEAD(headContent) {
		return fmt.Errorf("invalid HEAD content: %s", headContent)
	}

	g.log("HEAD is valid: %s", headContent)

	// 创建必要的git目录结构
	if err := g.createGitDirectoryStructure(outputDir); err != nil {
		return errors.Wrap(err, "failed to create git directory structure")
	}

	// 下载文件
	g.logWarning("Using comprehensive download")
	g.comprehensiveDownload(targetURL, outputDir)

	return g.restoreGitRepo(outputDir)
}

// 创建必要的git目录结构
func (g *GitDumper) createGitDirectoryStructure(outputDir string) error {
	dirs := []string{
		".git/objects",
		".git/objects/info",
		".git/objects/pack",
		".git/refs",
		".git/refs/heads",
		".git/refs/tags",
		".git/refs/remotes",
		".git/refs/remotes/origin",
		".git/info",
		".git/hooks",
		".git/logs",
		".git/logs/refs",
		".git/logs/refs/heads",
		".git/logs/refs/remotes",
		".git/logs/refs/remotes/origin",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(outputDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", dir)
		}
	}
	g.logSuccess("Created git directory structure")
	return nil
}

// 递归下载（基于目录列表）
func (g *GitDumper) recursiveDownload(targetURL, outputDir string) {
	g.log("Starting recursive download")
	g.downloadRecursiveItem(targetURL, ".git/", outputDir)
	g.downloadFile(targetURL, ".gitignore", outputDir)
}

// 递归下载项目
func (g *GitDumper) downloadRecursiveItem(baseURL, itemPath, outputDir string) {
	localPath := filepath.Join(outputDir, itemPath)

	// 如果文件已存在，跳过
	if _, err := os.Stat(localPath); err == nil {
		return
	}

	// 下载项目
	itemURL := fmt.Sprintf("%s/%s", baseURL, itemPath)
	resp, err := g.client.Get(itemURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 301 || resp.StatusCode == 302 {
		// 处理重定向
		if location := resp.Header.Get("Location"); location != "" && strings.HasSuffix(location, itemPath+"/") {
			g.downloadRecursiveItem(baseURL, itemPath+"/", outputDir)
			return
		}
	}

	if strings.HasSuffix(itemPath, "/") {
		// 目录 - 解析HTML并下载其中的文件
		if resp.StatusCode != 200 {
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		files := g.extractLinksFromHTML(string(body))

		for _, file := range files {
			fullPath := itemPath + file
			g.downloadRecursiveItem(baseURL, fullPath, outputDir)
		}
	} else {
		// 文件 - 直接下载
		if resp.StatusCode != 200 {
			return
		}

		// 创建目录
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return
		}

		// 写入文件
		file, err := os.Create(localPath)
		if err != nil {
			return
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			os.Remove(localPath)
			return
		}
	}
}

// 从HTML提取链接
func (g *GitDumper) extractLinksFromHTML(htmlContent string) []string {
	var links []string

	re := regexp.MustCompile(`<a\s+[^>]*href="([^"]*)"[^>]*>`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			// 过滤掉上级目录和无效链接
			if link != "../" && link != "/" && !strings.Contains(link, "://") {
				links = append(links, link)
			}
		}
	}
	return links
}

// 综合下载（当目录列表不可用时）
func (g *GitDumper) comprehensiveDownload(targetURL, outputDir string) {
	g.log("Starting comprehensive download")

	var wg sync.WaitGroup

	// 下载常见文件
	commonFiles := []string{
		".gitignore",
		".git/COMMIT_EDITMSG",
		".git/description",
		".git/hooks/applypatch-msg.sample",
		".git/hooks/commit-msg.sample",
		".git/hooks/post-commit.sample",
		".git/hooks/post-receive.sample",
		".git/hooks/post-update.sample",
		".git/hooks/pre-applypatch.sample",
		".git/hooks/pre-commit.sample",
		".git/hooks/pre-push.sample",
		".git/hooks/pre-rebase.sample",
		".git/hooks/pre-receive.sample",
		".git/hooks/prepare-commit-msg.sample",
		".git/hooks/update.sample",
		".git/index",
		".git/info/exclude",
		".git/objects/info/packs",
	}

	g.log("Downloading common files")
	for _, file := range commonFiles {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			g.downloadFile(targetURL, f, outputDir)
		}(file)
	}

	// 下载引用文件
	refFiles := []string{
		".git/FETCH_HEAD",
		".git/HEAD",
		".git/ORIG_HEAD",
		".git/config",
		".git/info/refs",
		".git/logs/HEAD",
		".git/logs/refs/heads/main",
		".git/logs/refs/heads/master",
		".git/logs/refs/heads/staging",
		".git/logs/refs/heads/production",
		".git/logs/refs/heads/development",
		".git/logs/refs/remotes/origin/HEAD",
		".git/logs/refs/remotes/origin/main",
		".git/logs/refs/remotes/origin/master",
		".git/logs/refs/remotes/origin/staging",
		".git/logs/refs/remotes/origin/production",
		".git/logs/refs/remotes/origin/development",
		".git/logs/refs/stash",
		".git/packed-refs",
		".git/refs/heads/main",
		".git/refs/heads/master",
		".git/refs/heads/staging",
		".git/refs/heads/production",
		".git/refs/heads/development",
		".git/refs/remotes/origin/HEAD",
		".git/refs/remotes/origin/main",
		".git/refs/remotes/origin/master",
		".git/refs/remotes/origin/staging",
		".git/refs/remotes/origin/production",
		".git/refs/remotes/origin/development",
		".git/refs/stash",
	}

	wg.Wait()

	g.log("Downloading ref files")
	for _, file := range refFiles {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			g.downloadFile(targetURL, f, outputDir)
		}(file)
	}

	wg.Wait()

	// 下载包文件
	g.log("Finding packs")
	infoPacksPath := filepath.Join(outputDir, ".git", "objects", "info", "packs")
	if content, err := os.ReadFile(infoPacksPath); err == nil {
		packRegex := regexp.MustCompile(`pack-([a-f0-9]{40})\.pack`)
		matches := packRegex.FindAllStringSubmatch(string(content), -1)

		for _, match := range matches {
			if len(match) > 1 {
				sha1 := match[1]
				wg.Add(1)
				go func(sha string) {
					defer wg.Done()
					g.downloadFile(targetURL, fmt.Sprintf(".git/objects/pack/pack-%s.idx", sha), outputDir)
				}(sha1)

				// 下载 pack 数据文件
				wg.Add(1)
				go func(sha string) {
					defer wg.Done()
					g.downloadFile(targetURL, fmt.Sprintf(".git/objects/pack/pack-%s.pack", sha), outputDir)
				}(sha1)
			}
		}
	}

	wg.Wait()

	// 查找并下载所有引用的对象
	g.log("Finding referenced objects")
	objects := g.findAllReferencedObjects(outputDir)

	g.log("Downloading %d referenced objects", len(objects))
	for obj := range objects {
		wg.Add(1)
		go func(o string) {
			defer wg.Done()
			g.downloadObject(targetURL, outputDir, o)
		}(obj)
	}

	wg.Wait()

	// 递归下载所有相关的对象
	g.log("Finding and downloading related objects")
	visited := make(map[string]bool)
	g.downloadRelatedObjects(targetURL, outputDir, objects, visited)
}

// 查找所有引用的对象
func (g *GitDumper) findAllReferencedObjects(outputDir string) map[string]bool {
	objects := make(map[string]bool)

	// 从各种文件中查找SHA1
	files := []string{
		filepath.Join(outputDir, ".git", "packed-refs"),
		filepath.Join(outputDir, ".git", "info", "refs"),
		filepath.Join(outputDir, ".git", "FETCH_HEAD"),
		filepath.Join(outputDir, ".git", "ORIG_HEAD"),
	}

	// 添加refs目录下的所有文件
	refsDir := filepath.Join(outputDir, ".git", "refs")
	if stat, err := os.Stat(refsDir); err == nil && stat.IsDir() {
		filepath.Walk(refsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	}

	// 添加logs目录下的所有文件
	logsDir := filepath.Join(outputDir, ".git", "logs")
	if stat, err := os.Stat(logsDir); err == nil && stat.IsDir() {
		filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	}

	// 从文件中提取SHA1
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		sha1Regex := regexp.MustCompile(`([a-f0-9]{40})`)
		matches := sha1Regex.FindAllStringSubmatch(string(content), -1)

		for _, match := range matches {
			if len(match) > 0 {
				objects[match[0]] = true
			}
		}
	}

	// 从index文件中查找对象
	indexPath := filepath.Join(outputDir, ".git", "index")
	if _, err := os.Stat(indexPath); err == nil {
		if repo, err := git.PlainOpen(outputDir); err == nil {
			if idx, err := repo.Storer.Index(); err == nil {
				for _, entry := range idx.Entries {
					objects[entry.Hash.String()] = true
				}
			}
		}
	}

	return objects
}

// 下载相关对象（递归）
func (g *GitDumper) downloadRelatedObjects(targetURL, outputDir string, objects map[string]bool, visited map[string]bool) {
	for obj := range objects {
		if visited[obj] {
			continue
		}
		visited[obj] = true

		// 下载对象
		if !g.downloadObject(targetURL, outputDir, obj) {
			continue
		}

		// 解析对象，找到相关的对象
		related := g.getReferencedObjects(outputDir, obj)
		if len(related) > 0 {
			newObjects := make(map[string]bool)
			for _, r := range related {
				if !visited[r] {
					newObjects[r] = true
				}
			}
			g.downloadRelatedObjects(targetURL, outputDir, newObjects, visited)
		}
	}
}

// 获取对象引用的其他对象
func (g *GitDumper) getReferencedObjects(outputDir, sha1 string) []string {
	if len(sha1) < 2 {
		return nil
	}

	objPath := filepath.Join(outputDir, ".git", "objects", sha1[:2], sha1[2:])
	if _, err := os.Stat(objPath); err != nil {
		return nil
	}

	data, err := os.ReadFile(objPath)
	if err != nil {
		return nil
	}

	// 解压zlib数据
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return nil
	}

	// 解析git对象
	nullIndex := bytes.IndexByte(decompressed, 0)
	if nullIndex == -1 {
		return nil
	}

	header := string(decompressed[:nullIndex])
	content := decompressed[nullIndex+1:]

	var referenced []string

	if strings.HasPrefix(header, "commit") {
		// 解析commit对象
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "tree ") {
				treeHash := strings.TrimSpace(line[5:])
				if isValidSHA1(treeHash) {
					referenced = append(referenced, treeHash)
				}
			} else if strings.HasPrefix(line, "parent ") {
				parentHash := strings.TrimSpace(line[7:])
				if isValidSHA1(parentHash) {
					referenced = append(referenced, parentHash)
				}
			}
		}
	} else if strings.HasPrefix(header, "tree") {
		// 解析tree对象
		pos := 0
		for pos < len(content) {
			// 解析模式
			spaceIndex := bytes.IndexByte(content[pos:], ' ')
			if spaceIndex == -1 {
				break
			}
			spaceIndex += pos

			// 解析文件名
			nullIndex := bytes.IndexByte(content[spaceIndex:], 0)
			if nullIndex == -1 {
				break
			}
			nullIndex += spaceIndex

			// 解析SHA1
			if nullIndex+21 > len(content) {
				break
			}
			hash := content[nullIndex+1 : nullIndex+21]
			sha1 := hex.EncodeToString(hash)
			if isValidSHA1(sha1) {
				referenced = append(referenced, sha1)
			}

			pos = nullIndex + 21
		}
	}

	return referenced
}

// 下载对象文件
func (g *GitDumper) downloadObject(targetURL, outputDir, sha1 string) bool {
	if len(sha1) < 2 {
		return false
	}

	objDir := sha1[:2]
	objFile := sha1[2:]
	objectPath := fmt.Sprintf(".git/objects/%s/%s", objDir, objFile)

	return g.downloadFile(targetURL, objectPath, outputDir)
}

// 下载单个文件
func (g *GitDumper) downloadFile(baseURL, filePath, outputDir string) bool {
	fileURL := fmt.Sprintf("%s/%s", baseURL, filePath)

	localPath := filepath.Join(outputDir, filePath)
	if _, err := os.Stat(localPath); err == nil {
		return true // 文件已存在
	}

	// 重试机制
	for attempt := 0; attempt < g.retry; attempt++ {
		// 检查取消信号
		select {
		case <-g.ctx.Done():
			return false
		default:
		}

		resp, err := g.client.Get(fileURL)
		if err != nil {
			if attempt < g.retry-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return false
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			if attempt < g.retry-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return false
		}

		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			resp.Body.Close()
			return false
		}

		file, err := os.Create(localPath)
		if err != nil {
			resp.Body.Close()
			return false
		}

		_, err = io.Copy(file, resp.Body)
		file.Close()
		resp.Body.Close()

		if err != nil {
			os.Remove(localPath)
			if attempt < g.retry-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return false
		}

		return true
	}

	return false
}

// 恢复git仓库
func (g *GitDumper) restoreGitRepo(repoPath string) error {
	g.log("Attempting to restore git repository")

	// 首先确保.git目录存在
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf(".git directory does not exist")
	}

	// 检查必要的文件是否存在
	requiredFiles := []string{
		filepath.Join(gitDir, "HEAD"),
		filepath.Join(gitDir, "config"),
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			g.logWarning("Required file missing: %s", file)
		}
	}

	// 初始化 Git 仓库（如果尚未初始化）
	if !g.isGitRepository(repoPath) {
		g.log("Initializing new git repository")
		cmd := exec.Command("git", "init")
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			g.logWarning("Git init failed: %v, output: %s", err, string(output))
			return fmt.Errorf("git init failed: %v", err)
		}
		g.logSuccess("Git repository initialized")
	}

	// 重新配置远程origin（避免安全问题）
	g.sanitizeConfig(repoPath)

	// 尝试修复 Git 对象
	g.log("Attempting to repair git objects")
	if err := g.repairGitObjects(repoPath); err != nil {
		g.logWarning("Git object repair failed: %v", err)
	}

	// 检查仓库完整性
	g.log("Checking repository integrity")
	cmd := exec.Command("git", "fsck", "--full")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		g.logWarning("Git fsck failed: %v, output: %s", err, string(output))
	} else {
		g.logSuccess("Repository integrity check passed")
	}

	// 尝试重置到最新提交
	g.log("Attempting to reset to latest commit")
	cmd = exec.Command("git", "reset", "--hard", "HEAD")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		g.logWarning("Git reset failed: %v, output: %s", err, string(output))

		// 如果reset失败，尝试checkout
		g.log("Trying git checkout instead")
		cmd = exec.Command("git", "checkout", "-f")
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			g.logWarning("Git checkout failed: %v, output: %s", err, string(output))
		} else {
			g.logSuccess("Git checkout completed successfully")
		}
	} else {
		g.logSuccess("Git reset completed successfully")
	}

	// 显示恢复的状态
	g.logRepositoryStatus(repoPath)

	g.logSuccess("Git repository restoration completed")
	return nil
}

// 检查是否为有效的 Git 仓库
func (g *GitDumper) isGitRepository(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}

	// 检查 HEAD 文件
	headFile := filepath.Join(gitDir, "HEAD")
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return false
	}

	// 运行 git status 验证
	cmd := exec.Command("git", "status")
	cmd.Dir = repoPath
	return cmd.Run() == nil
}

// 修复 Git 对象
func (g *GitDumper) repairGitObjects(repoPath string) error {
	objectsDir := filepath.Join(repoPath, ".git", "objects")

	// 检查对象目录是否存在
	if _, err := os.Stat(objectsDir); os.IsNotExist(err) {
		return fmt.Errorf("objects directory does not exist")
	}

	// 尝试重新打包对象
	g.log("Repacking git objects")
	cmd := exec.Command("git", "repack", "-a", "-d")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		g.logWarning("Git repack failed: %v, output: %s", err, string(output))
	}

	// 清理无效的引用
	g.log("Pruning invalid references")
	cmd = exec.Command("git", "prune")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		g.logWarning("Git prune failed: %v, output: %s", err, string(output))
	}

	return nil
}

// 显示仓库状态
func (g *GitDumper) logRepositoryStatus(repoPath string) {
	// 获取分支信息
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err == nil {
		g.log("Available branches:\n%s", string(output))
	}

	// 获取日志信息
	cmd = exec.Command("git", "log", "--oneline", "-10")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err == nil {
		g.log("Recent commits:\n%s", string(output))
	}

	// 获取文件状态
	cmd = exec.Command("git", "status")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err == nil {
		g.log("Repository status:\n%s", string(output))
	}
}

func (g *GitDumper) sanitizeConfig(repoPath string) {
	configPath := filepath.Join(repoPath, ".git", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	unsafeRegex := regexp.MustCompile(`(?i)^\s*(fsmonitor|sshcommand|askpass|editor|pager)`)
	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		if unsafeRegex.MatchString(line) {
			newLines = append(newLines, "# "+line)
		} else {
			newLines = append(newLines, line)
		}
	}

	os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func isValidHEAD(content string) bool {
	refRegex := regexp.MustCompile(`^ref:\s+refs/heads/`)
	shaRegex := regexp.MustCompile(`^[0-9a-f]{40}$`)
	return refRegex.MatchString(content) || shaRegex.MatchString(content)
}

func isValidSHA1(sha1 string) bool {
	if len(sha1) != 40 {
		return false
	}
	_, err := hex.DecodeString(sha1)
	return err == nil
}

// 清理文件名，移除非法字符但保留汉字
func sanitizeFilename(filename string) string {
	illegalChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range illegalChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	filename = strings.TrimSpace(filename)
	filename = strings.Trim(filename, ".")

	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

func (a *GitDumperApp) GetDefaultOutputDir() (string, error) {
	defaultDir := filepath.Join(system.GetAppBaseDir(), "file", "git-dumper-output")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		return "", err
	}
	return defaultDir, nil
}

func (a *GitDumperApp) ParseURLsFromText(text string) []string {
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

func (a *GitDumperApp) RetryScan(request RetryGitScanRequest) (GitScanResult, error) {
	return a.scanSingleURL(request.URL, request.OutputDir, request.Jobs, request.Timeout, request.Retry, ""), nil
}

func (a *GitDumperApp) GetScanStatus(url string) (GitScanResult, error) {
	return GitScanResult{
		URL:     url,
		Success: false,
		Message: "Status check not implemented",
	}, nil
}
