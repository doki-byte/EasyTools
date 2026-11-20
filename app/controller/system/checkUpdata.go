package system

import (
	"EasyTools/app/controller/proxy"
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	runtime2 "runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/minio/selfupdate"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 添加全局上下文变量定义
var globalCtx context.Context

// 全局事件发射器函数
func SetGlobalContext(ctx context.Context) {
	globalCtx = ctx
}

func EmitDownloadProgress(progress float64) {
	if globalCtx != nil {
		runtime.EventsEmit(globalCtx, "downloadProgress", progress)
	}
}

func EmitDownloadCancelled() {
	if globalCtx != nil {
		runtime.EventsEmit(globalCtx, "downloadCancelled")
	}
}

type Update struct {
	Base
	ctx context.Context

	// 添加下载控制相关字段
	downloadCancel context.CancelFunc
	downloadMutex  sync.Mutex
	isDownloading  bool
	currentTempDir string
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type CheckResult struct {
	HasUpdate      bool           `json:"hasUpdate"`
	CurrentVersion string         `json:"currentVersion"`
	LatestRelease  *GitHubRelease `json:"latestRelease"`
}

type DownloadResult struct {
	Error bool   `json:"error"`
	Msg   string `json:"msg"`
}

func NewUpdate() *Update {
	return &Update{}
}

// Startup 方法用于 Wails 注入上下文
func (u *Update) Startup(ctx context.Context) {
	u.ctx = ctx
	// 同时设置到全局上下文
	SetGlobalContext(ctx)
}

func (u *Update) GetLatestRelease() (*CheckResult, error) {
	owner := "doki-byte"
	repo := "EasyTools"
	currentVersion := "v2.3.1"

	latest, err := CheckLatestRelease(owner, repo)
	if err != nil {
		return nil, err
	}

	currentVer, err := version.NewVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("当前版本解析失败: %v", err)
	}

	latestVer, err := version.NewVersion(latest.TagName)
	if err != nil {
		return nil, fmt.Errorf("最新版本解析失败: %v", err)
	}

	hasUpdate := latestVer.GreaterThan(currentVer)

	return &CheckResult{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVersion,
		LatestRelease:  latest,
	}, nil
}

func CheckLatestRelease(owner, repo string) (*GitHubRelease, error) {
	return CheckLatestReleaseWithProxy(owner, repo)
}

// CheckLatestReleaseWithProxy 检查 GitHub 最新版本，支持可选的系统代理
func CheckLatestReleaseWithProxy(owner, repo string) (*GitHubRelease, error) {
	client := proxy.GlobalProxyManager.GetHTTPClient()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求创建失败: %s", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回错误状态码: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %s", err)
	}

	fmt.Printf("最新版本信息: %s - %s\n", release.TagName, release.HTMLURL)
	return &release, nil
}

// 获取适合当前系统的下载文件URL
func (u *Update) getDownloadURL(release *GitHubRelease) (string, error) {
	var candidateAssetNames []string

	switch runtime2.GOOS {
	case "windows":
		candidateAssetNames = []string{"EasyTools-windows-amd64.zip"}
	case "darwin":
		if runtime2.GOARCH == "amd64" {
			// 优先匹配新的 dmg 命名，同时兼容旧 zip 包
			candidateAssetNames = []string{
				"EasyTools-darwin-amd64.dmg",
				"EasyTools-darwin-amd64.zip",
			}
		} else {
			candidateAssetNames = []string{
				"EasyTools-darwin-arm64.dmg",
				"EasyTools-darwin-arm64.zip",
			}
		}
	case "linux":
		candidateAssetNames = []string{"EasyTools-linux-amd64.tar.gz"}
	default:
		return "", fmt.Errorf("不支持的操作系统: %s/%s", runtime2.GOOS, runtime2.GOARCH)
	}

	// 在assets中查找匹配的文件
	for _, asset := range release.Assets {
		for _, candidate := range candidateAssetNames {
			if asset.Name == candidate {
				return asset.BrowserDownloadURL, nil
			}
		}
	}

	return "", fmt.Errorf("未找到适合当前系统的发布文件，候选列表: %s", strings.Join(candidateAssetNames, ", "))
}

// 取消下载
func (u *Update) CancelDownload() (*DownloadResult, error) {
	u.downloadMutex.Lock()
	defer u.downloadMutex.Unlock()

	if u.isDownloading && u.downloadCancel != nil {
		fmt.Println("正在取消下载...")
		u.downloadCancel()
		u.cleanupTempFiles()
		u.isDownloading = false
		EmitDownloadCancelled()
		return &DownloadResult{Error: false, Msg: "下载已取消"}, nil
	}

	return &DownloadResult{Error: true, Msg: "没有正在进行的下载任务"}, nil
}

// 清理临时文件
func (u *Update) cleanupTempFiles() {
	if u.currentTempDir != "" {
		fmt.Printf("清理临时目录: %s\n", u.currentTempDir)
		os.RemoveAll(u.currentTempDir)
		u.currentTempDir = ""
	}
}

// 下载并更新应用
func (u *Update) DownloadAndUpdate() (*DownloadResult, error) {
	u.downloadMutex.Lock()
	if u.isDownloading {
		u.downloadMutex.Unlock()
		return &DownloadResult{Error: true, Msg: "已有下载任务正在进行"}, nil
	}

	// 创建可取消的上下文 - 修复 nil 上下文问题
	var ctx context.Context
	var cancel context.CancelFunc
	if u.ctx != nil {
		ctx, cancel = context.WithCancel(u.ctx)
	} else {
		// 如果 u.ctx 为 nil，使用 background context
		ctx, cancel = context.WithCancel(context.Background())
	}
	u.downloadCancel = cancel
	u.isDownloading = true
	u.downloadMutex.Unlock()

	// 确保在函数退出时清理资源
	defer func() {
		u.downloadMutex.Lock()
		u.isDownloading = false
		u.downloadCancel = nil
		u.downloadMutex.Unlock()
	}()

	result, err := u.GetLatestRelease()
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	if !result.HasUpdate {
		return &DownloadResult{Error: false, Msg: "当前已经是最新版本"}, nil
	}

	downloadURL, err := u.getDownloadURL(result.LatestRelease)
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 根据系统类型执行不同的更新逻辑
	switch runtime2.GOOS {
	case "windows":
		return u.updateWindows(ctx, downloadURL)
	case "darwin":
		return u.updateDarwin(ctx, downloadURL)
	case "linux":
		return u.updateLinux(ctx, downloadURL)
	default:
		return &DownloadResult{Error: true, Msg: "不支持的操作系统"}, nil
	}
}

// 下载文件并显示进度 - 使用可取消的上下文
func (u *Update) downloadWithProgress(ctx context.Context, downloadURL, filePath string) error {
	// 为更新下载单独构建长超时时间的 HTTP 客户端，沿用全局代理配置
	baseClient := proxy.GlobalProxyManager.GetHTTPClient()
	var transport *http.Transport
	if baseClient != nil {
		if t, ok := baseClient.Transport.(*http.Transport); ok {
			// 复制一份 transport，避免并发时修改共享对象
			copyT := t.Clone()
			// 确保超时较长，连接和 TLS 都保持
			if copyT.DialContext == nil {
				copyT.DialContext = (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext
			}
			if copyT.TLSClientConfig == nil {
				copyT.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			} else {
				copyT.TLSClientConfig.InsecureSkipVerify = true
			}
			transport = copyT
		}
	}
	// fallback：无全局配置时使用直连
	if transport == nil {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   0, // 由 ctx 控制超时/取消
	}
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("请求创建失败: %s", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		// 检查是否是取消导致的错误
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("下载已取消")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	totalSize := resp.ContentLength
	downloadedSize := int64(0)
	reader := bufio.NewReaderSize(resp.Body, 32*1024)
	writer := bufio.NewWriter(file)
	buf := make([]byte, 32*1024)

	// 使用全局事件发射器发送初始进度
	EmitDownloadProgress(0.0)

	var lastProgress float64 = 0
	lastEmitTime := time.Now()
	const minEmitInterval = 100 * time.Millisecond // 最小发送间隔

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			file.Close()
			// 删除已下载的部分文件
			os.Remove(filePath)
			return fmt.Errorf("下载已取消")
		default:
			// 继续下载
		}

		n, err := reader.Read(buf)
		if n > 0 {
			_, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloadedSize += int64(n)

			// 发送进度到前端
			now := time.Now()
			shouldEmit := false
			var progressPercent float64

			if totalSize > 0 {
				// 有总大小，计算百分比
				progress := float64(downloadedSize) / float64(totalSize)
				progressPercent = progress * 100

				// 只有当进度变化超过1%时才发送，避免过于频繁
				if progressPercent-lastProgress >= 1.0 || progressPercent >= 100 {
					shouldEmit = true
				}
			} else {
				// 没有总大小，基于时间发送更新
				// 每100ms发送一次更新，显示不确定进度
				if now.Sub(lastEmitTime) >= minEmitInterval {
					// 使用不确定进度（显示为50%左右，表示正在下载）
					progressPercent = 50.0
					shouldEmit = true
				}
			}

			if shouldEmit {
				EmitDownloadProgress(progressPercent)
				lastProgress = progressPercent
				lastEmitTime = now
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			// 检查是否是取消导致的错误
			if ctx.Err() == context.Canceled {
				file.Close()
				os.Remove(filePath)
				return fmt.Errorf("下载已取消")
			}
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	// 发送完成进度
	EmitDownloadProgress(100.0)

	return nil
}

// 解压ZIP文件
func (u *Update) extractZip(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var exePath string
	for _, f := range r.File {
		// 跳过目录
		if f.FileInfo().IsDir() {
			continue
		}

		// 构建目标文件路径
		targetPath := filepath.Join(destDir, f.Name)

		// 创建目录
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", err
		}

		// 打开源文件
		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		// 创建目标文件
		targetFile, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return "", err
		}

		// 复制文件内容
		_, err = io.Copy(targetFile, rc)
		targetFile.Close()
		rc.Close()

		if err != nil {
			return "", err
		}

		// 设置执行权限（非Windows系统）
		if runtime2.GOOS != "windows" {
			os.Chmod(targetPath, 0755)
		}

		// 检查是否是可执行文件
		if strings.Contains(f.Name, "EasyTools") && !strings.Contains(f.Name, ".dmg") {
			exePath = targetPath
		}
	}

	if exePath == "" {
		return "", fmt.Errorf("未找到可执行文件")
	}

	return exePath, nil
}

// 解压tar.gz文件
func (u *Update) extractTarGz(tarGzPath, destDir string) (string, error) {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var exePath string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// 跳过目录
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// 构建目标文件路径
		targetPath := filepath.Join(destDir, header.Name)

		// 创建目录
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", err
		}

		// 创建目标文件
		targetFile, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}

		// 复制文件内容
		if _, err := io.Copy(targetFile, tr); err != nil {
			targetFile.Close()
			return "", err
		}
		targetFile.Close()

		// 设置执行权限
		os.Chmod(targetPath, 0755)

		// 检查是否是可执行文件
		if strings.Contains(header.Name, "EasyTools") {
			exePath = targetPath
		}
	}

	if exePath == "" {
		return "", fmt.Errorf("未找到可执行文件")
	}

	return exePath, nil
}

// Windows更新
func (u *Update) updateWindows(ctx context.Context, downloadURL string) (*DownloadResult, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "easytools-update")
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}
	u.currentTempDir = tempDir

	zipPath := filepath.Join(tempDir, "update.zip")

	// 下载ZIP文件
	if err := u.downloadWithProgress(ctx, downloadURL, zipPath); err != nil {
		if err.Error() == "下载已取消" {
			u.cleanupTempFiles()
			return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
		}
		u.cleanupTempFiles()
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		u.cleanupTempFiles()
		return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
	default:
		// 继续执行
	}

	// 解压ZIP文件
	exePath, err := u.extractZip(zipPath, tempDir)
	if err != nil {
		u.cleanupTempFiles()
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 读取新的可执行文件
	newExeData, err := os.ReadFile(exePath)
	if err != nil {
		u.cleanupTempFiles()
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 应用更新
	err = selfupdate.Apply(bytes.NewReader(newExeData), selfupdate.Options{})
	if err != nil {
		// 如果更新失败，尝试回滚
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			u.cleanupTempFiles()
			return &DownloadResult{Error: true, Msg: fmt.Sprintf("更新失败且回滚也失败: %v", rerr)}, nil
		}
		u.cleanupTempFiles()
		return &DownloadResult{Error: true, Msg: fmt.Sprintf("更新失败: %v", err)}, nil
	}

	// 更新成功后，清理旧文件
	u.comprehensiveCleanup()
	u.cleanupTempFiles()

	return &DownloadResult{Error: false, Msg: "更新成功，应用将自动重启"}, nil
}

// macOS更新
func (u *Update) updateDarwin(ctx context.Context, downloadURL string) (*DownloadResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	downloadPath := filepath.Join(homeDir, "Downloads", filepath.Base(downloadURL))

	// 下载ZIP文件
	if err := u.downloadWithProgress(ctx, downloadURL, downloadPath); err != nil {
		if err.Error() == "下载已取消" {
			// 删除已下载的文件
			os.Remove(downloadPath)
			return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
		}
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		// 删除已下载的文件
		os.Remove(downloadPath)
		return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
	default:
		// 继续执行
	}

	// 兼容历史 ZIP 包与新的 DMG 包
	if strings.HasSuffix(strings.ToLower(downloadPath), ".zip") {
		// 创建临时目录用于解压
		tempDir, err := os.MkdirTemp("", "easytools-update")
		if err != nil {
			os.Remove(downloadPath)
			return &DownloadResult{Error: true, Msg: err.Error()}, nil
		}
		u.currentTempDir = tempDir
		defer u.cleanupTempFiles()

		// 解压ZIP文件
		_, err = u.extractZip(downloadPath, tempDir)
		if err != nil {
			os.Remove(downloadPath)
			return &DownloadResult{Error: true, Msg: err.Error()}, nil
		}
	}

	// 更新成功后，清理旧文件
	u.comprehensiveCleanup()

	installHint := "请手动解压并安装"
	if strings.HasSuffix(strings.ToLower(downloadPath), ".dmg") {
		installHint = "请打开 dmg 并手动安装"
	}

	return &DownloadResult{
		Error: false,
		Msg:   fmt.Sprintf("更新文件已下载到: %s\n%s", downloadPath, installHint),
	}, nil
}

// Linux更新
func (u *Update) updateLinux(ctx context.Context, downloadURL string) (*DownloadResult, error) {
	exePath, err := os.Executable()
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "easytools-update")
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}
	u.currentTempDir = tempDir
	defer u.cleanupTempFiles()

	tarGzPath := filepath.Join(tempDir, "update.tar.gz")

	// 下载tar.gz文件
	if err := u.downloadWithProgress(ctx, downloadURL, tarGzPath); err != nil {
		if err.Error() == "下载已取消" {
			return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
		}
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return &DownloadResult{Error: true, Msg: "下载已取消"}, nil
	default:
		// 继续执行
	}

	// 解压tar.gz文件
	newExePath, err := u.extractTarGz(tarGzPath, tempDir)
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 替换旧文件
	if err := os.Rename(newExePath, exePath); err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 设置执行权限
	if err := os.Chmod(exePath, 0755); err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 更新成功后，清理旧文件
	u.comprehensiveCleanup()

	return &DownloadResult{Error: false, Msg: "更新成功，请重启应用"}, nil
}

// 重启应用
func (u *Update) RestartApplication() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// 给一点时间让前端收到消息
	time.Sleep(500 * time.Millisecond)

	cmd := exec.Command(executable)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}

// 更完善的清理方法
func (u *Update) comprehensiveCleanup() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	exeDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)

	// 查找并删除所有相关的旧文件
	patterns := []string{
		exeName + ".old",
		exeName + ".backup",
		"*.old",
		"*.backup",
		"." + exeName + ".old",
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(exeDir, pattern))
		if err != nil {
			continue
		}

		for _, file := range files {
			// 确保不会误删当前运行的可执行文件
			if file != exePath {
				err := os.Remove(file)
				if err == nil {
					fmt.Printf("清理旧文件: %s\n", file)
				}
			}
		}
	}
}
