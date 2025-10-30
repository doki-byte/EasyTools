//go:build linux || darwin

package util

import (
	extractor "EasyTools/app/embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Util struct{}

func NewUtil() *Util {
	return &Util{}
}

// Db 获取数据库操作对象和数据库初始化
func (u *Util) Db() *gorm.DB {
	// 打开数据库，如果不存在，则创建
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/config.db", u.GetAppPath())), &gorm.Config{})
	if err != nil {
		u.Log("数据库连接失败")
	}
	return db
}

func (u *Util) InitFile() *Util {
	baseDir := extractor.GetExtractor().GetAppBaseDir()
	_ = u.PathExist(filepath.Join(baseDir, "icon"))
	_ = u.PathExist(filepath.Join(baseDir, "tools"))
	_ = u.PathExist(filepath.Join(baseDir, "file"))
	return u
}

// 初始化免杀模块
func (u *Util) InitMianSha() *Util {
	// 先初始化目录结构
	u.InitFile()

	// 初始化解压器配置
	extractor.InitExtractor()

	// 解压所有资源
	if err := extractor.GetExtractor().ExtractAll(); err != nil {
		log.Printf("资源解压失败: %v", err)
	}

	return u
}

// Log 增加日志记录
func (u *Util) Log(content string) *Util {
	path := u.PathExist(fmt.Sprintf("%s/logs", u.GetAppPath()))

	// 创建或打开日志文件
	logFile, err := os.OpenFile(fmt.Sprintf("%s/%s.log", path, time.Now().Format("2006-01-02")), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	// 记录文件路径和行号
	_, file, line, _ := runtime.Caller(1)
	// 初始化日志
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Println(fmt.Sprintf("\n文件路径：%s:%d\n日志内容：%s\n", file, line, content))
	return u
}

// Schema 根据model自动建立数据表
func (u *Util) Schema(dst ...interface{}) {
	db := u.Db()

	for _, model := range dst {
		// 自动创建表
		if err := db.AutoMigrate(model); err != nil {
			log.Fatalf("Error creating table for %T: %v", model, err)
		}

		// 检查模型是否实现了 Initializer 接口
		if initializer, ok := model.(interface{ Initialize(*gorm.DB) }); ok {
			initializer.Initialize(db)
		} else {
			log.Printf("No default data initializer for model: %T", model)
		}
	}
}

// GetAppPath 获取应用主目录
func (u *Util) GetAppPath() string {

	// 如果是 macOS，使用应用支持目录
	if runtime.GOOS == "darwin" {
		appName := "EasyTools" // 你的应用名称

		// 获取应用支持目录
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("获取用户主目录失败: " + err.Error())
		}

		// macOS 的应用支持目录是 ~/Library/Application Support/
		appSupportDir := filepath.Join(homeDir, "Library", "Application Support", appName)

		// 确保目录存在
		return u.PathExist(appSupportDir)
	}

	// 其他系统使用当前目录下的 EasyToolsFiles
	currentPath, err := os.Getwd()
	if err != nil {
		panic("获取当前路径失败: " + err.Error())
	}

	return u.PathExist(filepath.Join(currentPath, "EasyToolsFiles"))
}

// PathExist 判断文件目录是否存在，不存在创建
func (u *Util) PathExist(path string) string {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(path, os.ModePerm)
	}
	return path
}

// ShellCMD 跨平台执行 shell 命令
func (u *Util) ShellCMD(cmdPath, cmdStr, paramStr string, terminal int) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "darwin" {
			shellPath = "/bin/zsh"
		} else {
			shellPath = "/bin/bash"
		}
	}
	shellName := filepath.Base(shellPath)

	// 拼接完整命令：先切换到指定目录，然后执行命令
	var fullCommand string
	if cmdPath != "" {
		fullCommand = fmt.Sprintf("cd %s && %s %s", cmdPath, cmdStr, paramStr)
	} else {
		fullCommand = fmt.Sprintf("%s %s", cmdStr, paramStr)
	}

	fmt.Printf("执行命令: %s\n", fullCommand)

	var cmd *exec.Cmd
	if terminal == 1 {
		terminalApp, terminalArgs := getTerminalInfo(shellPath, shellName, fullCommand)
		if terminalApp == "" {
			fmt.Println("错误：未找到支持的终端程序")
			return
		}
		cmd = exec.Command(terminalApp, terminalArgs...)
	} else {
		cmd = exec.Command(shellPath, "-c", fullCommand)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("命令执行错误: %v\n", err)
	}
}

// macOS 终端处理（优化版）
func getMacTerminal(shellName, command string) (string, []string) {
	// 对命令进行转义处理
	escapedCmd := fmt.Sprintf(
		`unset HISTFILE 2>/dev/null; mkdir -p ~/.zsh_sessions 2>/dev/null; echo "执行结果: $(%s)"; exec zsh`,
		strings.Replace(command, `"`, `\"`, -1),
	)

	// 合并所有操作为单条命令
	mergedCmd := fmt.Sprintf(`zsh -ic '%s'`, escapedCmd)

	// 对完整命令进行AppleScript转义
	appleScriptCmd := strconv.Quote(mergedCmd)

	// 默认使用系统终端
	script := fmt.Sprintf(`
    tell application "Terminal"
        activate
        do script %s
    end tell
    `, appleScriptCmd)

	// 检查是否安装iTerm
	if _, err := os.Stat("/Applications/iTerm.app"); err == nil {
		script = fmt.Sprintf(`
        tell application "iTerm"
            activate
            create window with default profile
            tell current session of current window
                write text %s
            end tell
        end tell
        `, appleScriptCmd)
	}

	return "osascript", []string{"-e", strings.TrimSpace(script)}
}

// 跨平台终端检测
func getTerminalInfo(shellPath, shellName, command string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return getMacTerminal(shellName, command)
	case "linux":
		return getLinuxTerminal(shellPath, shellName, command)
	default:
		return "", nil
	}
}

// Linux 终端处理
func getLinuxTerminal(shellPath, shellName, command string) (string, []string) {
	terminals := []struct {
		name string
		test func() bool
		args []string
	}{
		{
			"gnome-terminal",
			func() bool { return exec.Command("which", "gnome-terminal").Run() == nil },
			[]string{
				"--",
				shellPath,
				"-c",
				fmt.Sprintf("echo \"执行命令: %s\" && %s; exec %s", command, command, shellName),
			},
		},
		{
			"konsole",
			func() bool { return exec.Command("which", "konsole").Run() == nil },
			[]string{
				"-e",
				"bash",
				"-c",
				fmt.Sprintf("echo \"执行命令: %s\" && %s; exec %s", command, command, shellName),
			},
		},
		{
			"xfce4-terminal",
			func() bool { return exec.Command("which", "xfce4-terminal").Run() == nil },
			[]string{
				"-x",
				shellPath,
				"-c",
				fmt.Sprintf("echo \"执行命令: %s\" && %s; exec %s", command, command, shellName),
			},
		},
		{
			"xterm",
			func() bool { return exec.Command("which", "xterm").Run() == nil },
			[]string{
				"-e",
				shellPath,
				"-c",
				fmt.Sprintf("echo \"执行命令: %s\" && %s; exec %s", command, command, shellName),
			},
		},
	}

	for _, term := range terminals {
		if term.test() {
			return term.name, term.args
		}
	}

	return "", nil
}

// OpenDir 打开指定目录
func (u *Util) OpenDir(path string, terminal int) {
	if terminal == 1 {
		// 如果需要终端，使用文件管理器打开目录
		if runtime.GOOS == "darwin" {
			u.ShellCMD("", "open", path, 0)
		} else {
			// Linux 使用 xdg-open 或特定文件管理器
			u.ShellCMD("", "xdg-open", path, 0)
		}
	} else {
		// 直接使用系统默认方式打开目录
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("open", path)
		} else {
			cmd = exec.Command("xdg-open", path)
		}
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error opening directory: %v\n", err)
		}
	}
}

// PathConvert 路径转换
func (u *Util) PathConvert(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}

// SetAutoStart 设置开机自启动
func (u *Util) SetAutoStart(enabled bool) bool {
	return false
}

// GetAutoStart 获取开机自启动状态
func (u *Util) GetAutoStart() bool {
	return false
}
