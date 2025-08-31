//go:build windows

package util

import (
	cyberchef "EasyTools/app/embedCyberChef"
	"EasyTools/app/note"
	"EasyTools/app/unwxapp"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	moduser32                 = windows.NewLazySystemDLL("user32.dll")
	procSystemParametersInfoW = moduser32.NewProc("SystemParametersInfoW")
)

type Util struct{}

func NewUtil() *Util {
	return &Util{}
}

// Db 获取数据库操作对象和数据库初始化
func (u *Util) Db() *gorm.DB {
	//打开数据库，如果不存在，则创建
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s\\config.db", u.GetAppPath())), &gorm.Config{})
	if err != nil {
		u.Log("数据库连接失败")
	}
	// u.Log("数据库连接成功")
	return db
}

func (u *Util) InitFile() *Util {
	_ = u.PathExist(fmt.Sprintf("%s\\icon", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s\\tools", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s\\file", u.GetAppPath()))
	return u
}

// 初始化免杀模块
func (u *Util) InitMianSha() *Util {
	_ = u.PathExist(fmt.Sprintf("%s\\icon", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s\\tools", u.GetAppPath()))
	_ = u.PathExist(fmt.Sprintf("%s\\file", u.GetAppPath()))

	// 定义目标解压目录
	targetNoteDir := filepath.Join(u.GetAppPath(), "notes")
	if _, err := os.Stat(targetNoteDir); os.IsNotExist(err) {
		// 如果目标目录不存在，执行资源解压
		fmt.Println("目标文件夹不存在，正在解压资源...")
		err := note.ExtractNoteFile() // 调用解压逻辑
		if err != nil {
			log.Printf("notes解压资源失败: %w", err)
		}
		fmt.Println("notes资源解压完成")
	} else {
		// 如果目标目录已存在
		fmt.Println("notes资源文件夹已存在，跳过解压。")
	}

	targetUnwxappDir := filepath.Join(u.GetAppPath(), "tools", "Unwxapp")
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
	targetCyberChefDir := filepath.Join(u.GetAppPath(), "CyberChef")
	if _, err := os.Stat(targetCyberChefDir); os.IsNotExist(err) {
		// 如果目标目录不存在，执行资源解压
		fmt.Println("目标文件夹不存在，正在解压资源...")
		err := cyberchef.ExtractAllResources() // 调用解压逻辑
		if err != nil {
			log.Printf("CyberChef解压资源失败: %w", err)
		}
		fmt.Println("CyberChef资源解压完成")
	} else {
		// 如果目标目录已存在
		fmt.Println("CyberChef资源文件夹已存在，跳过解压。")
	}

	return u
}

// Log 增加日志记录
func (u *Util) Log(content string) *Util {
	path := u.PathExist(fmt.Sprintf("%s\\logs", u.GetAppPath()))

	// 创建或打开日志文件
	logFile, err := os.OpenFile(fmt.Sprintf("%s\\%s.log", path, time.Now().Format("2006-01-02")), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	//记录文件路径和行号
	_, file, line, _ := runtime.Caller(1)
	// 初始化日志
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Println(fmt.Sprintf("\n文件路径：%s:%d\n日志内容：%s\n", file, line, content))
	return u
}

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

// ShellCMD 以shell方式运行cmd命令，支持终端和无终端两种模式
func (u *Util) ShellCMD(cmdPath, cmdStr, paramStr string, terminal int) {
	// 拼接完整命令
	command := fmt.Sprintf("cd /D %s && %s %s", cmdPath, cmdStr, paramStr)

	if terminal == 1 {
		// 需要终端，保持终端打开
		echocmd := fmt.Sprintf(`echo "使用命令: %s"`, command)
		cmd := exec.Command("cmd", "/K", echocmd+"&&"+command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error executing command: %v\n", err)
		}
	} else {
		// 使用 cmd /C 直接执行命令
		cmd := exec.Command("cmd", "/C", command)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // 隐藏窗口
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error executing command without terminal: %v\n", err)
		}
	}
}

// OpenDir 打开指定目录
func (u *Util) OpenDir(path string, terminal int) {
	cmd := exec.Command("explorer", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error opening directory: %v\n", err)
	}
}

// PathConvert 路径转换
func (u *Util) PathConvert(path string) string {
	return path
}
