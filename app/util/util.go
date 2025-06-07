//go:build linux || darwin

package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"EasyTools/app/unwxapp"

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
		panic(any("数据库连接失败"))
	}
	return db
}

// 初始化免杀模块
func (u *Util) InitMianSha() *Util {
	_ = u.PathExist(fmt.Sprintf("%s/icon", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s/tools", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s/file", u.GetAppPath()))

	return u
}

func (u *Util) InitFile() *Util {
	_ = u.PathExist(fmt.Sprintf("%s/icon", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s/tools", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s/file", u.GetAppPath()))

	targetUnwxappDir := filepath.Join("EasyToolsFiles", "tools", "Unwxapp")
	if _, err := os.Stat(targetUnwxappDir); os.IsNotExist(err) {
		// 如果目标目录不存在，执行资源解压
		fmt.Println("目标文件夹不存在，正在解压资源...")
		err := unwxapp.ExtractAllResources() // 调用解压逻辑
		if err != nil {
			log.Printf("Unwxapp解压资源失败: %w", err)
		}
		fmt.Println("Unwxapp资源解压完成")
	} else {
		// 如果目标目录已存在
		fmt.Println("Unwxapp资源文件夹已存在，跳过解压。")
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
	// 获取系统当前目录
	currentPath, err := os.Getwd()
	if err != nil {
		panic(any("获取当前路径失败"))
	}
	// 获取我的文档目录
	return u.PathExist(fmt.Sprintf("%s/EasyToolsFiles", currentPath))
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
func (u *Util) ShellCMD(cmdStr string, paramStr string, terminal int) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "darwin" {
			shellPath = "/bin/zsh"
		} else {
			shellPath = "/bin/bash"
		}
	}
	shellName := filepath.Base(shellPath)

	fullCommand := fmt.Sprintf("%s %s", cmdStr, paramStr)
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

// OpenDir 打开目录
func (u *Util) OpenDir(path string, terminal int) {
	if runtime.GOOS == "darwin" {
		u.ShellCMD("open", path, terminal)
		return
	}
	if !regexp.MustCompile(`^(http|ftp)s?://`).MatchString(path) {
		path = fmt.Sprintf("file://%s", path)
	}
	u.ShellCMD("xdg-open", path, terminal)
}

// PathConvert 路径转换
func (u *Util) PathConvert(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}
