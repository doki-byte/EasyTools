package controller

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	. "github.com/wailsapp/wails/v2/pkg/runtime"
)

// System 系统API
type System struct {
	Base
}

func NewSystem() *System {
	return &System{}
}

// BrowserOpenURL 默认浏览器打开网址
func (s *System) BrowserOpenURL(url string) {
	BrowserOpenURL(s.ctx, url)
}

// GetOpenDir 获取选择的目录路径
func (s System) GetOpenDir() string {
	path, _ := OpenDirectoryDialog(s.ctx, OpenDialogOptions{})
	return path
}

// GetOpenDir 获取选择的目录路径
func (s System) GetOpenFilePath() string {
	path, _ := OpenFileDialog(s.ctx, OpenDialogOptions{})
	return path
}

// ClipboardGetText 获取剪切板内容
func (s *System) ClipboardGetText() (string, error) {
	return ClipboardGetText(s.ctx)
}

// ClipboardSetText 设置剪切板内容
func (s *System) ClipboardSetText(text string) error {
	return ClipboardSetText(s.ctx, text)
}

// ShellCMD 以shell方式运行cmd命令
func (s *System) ShellCMD(cmdStr string, paramStr string, terminal int) {
	s.shellCMD(cmdStr, paramStr, terminal)
}

// OpenConfigDir 打开应用配置目录
func (s *System) OpenConfigDir() {
	s.openDir(s.getAppPath())
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
