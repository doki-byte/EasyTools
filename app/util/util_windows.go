//go:build windows

package util

import (
	extractor "EasyTools/app/embed"
	"fmt"
	"golang.org/x/sys/windows/registry"
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
	var command string
	if cmdPath == "" {
		command = fmt.Sprintf("%s %s", cmdStr, paramStr)
	} else {
		command = fmt.Sprintf("cd /D %s && %s %s", cmdPath, cmdStr, paramStr)
	}

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

// SetAutoStart 设置开机自启动
func (u *Util) SetAutoStart(enabled bool) bool {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		u.Log("获取可执行文件路径失败: " + err.Error())
		return false
	}

	// 转换为绝对路径
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		u.Log("获取绝对路径失败: " + err.Error())
		return false
	}

	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		u.Log("打开注册表键失败: " + err.Error())
		return false
	}
	defer key.Close()

	appName := "EasyTools" // 注册表项名称

	if enabled {
		// 添加开机自启动
		// 使用引号包裹路径，防止路径中有空格的问题
		cmd := fmt.Sprintf(`"%s"`, exePath)
		err = key.SetStringValue(appName, cmd)
		if err != nil {
			u.Log("设置注册表值失败: " + err.Error())
			return false
		}
		u.Log("开机自启动已启用: " + cmd)
	} else {
		// 删除开机自启动
		err = key.DeleteValue(appName)
		if err != nil {
			// 如果值不存在，也不认为是错误
			if err != registry.ErrNotExist {
				u.Log("删除注册表值失败: " + err.Error())
				return false
			}
		}
		u.Log("开机自启动已禁用")
	}

	return true
}

// GetAutoStart 获取开机自启动状态
func (u *Util) GetAutoStart() bool {
	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE)
	if err != nil {
		u.Log("打开注册表键失败: " + err.Error())
		return false
	}
	defer key.Close()

	appName := "EasyTools" // 注册表项名称

	// 检查值是否存在
	_, _, err = key.GetStringValue(appName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false
		}
		u.Log("读取注册表值失败: " + err.Error())
		return false
	}

	return true
}
