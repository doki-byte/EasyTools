package system

import (
	"encoding/base64"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/wailsapp/wails/v2/pkg/runtime"
)

// System 系统API
type System struct {
	Base
	portConfig *PortConfig // 添加端口配置字段
}

func NewSystem() *System {
	return &System{}
}

// 设置端口配置
func (s *System) SetPortConfig(config *PortConfig) {
	s.portConfig = config
}

// 获取端口配置
func (s *System) GetPortConfig() *PortConfig {
	return s.portConfig
}

// BrowserOpenURL 默认浏览器打开网址
func (s *System) BrowserOpenURL(url string) {
	BrowserOpenURL(s.Ctx, url)
}

// GetOpenDir 获取选择的目录路径
func (s System) GetOpenDir() string {
	path, _ := OpenDirectoryDialog(s.Ctx, OpenDialogOptions{})
	return path
}

// GetOpenFilePath 获取选择的目录路径
func (s System) GetOpenFilePath() string {
	path, _ := OpenFileDialog(s.Ctx, OpenDialogOptions{})
	return path
}

// GetOpenChooseFilePath 获取选择的目录路径
func (s System) GetOpenChooseFilePath(initialDir string) string {
	options := OpenDialogOptions{}

	// 如果传入了初始目录，设置初始目录
	if initialDir != "" {
		options.DefaultDirectory = initialDir
	}

	path, _ := OpenFileDialog(s.Ctx, options)
	return path
}

// ClipboardGetText 获取剪切板内容
func (s *System) ClipboardGetText() (string, error) {
	return ClipboardGetText(s.Ctx)
}

// ClipboardSetText 设置剪切板内容
func (s *System) ClipboardSetText(text string) error {
	return ClipboardSetText(s.Ctx, text)
}

// ShellCMD 以shell方式运行cmd命令
func (s *System) ShellCMD(cmdPath, cmdStr, paramStr string, terminal int) {
	s.shellCMD(cmdPath, cmdStr, paramStr, terminal)
}

// OpenConfigDir 打开应用配置目录
func (s *System) OpenConfigDir() {
	s.openDir(s.GetAppPath())
}

// GetConfigDir 打开应用配置目录
func (s *System) GetConfigDir() string {
	return s.GetAppPath()
}

// GetOs 获取系统类型
func (s *System) GetOs() string {
	//windows、darwin、linux
	return runtime.GOOS
}

func (s *System) OpenPath(path string) error {
	// 标准化路径（处理 ~ 等特殊符号）
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	log.Println("将打开路径：", absPath)

	// 检查路径是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return err
	}
	// 跨平台支持
	if runtime.GOOS == "windows" {
		path = filepath.Clean(path)
		cmd := exec.Command("explorer", path)
		return cmd.Start()
	} else if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", path)
		return cmd.Start()
	} else if runtime.GOOS == "linux" {
		cmd := exec.Command("xdg-open", path)
		return cmd.Start()
	} else {
		return exec.Command("open", absPath).Start() // 默认尝试 unix 方式
	}

}

// OpenTerminalInDir 在指定路径所在的目录中打开系统终端（path 可为文件或文件夹）
func (s *System) OpenTerminalInDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return os.ErrInvalid
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	workDir := absPath
	if !info.IsDir() {
		workDir = filepath.Dir(absPath)
	}
	workDir = filepath.Clean(workDir)
	log.Println("将在目录打开终端：", workDir)

	switch runtime.GOOS {
	case "windows":
		if wt, err := exec.LookPath("wt"); err == nil {
			return exec.Command(wt, "-d", workDir).Start()
		}
		if wt, err := exec.LookPath("wt.exe"); err == nil {
			return exec.Command(wt, "-d", workDir).Start()
		}
		q := `"` + strings.ReplaceAll(workDir, `"`, `""`) + `"`
		return exec.Command("cmd.exe", "/K", "cd /d "+q).Start()
	case "darwin":
		esc := strings.ReplaceAll(strings.ReplaceAll(workDir, `\`, `\\`), `"`, `\"`)
		script := fmt.Sprintf(`tell application "Terminal" to do script "cd " & quoted form of POSIX path of (POSIX file "%s")`, esc)
		return exec.Command("osascript", "-e", script).Start()
	default:
		shCmd := fmt.Sprintf("cd %q && exec bash", workDir)
		if _, err := exec.LookPath("gnome-terminal"); err == nil {
			return exec.Command("gnome-terminal", "--", "bash", "-c", shCmd).Start()
		}
		if _, err := exec.LookPath("konsole"); err == nil {
			return exec.Command("konsole", "-e", "bash", "-c", shCmd).Start()
		}
		if _, err := exec.LookPath("xfce4-terminal"); err == nil {
			return exec.Command("xfce4-terminal", "-x", "bash", "-c", shCmd).Start()
		}
		if _, err := exec.LookPath("xterm"); err == nil {
			return exec.Command("xterm", "-e", "bash", "-c", shCmd).Start()
		}
		if _, err := exec.LookPath("x-terminal-emulator"); err == nil {
			return exec.Command("x-terminal-emulator", "-e", "bash", "-c", shCmd).Start()
		}
		return fmt.Errorf("未找到可用的终端程序")
	}
}

func (s *System) GetCPUUsage() float64 {
	percent, _ := cpu.Percent(time.Second, false)
	if len(percent) > 0 {
		return percent[0]
	}
	return 0
}

func (s *System) GetMemUsageSelf() float64 {
	p, _ := process.NewProcess(int32(os.Getpid()))
	memoryInfo, _ := p.MemoryInfo()
	if memoryInfo != nil {
		return float64(memoryInfo.RSS) / 1024 / 1024 // 转换为MB
	}
	return 0
}

func (s *System) GetMemUsageTotal() float64 {
	memory, _ := mem.VirtualMemory()
	if memory != nil {
		return memory.UsedPercent
	}
	return 0
}

// 获取应用基础目录
func GetAppBaseDir() string {
	// 如果是 macOS，使用应用支持目录
	if runtime.GOOS == "darwin" {
		appName := "EasyTools"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("获取用户主目录失败: " + err.Error())
		}
		return filepath.Join(homeDir, "Library", "Application Support", appName)
	}

	// 其他系统使用当前目录下的 EasyToolsFiles
	currentPath, err := os.Getwd()
	if err != nil {
		panic("获取当前路径失败: " + err.Error())
	}
	return filepath.Join(currentPath, "EasyToolsFiles")
}

// GetUsernameDictPath 获取用户名字典文件路径
// protocol: 协议名，如 "ftp", "ssh", "telnet" 等
// 返回: 应用基础目录/tools/dict/username/协议名.txt
func GetUsernameDictPath(protocol string) string {
	baseDir := GetAppBaseDir()
	// 将协议名转换为小写，确保路径一致性
	protocol = strings.ToLower(protocol)
	return filepath.Join(baseDir, "tools", "dict", "username", protocol+".txt")
}

// GetPasswordDictPath 获取密码字典文件路径
// 返回: 应用基础目录/tools/dict/password.txt
func GetPasswordDictPath() string {
	baseDir := GetAppBaseDir()
	return filepath.Join(baseDir, "tools", "dict", "password.txt")
}

// EnsureDictDir 确保字典目录存在
func EnsureDictDir() error {
	baseDir := GetAppBaseDir()
	usernameDictDir := filepath.Join(baseDir, "tools", "dict", "username")
	passwordDictDir := filepath.Join(baseDir, "tools", "dict")
	
	// 创建用户名字典目录
	if err := os.MkdirAll(usernameDictDir, 0755); err != nil {
		return err
	}
	
	// 确保密码字典目录存在
	if err := os.MkdirAll(passwordDictDir, 0755); err != nil {
		return err
	}
	
	return nil
}

// 获取自启动配置
func (s *System) GetAutoStart() bool {
	return s.getAutoStart()
}

// 设置自启动
func (s *System) SetAutoStart(enabled bool) bool {
	return s.setAutoStart(enabled)
}

func (s *System) ExitApp() {
	// 释放所有占用的端口
	if s.portConfig != nil {
		PortManager.ReleasePort(s.portConfig.MainPort)
		PortManager.ReleasePort(s.portConfig.SSHPort)
		PortManager.ReleasePort(s.portConfig.FTPPort)
		PortManager.ReleasePort(s.portConfig.FuzzPort)
	}
	os.Exit(0)
}

// ReadFileAsDataURL 读取文件并转换为 data URL 格式（支持图片和视频）
func (s *System) ReadFileAsDataURL(filePath string) string {
	if filePath == "" {
		return ""
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("读取文件错误: %v", err)
		return ""
	}

	// 检测 MIME 类型
	mimeType := http.DetectContentType(data)
	
	// 如果检测失败或检测为通用类型，尝试根据文件扩展名判断
	if mimeType == "application/octet-stream" || mimeType == "text/plain; charset=utf-8" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		case ".bmp":
			mimeType = "image/bmp"
		case ".mp4":
			mimeType = "video/mp4"
		case ".webm":
			mimeType = "video/webm"
		case ".ogg":
			mimeType = "video/ogg"
		case ".mov":
			mimeType = "video/quicktime"
		case ".avi":
			mimeType = "video/x-msvideo"
		case ".mkv":
			mimeType = "video/x-matroska"
		default:
			// 如果无法确定类型，尝试使用检测到的类型
		}
	}

	// 编码为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	
	// 构建 data URL
	dataURL := "data:" + mimeType + ";base64," + base64Data
	
	return dataURL
}

// FileItem 文件项结构
type FileItem struct {
	Name     string `json:"name"`     // 文件名或目录名
	Path     string `json:"path"`     // 完整路径
	IsDir    bool   `json:"isDir"`    // 是否为目录
	Size     int64  `json:"size"`     // 文件大小（字节）
	Modified string `json:"modified"` // 修改时间
}

// GetUserHomeDir 获取用户主目录
func (s *System) GetUserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetDefaultBackupPath 获取默认备份路径（用户主目录/EasyTools）
func (s *System) GetDefaultBackupPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "EasyTools"), nil
}

// ListConfigFiles 获取配置目录下的文件和文件夹列表
func (s *System) ListConfigFiles() ([]FileItem, error) {
	configDir := s.GetAppPath()
	var items []FileItem

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(configDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := FileItem{
			Name:  entry.Name(),
			Path:  fullPath,
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		}

		if modTime := info.ModTime(); !modTime.IsZero() {
			item.Modified = modTime.Format("2006-01-02 15:04:05")
		}

		items = append(items, item)
	}

	return items, nil
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 普通文件直接复制
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return copyFile(path, targetPath)
	})
}

// BackupConfigFiles 执行备份操作
// backupPath: 备份保存路径
// items: 要备份的文件/文件夹路径列表（相对于配置目录）
// overwrite: 是否覆盖模式（true=覆盖，false=创建副本）
func (s *System) BackupConfigFiles(backupPath string, items []string, overwrite bool) error {
	configDir := s.GetAppPath()

	// 创建备份目录（如果不存在）
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	var backupDir string
	if overwrite {
		// 覆盖模式：直接使用备份路径作为目标目录
		backupDir = backupPath
	} else {
		// 创建副本模式：创建带时间戳的目录
		timestamp := time.Now().Format("20060102_150405")
		backupDir = filepath.Join(backupPath, "backup_"+timestamp)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return err
		}
	}

	// 复制选中的文件和目录
	for _, item := range items {
		srcPath := filepath.Join(configDir, item)
		dstPath := filepath.Join(backupDir, item)

		info, err := os.Stat(srcPath)
		if err != nil {
			log.Printf("跳过不存在的项: %s", srcPath)
			continue
		}

		if info.IsDir() {
			// 如果是覆盖模式，先删除目标目录
			if overwrite {
				if err := os.RemoveAll(dstPath); err != nil && !os.IsNotExist(err) {
					log.Printf("删除旧目录失败 %s: %v", dstPath, err)
				}
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				log.Printf("复制目录失败 %s: %v", srcPath, err)
				return err
			}
		} else {
			// 如果是覆盖模式，目标文件已存在会被覆盖
			if err := copyFile(srcPath, dstPath); err != nil {
				log.Printf("复制文件失败 %s: %v", srcPath, err)
				return err
			}
		}
	}

	return nil
}
