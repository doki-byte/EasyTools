package proxy

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	runtime2 "runtime"
)

// 获取应用基础目录
func GetAppBaseDir() string {
	// 如果是 macOS，使用应用支持目录
	if runtime2.GOOS == "darwin" {
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

func PathExist(path string) string {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(path, os.ModePerm)
	}
	return path
}

func Db() *gorm.DB {
	//打开数据库，如果不存在，则创建
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s\\config.db", GetAppBaseDir())), &gorm.Config{})
	if err != nil {
		fmt.Println("数据库连接失败")
	}
	// u.Log("数据库连接成功")
	return db
}
