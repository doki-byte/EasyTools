package system

import (
	"EasyTools/app/proxy"
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	runtime2 "runtime"
	"strings"
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
		//fmt.Printf("发送下载进度: %.1f%%\n", progress)
	} else {
		//fmt.Printf("警告: 全局上下文为空，无法发送进度: %.1f%%\n", progress)
	}
}

type Update struct {
	Base
	ctx context.Context
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
	currentVersion := "v1.9.4"

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
	var assetName string

	switch runtime2.GOOS {
	case "windows":
		assetName = "EasyTools-windows-amd64.zip"
	case "darwin":
		if runtime2.GOARCH == "amd64" {
			assetName = "EasyTools-darwin-amd64.zip"
		} else {
			assetName = "EasyTools-darwin-arm64.zip"
		}
	case "linux":
		assetName = "EasyTools-linux-amd64.tar.gz"
	default:
		return "", fmt.Errorf("不支持的操作系统: %s/%s", runtime2.GOOS, runtime2.GOARCH)
	}

	// 在assets中查找匹配的文件
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("未找到适合当前系统的发布文件: %s", assetName)
}

// 下载并更新应用
func (u *Update) DownloadAndUpdate() (*DownloadResult, error) {
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
		return u.updateWindows(downloadURL)
	case "darwin":
		return u.updateDarwin(downloadURL)
	case "linux":
		return u.updateLinux(downloadURL)
	default:
		return &DownloadResult{Error: true, Msg: "不支持的操作系统"}, nil
	}
}

// 下载文件并显示进度 - 使用全局事件发射器
func (u *Update) downloadWithProgress(downloadURL, filePath string) error {
	client := proxy.GlobalProxyManager.GetHTTPClient()
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("请求创建失败: %s", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
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

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			writer.Write(buf[:n])
			downloadedSize += int64(n)

			// 发送进度到前端
			if totalSize > 0 {
				progress := float64(downloadedSize) / float64(totalSize)
				progressPercent := progress * 100

				// 只有当进度变化超过1%时才发送，避免过于频繁
				if progressPercent-lastProgress >= 1.0 || progressPercent >= 100 {
					EmitDownloadProgress(progressPercent)
					lastProgress = progressPercent
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	writer.Flush()

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
func (u *Update) updateWindows(downloadURL string) (*DownloadResult, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "easytools-update")
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "update.zip")

	// 下载ZIP文件
	if err := u.downloadWithProgress(downloadURL, zipPath); err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 解压ZIP文件
	exePath, err := u.extractZip(zipPath, tempDir)
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 读取新的可执行文件
	newExeData, err := os.ReadFile(exePath)
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 应用更新
	err = selfupdate.Apply(bytes.NewReader(newExeData), selfupdate.Options{})
	if err != nil {
		// 如果更新失败，尝试回滚
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return &DownloadResult{Error: true, Msg: fmt.Sprintf("更新失败且回滚也失败: %v", rerr)}, nil
		}
		return &DownloadResult{Error: true, Msg: fmt.Sprintf("更新失败: %v", err)}, nil
	}

	// 更新成功后，清理旧文件
	u.comprehensiveCleanup()

	return &DownloadResult{Error: false, Msg: "更新成功，应用将自动重启"}, nil
}

// macOS更新
func (u *Update) updateDarwin(downloadURL string) (*DownloadResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	downloadPath := filepath.Join(homeDir, "Downloads", filepath.Base(downloadURL))

	// 下载ZIP文件
	if err := u.downloadWithProgress(downloadURL, downloadPath); err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 创建临时目录用于解压
	tempDir, err := os.MkdirTemp("", "easytools-update")
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}
	defer os.RemoveAll(tempDir)

	// 解压ZIP文件
	_, err = u.extractZip(downloadPath, tempDir)
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 更新成功后，清理旧文件
	u.comprehensiveCleanup()

	return &DownloadResult{
		Error: false,
		Msg:   fmt.Sprintf("更新文件已下载到: %s\n请手动解压并安装", downloadPath),
	}, nil
}

// Linux更新
func (u *Update) updateLinux(downloadURL string) (*DownloadResult, error) {
	exePath, err := os.Executable()
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "easytools-update")
	if err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
	}
	defer os.RemoveAll(tempDir)

	tarGzPath := filepath.Join(tempDir, "update.tar.gz")

	// 下载tar.gz文件
	if err := u.downloadWithProgress(downloadURL, tarGzPath); err != nil {
		return &DownloadResult{Error: true, Msg: err.Error()}, nil
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
